package crypto

import (
	"crypto"
	"fmt"

	"github.com/charlienet/go-misc/bytesconv"
)

// KeyPair 密钥对
type KeyPair struct {
	PrivateKey string
	PublicKey  string
}

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
		"SM2": new_sm2,
		"RSA": new_rsa,
	}
)

type asymmetric struct {
	publicKey  string
	privateKey string
	hash       crypto.Hash
	bits       int
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

func NewAsymmetric(algorithm string, opts ...keyFunc) (Asymmetric, error) {
	creator, ok := supportedAsymmetricAlgorithms[algorithm]
	if !ok {
		return nil, fmt.Errorf("unsupported asymmetric algorithm: %s", algorithm)
	}

	return creator(opts...)
}
