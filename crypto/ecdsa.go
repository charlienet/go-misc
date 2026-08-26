package crypto

import (
    "crypto"
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/x509"
    "encoding/asn1"
    "encoding/base64"
    "errors"
    "math/big"
    
    "github.com/charlienet/go-misc/bytesconv"
)

type ecdsa_algo struct {
    prk  *ecdsa.PrivateKey
    puk  *ecdsa.PublicKey
    hash crypto.Hash
}

func new_ecdsa(opts ...keyFunc) (Asymmetric, error) {
    key := asymmetric{
        hash: crypto.SHA256,
    }
    
    for _, opt := range opts {
        if err := opt(&key); err != nil {
            return nil, err
        }
    }
    
    algo := &ecdsa_algo{
        hash: key.hash,
    }
    
    if key.privateKeyObject != nil {
        ecdsaKey, ok := key.privateKeyObject.(*ecdsa.PrivateKey)
        if !ok {
            return nil, errors.New("not an ECDSA private key")
        }
        algo.prk = ecdsaKey
    }
    
    if key.publicKeyObject != nil {
        ecdsaKey, ok := key.publicKeyObject.(*ecdsa.PublicKey)
        if !ok {
            return nil, errors.New("not an ECDSA public key")
        }
        algo.puk = ecdsaKey
    }
    
    return algo, nil
}

func (s *ecdsa_algo) Name() string {
    return "ECDSA"
}

func (s *ecdsa_algo) GenerateKey() (KeyPair, error) {
    // 使用 P256 曲线
    key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return KeyPair{}, err
    }
    s.prk = key
    return KeyPair{PrivateKey: key, PublicKey: &key.PublicKey}, nil
}

func (s *ecdsa_algo) WithPrivateKey(privateKey string) error {
    return errors.New("ECDSA WithPrivateKey(string) not implemented, use WithPrivateKeyObject")
}

func (s *ecdsa_algo) WithPublicKey(publicKey string) error {
    return errors.New("ECDSA WithPublicKey(string) not implemented, use WithPublicKeyObject")
}

func (s *ecdsa_algo) ExportPublicKey() (string, error) {
    if s.prk == nil && s.puk == nil {
        return "", errors.New("no key set")
    }
    
    pub := s.puk
    if pub == nil {
        pub = &s.prk.PublicKey
    }
    
    der, err := x509.MarshalPKIXPublicKey(pub)
    if err != nil {
        return "", err
    }
    
    return base64Encode(der), nil
}

func (s *ecdsa_algo) Encrypt(msg []byte) (bytesconv.BytesResult, error) {
    return nil, errors.New("ECDSA does not support encryption")
}

func (s *ecdsa_algo) Decrypt(ciphertext []byte) (bytesconv.BytesResult, error) {
    return nil, errors.New("ECDSA does not support decryption")
}

func (s *ecdsa_algo) Sign(data []byte) (bytesconv.BytesResult, error) {
    if s.prk == nil {
        return nil, errors.New("ECDSA private key not set")
    }
    
    h := s.hash.New()
    h.Write(data)
    hashed := h.Sum(nil)
    
    r, s_val, err := ecdsa.Sign(rand.Reader, s.prk, hashed)
    if err != nil {
        return nil, err
    }
    
    // 返回 ASN.1 DER 编码的签名
    type ecdsaSignature struct {
        R, S *big.Int
    }
    sig := ecdsaSignature{R: r, S: s_val}
    return asn1.Marshal(sig)
}

func (s *ecdsa_algo) Verify(data, signature []byte) bool {
    if s.puk == nil {
        return false
    }
    
    h := s.hash.New()
    h.Write(data)
    hashed := h.Sum(nil)
    
    // 解析 ASN.1 DER 签名
    type ecdsaSignature struct {
        R, S *big.Int
    }
    var sig ecdsaSignature
    _, err := asn1.Unmarshal(signature, &sig)
    if err != nil {
        return false
    }
    
    return ecdsa.Verify(s.puk, hashed, sig.R, sig.S)
}

// base64Encode 是一个辅助函数，用于编码为base64字符串
func base64Encode(data []byte) string {
    return base64.StdEncoding.EncodeToString(data)
}