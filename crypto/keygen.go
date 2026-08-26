package crypto

import (
    "crypto/ecdsa"
    "crypto/ed25519"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/rsa"
    "fmt"
    "os"
    
    "github.com/emmansun/gmsm/sm2"
)

func GenerateKeyPair(algorithm string, opts ...KeyGenOption) (*KeyPair, error) {
    cfg := &keyGenConfig{
        keySize: 2048,
        curve:   "P256",
    }
    for _, opt := range opts {
        opt(cfg)
    }
    
    algo, err := NormalizeAlgorithm(algorithm)
    if err != nil {
        return nil, err
    }
    
    switch algo {
    case "RSA":
        key, err := rsa.GenerateKey(rand.Reader, cfg.keySize)
        if err != nil {
            return nil, err
        }
        return &KeyPair{PrivateKey: key, PublicKey: &key.PublicKey}, nil
        
    case "SM2":
        key, err := sm2.GenerateKey(rand.Reader)
        if err != nil {
            return nil, err
        }
        return &KeyPair{PrivateKey: key, PublicKey: &key.PublicKey}, nil
        
    case "ECDSA":
        curve := getCurve(cfg.curve)
        if curve == nil {
            return nil, fmt.Errorf("unsupported curve: %s", cfg.curve)
        }
        key, err := ecdsa.GenerateKey(curve, rand.Reader)
        if err != nil {
            return nil, err
        }
        return &KeyPair{PrivateKey: key, PublicKey: &key.PublicKey}, nil
        
    case "ED25519":
        pub, priv, err := ed25519.GenerateKey(rand.Reader)
        if err != nil {
            return nil, err
        }
        return &KeyPair{PrivateKey: priv, PublicKey: pub}, nil
        
    default:
        return nil, fmt.Errorf("unsupported algorithm: %s", algorithm)
    }
}

func getCurve(name string) elliptic.Curve {
    switch name {
    case "P224":
        return elliptic.P224()
    case "P256":
        return elliptic.P256()
    case "P384":
        return elliptic.P384()
    case "P521":
        return elliptic.P521()
    default:
        return nil
    }
}

func LoadKeyPair(filename string, format KeyFormat, opts ...LoadOption) (*KeyPair, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    return ParseKeyPair(data, format, opts...)
}

func LoadPrivateKeyPair(filename string, format KeyFormat, opts ...LoadOption) (*KeyPair, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    return ParsePrivateKeyPair(data, format, opts...)
}

func LoadPublicKeyPair(filename string, format KeyFormat) (*KeyPair, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    return ParsePublicKeyPair(data, format)
}

func ParseKeyPair(data []byte, format KeyFormat, opts ...LoadOption) (*KeyPair, error) {
    kp, err := ParsePrivateKeyPair(data, format, opts...)
    if err == nil {
        return kp, nil
    }
    return ParsePublicKeyPair(data, format)
}

func ParsePrivateKeyPair(data []byte, format KeyFormat, opts ...LoadOption) (*KeyPair, error) {
    cfg := &loadConfig{}
    for _, opt := range opts {
        opt(cfg)
    }
    
    kp := &KeyPair{}
    err := kp.unmarshalPrivateKeyWithPassword(data, format, cfg)
    if err != nil {
        return nil, err
    }
    return kp, nil
}

func ParsePublicKeyPair(data []byte, format KeyFormat) (*KeyPair, error) {
    kp := &KeyPair{}
    err := kp.UnmarshalPublicKey(data, format)
    if err != nil {
        return nil, err
    }
    return kp, nil
}