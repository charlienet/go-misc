package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"

	"github.com/charlienet/go-misc/bytesconv"
	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/x509"
)

type sm2_algo struct {
	prk *sm2.PrivateKey
	puk *sm2.PublicKey
}

func new_sm2(opts ...keyFunc) (Asymmetric, error) {
	key := asymmetric{}
	for _, opt := range opts {
		opt(&key)
	}

	s := &sm2_algo{}

	if key.privateKey != "" {
		if err := s.WithPrivateKey(key.privateKey); err != nil {
			return nil, err
		}
	}

	if key.publicKey != "" {
		if err := s.WithPublicKey(key.publicKey); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *sm2_algo) GenerateKey() (KeyPair, error) {
	prv, err := sm2.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, err
	}

	s.prk = prv
	s.puk = &s.prk.PublicKey

	privPem, err := x509.WritePrivateKeyToPem(s.prk, nil)
	if err != nil {
		return KeyPair{}, err
	}

	pubkeyPem, err := x509.WritePublicKeyToPem(s.puk)
	if err != nil {
		return KeyPair{}, err
	}

	prvBlock, _ := pem.Decode(privPem)
	pubBlock, _ := pem.Decode(pubkeyPem)

	return KeyPair{
		PrivateKey: base64.StdEncoding.EncodeToString(prvBlock.Bytes),
		PublicKey:  base64.StdEncoding.EncodeToString(pubBlock.Bytes),
	}, nil
}

func (s *sm2_algo) WithPrivateKey(key string) error {
	der, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}

	s.prk, err = x509.ParsePKCS8PrivateKey(der, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *sm2_algo) WithPublicKey(key string) error {
	pubBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}

	s.puk, err = x509.ParseSm2PublicKey(pubBytes)
	if err != nil {
		return err
	}

	return nil
}

// 导出公钥所对应的公钥
func (s *sm2_algo) ExportPublicKey() (string, error) {
	s.puk = &s.prk.PublicKey

	pubkeyPem, err := x509.WritePublicKeyToPem(s.puk)
	if err != nil {
		return "", err
	}

	pubBlock, _ := pem.Decode(pubkeyPem)

	return base64.StdEncoding.EncodeToString(pubBlock.Bytes), nil
}

func (s *sm2_algo) Encrypt(msg []byte) (bytesconv.BytesResult, error) {
	return s.puk.EncryptAsn1(msg, rand.Reader)
}

func (s *sm2_algo) Decrypt(ciphertext []byte) (bytesconv.BytesResult, error) {
	return s.prk.DecryptAsn1(ciphertext)
}

func (s *sm2_algo) Sign(msg []byte) (bytesconv.BytesResult, error) {
	b, err := s.prk.Sign(rand.Reader, msg, nil)
	return b, err
}

func (s *sm2_algo) Verify(msg, sign []byte) bool {
	return s.puk.Verify(msg, sign)
}
