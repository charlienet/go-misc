package crypto

import (
    "crypto/ed25519"
    "crypto/rand"
    "crypto/x509"
    "encoding/base64"
    "errors"
    
    "github.com/charlienet/go-misc/bytesconv"
)

type ed25519_algo struct {
    prk ed25519.PrivateKey
    puk ed25519.PublicKey
}

func new_ed25519(opts ...keyFunc) (Asymmetric, error) {
    key := asymmetric{}
    
    for _, opt := range opts {
        if err := opt(&key); err != nil {
            return nil, err
        }
    }
    
    algo := &ed25519_algo{}
    
    if key.privateKeyObject != nil {
        edKey, ok := key.privateKeyObject.(ed25519.PrivateKey)
        if !ok {
            return nil, errors.New("not an Ed25519 private key")
        }
        algo.prk = edKey
    }
    
    if key.publicKeyObject != nil {
        edKey, ok := key.publicKeyObject.(ed25519.PublicKey)
        if !ok {
            return nil, errors.New("not an Ed25519 public key")
        }
        algo.puk = edKey
    }
    
    return algo, nil
}

func (s *ed25519_algo) Name() string {
    return "Ed25519"
}

func (s *ed25519_algo) GenerateKey() (KeyPair, error) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return KeyPair{}, err
    }
    s.prk = priv
    s.puk = pub
    return KeyPair{PrivateKey: priv, PublicKey: pub}, nil
}

func (s *ed25519_algo) WithPrivateKey(privateKey string) error {
    return errors.New("Ed25519 WithPrivateKey(string) not implemented, use WithPrivateKeyObject")
}

func (s *ed25519_algo) WithPublicKey(publicKey string) error {
    return errors.New("Ed25519 WithPublicKey(string) not implemented, use WithPublicKeyObject")
}

func (s *ed25519_algo) ExportPublicKey() (string, error) {
    if s.prk == nil && s.puk == nil {
        return "", errors.New("no key set")
    }
    
    pub := s.puk
    if pub == nil {
        pub = s.prk.Public().(ed25519.PublicKey)
    }
    
    der, err := x509.MarshalPKIXPublicKey(pub)
    if err != nil {
        return "", err
    }
    
    return base64.StdEncoding.EncodeToString(der), nil
}

func (s *ed25519_algo) Encrypt(msg []byte) (bytesconv.BytesResult, error) {
    return nil, errors.New("Ed25519 does not support encryption")
}

func (s *ed25519_algo) Decrypt(ciphertext []byte) (bytesconv.BytesResult, error) {
    return nil, errors.New("Ed25519 does not support decryption")
}

func (s *ed25519_algo) Sign(data []byte) (bytesconv.BytesResult, error) {
    if s.prk == nil {
        return nil, errors.New("Ed25519 private key not set")
    }
    
    signature := ed25519.Sign(s.prk, data)
    return signature, nil
}

func (s *ed25519_algo) Verify(data, signature []byte) bool {
    if s.puk == nil {
        return false
    }
    
    return ed25519.Verify(s.puk, data, signature)
}