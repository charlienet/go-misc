package crypto

import (
    "crypto"
    "crypto/ecdh"
    "crypto/ecdsa"
    "crypto/rand"
    "errors"
    "fmt"
    
    "github.com/emmansun/gmsm/sm2"
)

// KeyAgreement 密钥协商接口
type KeyAgreement interface {
    GenerateKey() (*KeyPair, error)
    DeriveSharedSecret(peerPublicKey crypto.PublicKey) ([]byte, error)
    Name() string
}

// 密钥协商算法注册表
var keyAgreementAlgorithms = map[string]func() (KeyAgreement, error){
    "ECDH":   newECDH,
    "X25519": newX25519,
    "SM2":    newSM2KeyAgreement,
}

// NewKeyAgreement 创建密钥协商器
func NewKeyAgreement(algorithm string) (KeyAgreement, error) {
    normalizedAlg, err := NormalizeAlgorithm(algorithm)
    if err != nil {
        return nil, fmt.Errorf("invalid algorithm: %w", err)
    }
    creator, ok := keyAgreementAlgorithms[normalizedAlg]
    if !ok {
        return nil, fmt.Errorf("unsupported key agreement algorithm: %s", algorithm)
    }
    return creator()
}

// --- ECDH 实现 ---

type ecdhKA struct {
    privateKey *ecdh.PrivateKey
    curve      ecdh.Curve
}

func newECDH() (KeyAgreement, error) {
    return &ecdhKA{curve: ecdh.P256()}, nil
}

func (k *ecdhKA) Name() string {
    return "ECDH"
}

func (k *ecdhKA) GenerateKey() (*KeyPair, error) {
    priv, err := k.curve.GenerateKey(rand.Reader)
    if err != nil {
        return nil, err
    }
    k.privateKey = priv
    return &KeyPair{
        PrivateKey: priv,
        PublicKey:  priv.PublicKey(),
    }, nil
}

func (k *ecdhKA) DeriveSharedSecret(peerPublicKey crypto.PublicKey) ([]byte, error) {
    if k.privateKey == nil {
        return nil, errors.New("private key not set")
    }
    
    ecdhPub, ok := peerPublicKey.(*ecdh.PublicKey)
    if !ok {
        return nil, errors.New("invalid public key type for ECDH")
    }
    
    return k.privateKey.ECDH(ecdhPub)
}

// --- X25519 实现 ---

type x25519KA struct {
    privateKey *ecdh.PrivateKey
}

func newX25519() (KeyAgreement, error) {
    return &x25519KA{}, nil
}

func (k *x25519KA) Name() string {
    return "X25519"
}

func (k *x25519KA) GenerateKey() (*KeyPair, error) {
    priv, err := ecdh.X25519().GenerateKey(rand.Reader)
    if err != nil {
        return nil, err
    }
    k.privateKey = priv
    return &KeyPair{
        PrivateKey: priv,
        PublicKey:  priv.PublicKey(),
    }, nil
}

func (k *x25519KA) DeriveSharedSecret(peerPublicKey crypto.PublicKey) ([]byte, error) {
    if k.privateKey == nil {
        return nil, errors.New("private key not set")
    }
    
    ecdhPub, ok := peerPublicKey.(*ecdh.PublicKey)
    if !ok {
        return nil, errors.New("invalid public key type for X25519")
    }
    
    return k.privateKey.ECDH(ecdhPub)
}

// --- SM2 密钥协商实现 ---

type sm2KA struct {
    privateKey *sm2.PrivateKey
}

func newSM2KeyAgreement() (KeyAgreement, error) {
    return &sm2KA{}, nil
}

func (k *sm2KA) Name() string {
    return "SM2"
}

func (k *sm2KA) GenerateKey() (*KeyPair, error) {
    priv, err := sm2.GenerateKey(rand.Reader)
    if err != nil {
        return nil, err
    }
    k.privateKey = priv
    return &KeyPair{
        PrivateKey: priv,
        PublicKey:  &priv.PublicKey,
    }, nil
}

func (k *sm2KA) DeriveSharedSecret(peerPublicKey crypto.PublicKey) ([]byte, error) {
    if k.privateKey == nil {
        return nil, errors.New("private key not set")
    }
    
    // 检查是否为ecdsa公钥并验证是否是SM2密钥
    ecdsaPub, isEcdsa := peerPublicKey.(*ecdsa.PublicKey)
    if !isEcdsa || !sm2.IsSM2PublicKey(ecdsaPub) {
        return nil, errors.New("invalid public key type for SM2")
    }
    
    // 使用SM2曲线进行密钥协商
    // 在实际SM2密钥协商协议中，还需要额外的参数和步骤
    // 为简化，这里使用椭圆曲线上的标量乘法
    x, y := k.privateKey.Curve.ScalarMult(ecdsaPub.X, ecdsaPub.Y, k.privateKey.D.Bytes())
    sharedSecret := append(x.Bytes(), y.Bytes()...)
    return sharedSecret, nil
}