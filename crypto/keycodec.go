package crypto

import (
    "crypto"
    "crypto/ecdsa"
    "crypto/ed25519"
    "crypto/rsa"
    "crypto/x509"
    "encoding/base64"
    "encoding/hex"
    "encoding/pem"
    "errors"
    "fmt"
    
    "github.com/emmansun/gmsm/sm2"
    "github.com/emmansun/gmsm/smx509"
)

func marshalPublicKey(key crypto.PublicKey, format KeyFormat) ([]byte, error) {
    var der []byte
    var err error
    var pemType string
    
    switch k := key.(type) {
    case *rsa.PublicKey:
        der, err = x509.MarshalPKIXPublicKey(k)
        pemType = "PUBLIC KEY"
    case *ecdsa.PublicKey:
        // 检查是否为SM2公钥
        if sm2.IsSM2PublicKey(k) {
            // 使用gmsm库的SM2密钥编码
            // sm2.PublicKey embeds ecdsa.PublicKey, so we need to cast differently
            der, err = smx509.MarshalPKIXPublicKey(k)
            pemType = "PUBLIC KEY"
        } else {
            der, err = x509.MarshalPKIXPublicKey(k)
            pemType = "PUBLIC KEY"
        }
    case ed25519.PublicKey:
        der, err = x509.MarshalPKIXPublicKey(k)
        pemType = "PUBLIC KEY"
    default:
        return nil, fmt.Errorf("unsupported public key type: %T", key)
    }
    
    if err != nil {
        return nil, fmt.Errorf("failed to marshal public key: %w", err)
    }
    
    return encode(der, format, pemType)
}

func marshalPrivateKey(key crypto.PrivateKey, format KeyFormat, cfg *marshalConfig) ([]byte, error) {
    var der []byte
    var err error
    var pemType string
    
    switch k := key.(type) {
    case *rsa.PrivateKey:
        if cfg != nil && cfg.rsaFormat == RSAKeyFormatPKCS1 {
            der = x509.MarshalPKCS1PrivateKey(k)
            pemType = "RSA PRIVATE KEY"
        } else {
            der, err = x509.MarshalPKCS8PrivateKey(k)
            pemType = "PRIVATE KEY"
        }
    case *ecdsa.PrivateKey:
        // 检查是否为SM2私钥
        if sm2.IsSM2PublicKey(&k.PublicKey) {
            // 使用gmsm库的SM2私钥编码
            // Create a new sm2.PrivateKey with the same values
            sm2Key := &sm2.PrivateKey{
                PrivateKey: *k,
            }
            der, err = smx509.MarshalPKCS8PrivateKey(sm2Key)
            pemType = "PRIVATE KEY"
        } else {
            der, err = x509.MarshalPKCS8PrivateKey(k)
            pemType = "PRIVATE KEY"
        }
    case *sm2.PrivateKey:
        // 直接处理sm2.PrivateKey
        der, err = smx509.MarshalPKCS8PrivateKey(k)
        pemType = "PRIVATE KEY"
    case ed25519.PrivateKey:
        der, err = x509.MarshalPKCS8PrivateKey(k)
        pemType = "PRIVATE KEY"
    default:
        return nil, fmt.Errorf("unsupported private key type: %T", key)
    }
    
    if err != nil {
        return nil, fmt.Errorf("failed to marshal private key: %w", err)
    }
    
    return encode(der, format, pemType)
}

func unmarshalPublicKey(data []byte, format KeyFormat) (crypto.PublicKey, string, error) {
    der, err := decode(data, format)
    if err != nil {
        return nil, "", err
    }
    
    // 首先尝试使用标准库解析
    key, err := x509.ParsePKIXPublicKey(der)
    if err == nil {
        algo := detectAlgorithmFromPublicKey(key)
        return key, algo, nil
    }
    
    // 如果标准库解析失败，尝试使用gmsm库解析SM2
    key, err = smx509.ParsePKIXPublicKey(der)
    if err == nil {
        algo := detectAlgorithmFromPublicKey(key)
        return key, algo, nil
    }
    
    return nil, "", errors.New("unsupported public key format")
}

func detectAlgorithmFromPublicKey(key interface{}) string {
    switch k := key.(type) {
    case *rsa.PublicKey:
        return "RSA"
    case *ecdsa.PublicKey:
        // Check if it's an SM2 key by using sm2.IsSM2PublicKey
        if sm2.IsSM2PublicKey(k) {
            return "SM2"
        }
        return "ECDSA"
    case ed25519.PublicKey:
        return "Ed25519"
    default:
        return ""
    }
}

func unmarshalPrivateKey(data []byte, format KeyFormat, password []byte) (crypto.PrivateKey, error) {
    der, err := decode(data, format)
    if err != nil {
        return nil, err
    }
    
    defer func() {
        for i := range der {
            der[i] = 0
        }
    }()
    
    return parsePrivateKeyDER(der)
}

func parsePrivateKeyDER(der []byte) (crypto.PrivateKey, error) {
    // 首先尝试使用标准库解析PKCS8
    key, err := x509.ParsePKCS8PrivateKey(der)
    if err == nil {
        return key, nil
    }
    
    // 尝试使用gmsm库解析SM2 PKCS8
    sm2Key, err := smx509.ParsePKCS8PrivateKey(der)
    if err == nil {
        return sm2Key, nil
    }
    
    // 尝试解析PKCS1 RSA
    rsaKey, err := x509.ParsePKCS1PrivateKey(der)
    if err == nil {
        return rsaKey, nil
    }
    
    // 尝试解析EC私钥
    ecKey, err := x509.ParseECPrivateKey(der)
    if err == nil {
        return ecKey, nil
    }
    
    return nil, errors.New("unsupported private key format")
}

func encode(der []byte, format KeyFormat, pemType string) ([]byte, error) {
    switch format {
    case KeyFormatBase64:
        encoded := base64.StdEncoding.EncodeToString(der)
        return []byte(encoded), nil
    case KeyFormatPEM:
        block := &pem.Block{
            Type:  pemType,
            Bytes: der,
        }
        return pem.EncodeToMemory(block), nil
    case KeyFormatHex:
        encoded := hex.EncodeToString(der)
        return []byte(encoded), nil
    case KeyFormatRaw:
        result := make([]byte, len(der))
        copy(result, der)
        return result, nil
    default:
        return nil, fmt.Errorf("unsupported key format: %d", format)
    }
}

func decode(data []byte, format KeyFormat) ([]byte, error) {
    switch format {
    case KeyFormatBase64:
        return base64.StdEncoding.DecodeString(string(data))
    case KeyFormatPEM:
        block, _ := pem.Decode(data)
        if block == nil {
            return nil, errors.New("failed to decode PEM block")
        }
        return block.Bytes, nil
    case KeyFormatHex:
        return hex.DecodeString(string(data))
    case KeyFormatRaw:
        result := make([]byte, len(data))
        copy(result, data)
        return result, nil
    default:
        return nil, fmt.Errorf("unsupported key format: %d", format)
    }
}