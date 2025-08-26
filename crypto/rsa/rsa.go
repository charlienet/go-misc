package rsa

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strconv"

	"github.com/charlienet/go-misc/bytesconv"
	gcrypto "github.com/charlienet/go-misc/crypto"
)

const (
	defaultRsaBits = 1024
)

var _ gcrypto.IAsymmetric = &rsaInstance{}

type rsaInstance struct {
	gcrypto.HashOptions
	prk *rsa.PrivateKey
	puk *rsa.PublicKey
}

type rsaOption func(o *rsaInstance) error

func New(h crypto.Hash, opts ...rsaOption) (*rsaInstance, error) {
	o := &rsaInstance{}

	sh := crypto.Hash(h)
	if !sh.Available() {
		return nil, errors.New("unknown hash value " + strconv.Itoa(int(h)))
	}

	o.H = sh

	for _, f := range opts {
		if err := f(o); err != nil {
			return nil, err
		}
	}

	// 未设置私钥时随机生成密钥
	if o.prk == nil {
		prk, err := rsa.GenerateKey(rand.Reader, defaultRsaBits)
		if err != nil {
			return nil, err
		}

		o.prk = prk
	}

	// 公钥未设置时从私钥导出
	if o.puk == nil {
		o.puk = &o.prk.PublicKey
	}

	return o, nil
}

var pemStart = []byte("-----BEGIN ")

func ParsePKCS8PrivateKey(pri []byte) rsaOption {
	return func(o *rsaInstance) error {
		if bytes.HasPrefix(pri, pemStart) {

			block, _ := pem.Decode(pri)
			if block == nil {
				return errors.New("failed to decode private key")
			}

			pri = block.Bytes
		}

		prk, err := x509.ParsePKCS8PrivateKey(pri)
		if err != nil {
			return err
		}

		o.prk = prk.(*rsa.PrivateKey)

		return nil
	}
}

func ParsePKCS1PrivateKey(pri []byte) rsaOption {
	return func(o *rsaInstance) error {
		if bytes.HasPrefix(pri, pemStart) {
			block, _ := pem.Decode(pri)
			if block == nil {
				return errors.New("failed to decode private key")
			}
			pri = block.Bytes
		}
		prk, err := x509.ParsePKCS1PrivateKey(pri)
		if err != nil {
			return err
		}

		o.prk = prk

		return nil
	}
}

func ParsePKIXPublicKey(pub []byte) rsaOption {
	return func(o *rsaInstance) error {
		if bytes.HasPrefix(pub, pemStart) {
			block, _ := pem.Decode(pub)
			if block == nil {
				return errors.New("failed to decode public key")
			}

			pub = block.Bytes
		}

		k, err := x509.ParsePKIXPublicKey(pub)
		if err != nil {
			return err
		}

		puk := k.(*rsa.PublicKey)

		o.puk = puk

		return nil
	}
}

func (o *rsaInstance) Encrypt(msg []byte) (bytesconv.BytesResult, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, o.puk, msg)
}

func (o *rsaInstance) Decrypt(ciphertext []byte) (bytesconv.BytesResult, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, o.prk, ciphertext)
}

func (o *rsaInstance) Sign(msg []byte) (bytesconv.BytesResult, error) {
	hashed := o.GetHash(msg)
	sign, err := rsa.SignPKCS1v15(rand.Reader, o.prk, o.H, hashed)
	return sign, err
}

func (o *rsaInstance) Verify(msg, sign []byte) bool {
	hashed := o.GetHash(msg)
	if err := rsa.VerifyPKCS1v15(o.puk, o.H, hashed, sign); err != nil {
		return false
	}

	return true
}
