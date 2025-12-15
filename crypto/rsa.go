package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"

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

	prk, err := x509.ParsePKCS1PrivateKey(prkBytes)
	if err != nil {

	}

	s.prk = prk
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

	s.puk = k.(*rsa.PublicKey)
	return nil
}

func (s *rsa_algo) ExportPublicKey() (string, error) {
	pub, err := x509.MarshalPKIXPublicKey(&s.prk.PublicKey)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(pub), nil
}

var label = []byte("")

func (r *rsa_algo) Encrypt(msg []byte) (bytesconv.BytesResult, error) {
	cipher, err := rsa.EncryptOAEP(r.hash.New(), rand.Reader, r.puk, msg, label)
	return cipher, err
}

func (r *rsa_algo) Decrypt(msg []byte) (bytesconv.BytesResult, error) {
	plain, err := rsa.DecryptOAEP(r.hash.New(), rand.Reader, r.prk, msg, label)
	return plain, err
}

func (r *rsa_algo) Sign(data []byte) (bytesconv.BytesResult, error) {
	hashed := r.hash.New().Sum(data)

	signature, err := rsa.SignPSS(rand.Reader, r.prk, r.hash, hashed[:], nil)
	return signature, err
}

func (r *rsa_algo) Verify(data, signature []byte) bool {
	hashed := r.hash.New().Sum(data)

	err := rsa.VerifyPSS(r.puk, r.hash, hashed[:], signature, nil)
	return err == nil
}
