package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/charlienet/go-misc/bytesconv"
	"github.com/emmansun/gmsm/sm2"
	"github.com/emmansun/gmsm/smx509"
)

type sm2_algo struct {
	prk *sm2.PrivateKey
	puk *ecdsa.PublicKey
}

func new_sm2(opts ...keyFunc) (Asymmetric, error) {
	key := asymmetric{}
	for _, opt := range opts {
		if err := opt(&key); err != nil {
			return nil, err
		}
	}

	s := &sm2_algo{}

	// 优先使用密钥对象
	if key.privateKeyObject != nil {
		// First try direct sm2.PrivateKey
		if sm2Key, ok := key.privateKeyObject.(*sm2.PrivateKey); ok {
			s.prk = sm2Key
		} else {
			// Check if it's an ecdsa.PrivateKey that is actually an SM2 key
			ecdsaKey, ok := key.privateKeyObject.(*ecdsa.PrivateKey)
			if !ok {
				return nil, errors.New("not an SM2 private key")
			}
			// Check if it's actually an SM2 key by checking the public key
			if !sm2.IsSM2PublicKey(&ecdsaKey.PublicKey) {
				return nil, errors.New("not an SM2 private key")
			}
			// We need to convert ecdsa.PrivateKey back to sm2.PrivateKey
			// Create a new sm2.PrivateKey and copy the ecdsa.PrivateKey data
			s.prk = &sm2.PrivateKey{
				PrivateKey: *ecdsaKey,
			}
		}
	} else if key.privateKey != "" {
		if err := s.WithPrivateKey(key.privateKey); err != nil {
			return nil, err
		}
	}

	if key.publicKeyObject != nil {
		// SM2 uses ecdsa.PublicKey internally
		ecdsaKey, ok := key.publicKeyObject.(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("not an SM2 public key")
		}
		// Check if it's actually an SM2 key
		if !sm2.IsSM2PublicKey(ecdsaKey) {
			return nil, errors.New("not an SM2 public key")
		}
		s.puk = ecdsaKey
	} else if key.publicKey != "" {
		if err := s.WithPublicKey(key.publicKey); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *sm2_algo) Name() string {
	return "SM2"
}

func (s *sm2_algo) GenerateKey() (KeyPair, error) {
	prv, err := sm2.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, err
	}

	s.prk = prv
	s.puk = &s.prk.PublicKey

	return KeyPair{
		PrivateKey: prv,
		PublicKey:  &prv.PublicKey,
	}, nil
}

func (s *sm2_algo) WithPrivateKey(key string) error {
	der, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}

	parsed, err := smx509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return err
	}

	var ok bool
	s.prk, ok = parsed.(*sm2.PrivateKey)
	if !ok {
		return errors.New("failed to assert SM2 private key type")
	}

	return nil
}

func (s *sm2_algo) WithPublicKey(key string) error {
	der, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}

	parsed, err := smx509.ParsePKIXPublicKey(der)
	if err != nil {
		return err
	}

	var ok bool
	s.puk, ok = parsed.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("failed to assert ECDSA public key type")
	}

	return nil
}

// 导出公钥所对应的公钥
func (s *sm2_algo) ExportPublicKey() (string, error) {
	if s.prk == nil {
		return "", errors.New("SM2 private key not set")
	}

	s.puk = &s.prk.PublicKey

	pubDER, err := smx509.MarshalPKIXPublicKey(s.puk)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(pubDER), nil
}

func (s *sm2_algo) Encrypt(msg []byte) (bytesconv.BytesResult, error) {
	if s.puk == nil {
		return nil, errors.New("SM2 public key not set")
	}

	return sm2.EncryptASN1(rand.Reader, s.puk, msg)
}

func (s *sm2_algo) Decrypt(ciphertext []byte) (bytesconv.BytesResult, error) {
	if s.prk == nil {
		return nil, errors.New("SM2 private key not set")
	}

	return s.prk.Decrypt(rand.Reader, ciphertext, nil)
}

func (s *sm2_algo) Sign(msg []byte) (bytesconv.BytesResult, error) {
	if s.prk == nil {
		return nil, errors.New("SM2 private key not set")
	}

	return s.prk.SignWithSM2(rand.Reader, nil, msg)
}

func (s *sm2_algo) Verify(msg, sign []byte) bool {
	if s.puk == nil {
		return false
	}

	return sm2.VerifyASN1WithSM2(s.puk, nil, msg, sign)
}
