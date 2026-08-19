package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/charlienet/go-misc/bytesconv"
)

type rsa_algo struct {
	prk  *rsa.PrivateKey
	puk  *rsa.PublicKey
	hash crypto.Hash
	bits int
}

func new_rsa(opts ...keyFunc) (Asymmetric, error) {
	key := asymmetric{
		hash: crypto.SHA256,
		bits: 2048,
	}

	for _, opt := range opts {
		opt(&key)
	}

	algo := &rsa_algo{
		hash: key.hash,
		bits: key.bits,
	}

	if key.privateKey != "" {
		if err := algo.WithPrivateKey(key.privateKey); err != nil {
			return nil, err
		}
	}

	if key.publicKey != "" {
		if err := algo.WithPublicKey(key.publicKey); err != nil {
			return nil, err
		}
	}

	return algo, nil
}

func (s *rsa_algo) Name() string {
	return "RSA"
}

func (s *rsa_algo) GenerateKey() (KeyPair, error) {
	key, err := rsa.GenerateKey(rand.Reader, s.bits)
	if err != nil {
		return KeyPair{}, err
	}

	s.prk = key

	prk, err := x509.MarshalPKCS8PrivateKey(s.prk)
	if err != nil {
		return KeyPair{}, err
	}

	pub, err := x509.MarshalPKIXPublicKey(&s.prk.PublicKey)
	if err != nil {
		return KeyPair{}, err
	}

	return KeyPair{
		PrivateKey: base64.StdEncoding.EncodeToString(prk),
		PublicKey:  base64.StdEncoding.EncodeToString(pub),
	}, nil
}

func (s *rsa_algo) WithPrivateKey(privateKey string) error {
	prkBytes, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return err
	}

	// 先尝试 PKCS1 格式
	prk, err := x509.ParsePKCS1PrivateKey(prkBytes)
	if err == nil {
		s.prk = prk
	} else {
		// 再尝试 PKCS8 格式
		key, err := x509.ParsePKCS8PrivateKey(prkBytes)
		if err != nil {
			return err
		}

		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return errors.New("not an RSA private key")
		}

		s.prk = rsaKey
	}

	// 弱密钥校验：RSA 长度低于 2048-bit 视为不安全（如 512/1024-bit 已被视为可破解），直接拒绝
	if s.prk.N.BitLen() < 2048 {
		bits := s.prk.N.BitLen()
		s.prk = nil
		return fmt.Errorf("RSA private key too weak: %d bits, minimum required is 2048 bits", bits)
	}
	return nil
}

func (s *rsa_algo) WithPublicKey(publicKey string) error {
	pubBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return err
	}

	k, err := x509.ParsePKIXPublicKey(pubBytes)
	if err != nil {
		return err
	}

	puk, ok := k.(*rsa.PublicKey)
	if !ok {
		return errors.New("not an RSA public key")
	}

	// 弱密钥校验：与私钥策略对称，公钥低于 2048-bit 同样拒绝
	if puk.N.BitLen() < 2048 {
		bits := puk.N.BitLen()
		return fmt.Errorf("RSA public key too weak: %d bits, minimum required is 2048 bits", bits)
	}

	s.puk = puk
	return nil
}

func (s *rsa_algo) ExportPublicKey() (string, error) {
	if s.prk == nil {
		return "", errors.New("RSA private key not set")
	}

	pub, err := x509.MarshalPKIXPublicKey(&s.prk.PublicKey)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(pub), nil
}

var label = []byte("")

func (r *rsa_algo) Encrypt(msg []byte) (bytesconv.BytesResult, error) {
	if r.puk == nil {
		return nil, errors.New("RSA public key not set")
	}

	cipher, err := rsa.EncryptOAEP(r.hash.New(), rand.Reader, r.puk, msg, label)
	return cipher, err
}

func (r *rsa_algo) Decrypt(msg []byte) (bytesconv.BytesResult, error) {
	if r.prk == nil {
		return nil, errors.New("RSA private key not set")
	}

	plain, err := rsa.DecryptOAEP(r.hash.New(), rand.Reader, r.prk, msg, label)
	return plain, err
}

func (r *rsa_algo) Sign(data []byte) (bytesconv.BytesResult, error) {
	if r.prk == nil {
		return nil, errors.New("RSA private key not set")
	}

	h := r.hash.New()
	h.Write(data)
	hashed := h.Sum(nil)

	signature, err := rsa.SignPSS(rand.Reader, r.prk, r.hash, hashed, nil)
	return signature, err
}

func (r *rsa_algo) Verify(data, signature []byte) bool {
	if r.puk == nil {
		return false
	}

	h := r.hash.New()
	h.Write(data)
	hashed := h.Sum(nil)

	err := rsa.VerifyPSS(r.puk, r.hash, hashed, signature, nil)
	return err == nil
}
