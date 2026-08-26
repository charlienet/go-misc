package crypto

import (
	"crypto"
	"fmt"

	"github.com/charlienet/go-misc/bytesconv"
)

// KeyPair 已迁移到 keypair.go
// 此文件中的 LegacyKeyPair 保留向后兼容
// 非对称加密算法
type Asymmetric interface {
	GenerateKey() (KeyPair, error)
	WithPrivateKey(privateKey string) error
	WithPublicKey(publicKey string) error
	ExportPublicKey() (string, error)
	Name() string
	Encrypt(msg []byte) (bytesconv.BytesResult, error)
	Decrypt(ciphertext []byte) (bytesconv.BytesResult, error)
	Signer
}

type Signer interface {
	Sign(msg []byte) (bytesconv.BytesResult, error)
	Verify(msg, sign []byte) bool
}

var (
	supportedAsymmetricAlgorithms = map[string]func(opts ...keyFunc) (Asymmetric, error){
		"SM2":     new_sm2,
		"RSA":     new_rsa,
		"ECDSA":   new_ecdsa,
		"Ed25519": new_ed25519,
	}
)

type asymmetric struct {
	publicKey        string
	privateKey       string
	publicKeyObject  crypto.PublicKey
	privateKeyObject crypto.PrivateKey
	hash             crypto.Hash
	bits             int
}

type keyFunc func(*asymmetric) error

func WithPublicKey(publicKey string) keyFunc {
	return func(a *asymmetric) error {
		a.publicKey = publicKey
		return nil
	}
}

func WithPrivateKey(privateKey string) keyFunc {
	return func(a *asymmetric) error {
		a.privateKey = privateKey
		return nil
	}
}

// WithPrivateKeyObject 使用密钥对象创建非对称加密器
func WithPrivateKeyObject(key crypto.PrivateKey) keyFunc {
	return func(a *asymmetric) error {
		a.privateKeyObject = key
		return nil
	}
}

// WithPublicKeyObject 使用密钥对象创建非对称加密器
func WithPublicKeyObject(key crypto.PublicKey) keyFunc {
	return func(a *asymmetric) error {
		a.publicKeyObject = key
		return nil
	}
}

func NewAsymmetric(algorithm string, opts ...keyFunc) (Asymmetric, error) {
	creator, ok := supportedAsymmetricAlgorithms[algorithm]
	if !ok {
		return nil, fmt.Errorf("unsupported asymmetric algorithm: %s", algorithm)
	}

	return creator(opts...)
}
