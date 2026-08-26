package crypto

import (
    "crypto"
    "crypto/ecdsa"
    "crypto/ed25519"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
    "errors"
    "fmt"
    "os"
    
    "github.com/emmansun/gmsm/sm2"
)

// KeyFormat 密钥编码格式
type KeyFormat int

const (
    KeyFormatBase64 KeyFormat = iota // Base64 编码的 DER（默认）
    KeyFormatPEM                     // PEM 文本格式
    KeyFormatHex                     // 十六进制编码的 DER
    KeyFormatRaw                     // 原始 DER 字节
)

// RSAKeyFormat RSA 私钥结构格式
type RSAKeyFormat int

const (
    RSAKeyFormatPKCS8 RSAKeyFormat = iota // PKCS#8（默认）
    RSAKeyFormatPKCS1                     // PKCS#1
)

// MarshalOption 导出选项
type MarshalOption func(*marshalConfig)

type marshalConfig struct {
    rsaFormat RSAKeyFormat
}

// WithRSAKeyFormat 设置 RSA 私钥导出格式
func WithRSAKeyFormat(format RSAKeyFormat) MarshalOption {
    return func(cfg *marshalConfig) {
        cfg.rsaFormat = format
    }
}

// LoadOption 加载选项
type LoadOption func(*loadConfig)

type loadConfig struct {
    password []byte
}

// WithPassword 设置密码（用于加密的 PEM）
func WithPassword(password []byte) LoadOption {
    return func(cfg *loadConfig) {
        cfg.password = password
    }
}

// KeyGenOption 密钥生成选项
type KeyGenOption func(*keyGenConfig)

type keyGenConfig struct {
    keySize int    // RSA 密钥位数
    curve   string // ECDSA 曲线名
}

// WithKeySize 设置 RSA 密钥位数
func WithKeySize(bits int) KeyGenOption {
    return func(cfg *keyGenConfig) {
        cfg.keySize = bits
    }
}

// WithCurve 设置 ECDSA 曲线
func WithCurve(curve string) KeyGenOption {
    return func(cfg *keyGenConfig) {
        cfg.curve = curve
    }
}

// KeyPair 存储底层密钥对象
//
// 并发安全说明：
// - 只读方法（MarshalPublicKey、MarshalPrivateKey）是并发安全的
// - 修改方法（UnmarshalPublicKey、UnmarshalPrivateKey、LoadPublicKey、LoadPrivateKey、Reset）不是并发安全的
type KeyPair struct {
    PrivateKey crypto.PrivateKey
    PublicKey  crypto.PublicKey
}

// LegacyKeyPair 使用 Base64 string 存储密钥
//
// Deprecated: 使用 KeyPair 代替。LegacyKeyPair 使用 Base64 string 存储密钥，
// 转换其他格式需要二次编解码。新代码应使用 KeyPair 存储底层密钥对象。
type LegacyKeyPair struct {
    PrivateKey string
    PublicKey  string
}

// Reset 清除密钥并清零敏感内存
func (kp *KeyPair) Reset() {
    switch k := kp.PrivateKey.(type) {
    case *rsa.PrivateKey:
        if k != nil {
            b := k.D.Bits()
            for i := range b {
                b[i] = 0
            }
            for _, p := range k.Primes {
                pb := p.Bits()
                for i := range pb {
                    pb[i] = 0
                }
            }
        }
    case *ecdsa.PrivateKey:
        if k != nil {
            b := k.D.Bits()
            for i := range b {
                b[i] = 0
            }
        }
    case ed25519.PrivateKey:
        for i := range k {
            k[i] = 0
        }
    }
    kp.PrivateKey = nil
    kp.PublicKey = nil
}

// MarshalJSON 禁止序列化（防止私钥泄露）
func (kp *KeyPair) MarshalJSON() ([]byte, error) {
    return nil, errors.New("KeyPair contains private key and cannot be serialized")
}

// UnmarshalJSON 禁止反序列化
func (kp *KeyPair) UnmarshalJSON(data []byte) error {
    return errors.New("KeyPair cannot be deserialized")
}

// --- 编解码方法 ---

func (kp *KeyPair) MarshalPublicKey(format KeyFormat) ([]byte, error) {
    if kp.PublicKey == nil {
        return nil, errors.New("public key is nil")
    }
    return marshalPublicKey(kp.PublicKey, format)
}

func (kp *KeyPair) MarshalPrivateKey(format KeyFormat, opts ...MarshalOption) ([]byte, error) {
    if kp.PrivateKey == nil {
        return nil, errors.New("private key is nil")
    }
    cfg := &marshalConfig{rsaFormat: RSAKeyFormatPKCS8}
    for _, opt := range opts {
        opt(cfg)
    }
    return marshalPrivateKey(kp.PrivateKey, format, cfg)
}

func (kp *KeyPair) UnmarshalPublicKey(data []byte, format KeyFormat) error {
    key, _, err := unmarshalPublicKey(data, format)
    if err != nil {
        return err
    }
    kp.PublicKey = key
    return nil
}

func (kp *KeyPair) UnmarshalPrivateKey(data []byte, format KeyFormat) error {
    key, err := unmarshalPrivateKey(data, format, nil)
    if err != nil {
        return err
    }
    kp.PrivateKey = key
    kp.PublicKey = extractPublicKey(key)
    return nil
}

// --- 文件读写 ---

func (kp *KeyPair) SavePublicKey(filename string, format KeyFormat) error {
    data, err := kp.MarshalPublicKey(format)
    if err != nil {
        return err
    }
    return os.WriteFile(filename, data, 0644)
}

func (kp *KeyPair) SavePrivateKey(filename string, format KeyFormat, opts ...MarshalOption) error {
    data, err := kp.MarshalPrivateKey(format, opts...)
    if err != nil {
        return err
    }
    return os.WriteFile(filename, data, 0600)
}

func (kp *KeyPair) LoadPublicKey(filename string, format KeyFormat) error {
    data, err := os.ReadFile(filename)
    if err != nil {
        return err
    }
    return kp.UnmarshalPublicKey(data, format)
}

func (kp *KeyPair) LoadPrivateKey(filename string, format KeyFormat, opts ...LoadOption) error {
    data, err := os.ReadFile(filename)
    if err != nil {
        return err
    }
    cfg := &loadConfig{}
    for _, opt := range opts {
        opt(cfg)
    }
    return kp.unmarshalPrivateKeyWithPassword(data, format, cfg)
}

func (kp *KeyPair) unmarshalPrivateKeyWithPassword(data []byte, format KeyFormat, cfg *loadConfig) error {
    if format == KeyFormatPEM {
        block, _ := pem.Decode(data)
        if block != nil && x509.IsEncryptedPEMBlock(block) {
            if cfg.password == nil {
                return errors.New("private key is encrypted, password required")
            }
            der, err := x509.DecryptPEMBlock(block, cfg.password)
            if err != nil {
                return fmt.Errorf("failed to decrypt PEM: %w", err)
            }
            defer func() {
                for i := range der {
                    der[i] = 0
                }
            }()
            key, err := parsePrivateKeyDER(der)
            if err != nil {
                return err
            }
            kp.PrivateKey = key
            kp.PublicKey = extractPublicKey(key)
            return nil
        }
    }
    
    key, err := unmarshalPrivateKey(data, format, cfg.password)
    if err != nil {
        return err
    }
    kp.PrivateKey = key
    kp.PublicKey = extractPublicKey(key)
    return nil
}

// --- 辅助函数 ---

func extractPublicKey(key crypto.PrivateKey) crypto.PublicKey {
    switch k := key.(type) {
    case *rsa.PrivateKey:
        return &k.PublicKey
    case *ecdsa.PrivateKey:
        return &k.PublicKey
    case *sm2.PrivateKey:
        return &k.PublicKey
    case ed25519.PrivateKey:
        return k.Public()
    default:
        return nil
    }
}

func detectAlgorithm(key interface{}) string {
    switch k := key.(type) {
    case *rsa.PrivateKey, *rsa.PublicKey:
        return "RSA"
    case *sm2.PrivateKey:
        return "SM2"
    case *ecdsa.PrivateKey:
        // Check if it's an SM2 key by checking if the corresponding public key is an SM2 key
        if sm2.IsSM2PublicKey(&k.PublicKey) {
            return "SM2"
        }
        return "ECDSA"
    case *ecdsa.PublicKey:
        if sm2.IsSM2PublicKey(k) {
            return "SM2"
        }
        return "ECDSA"
    case ed25519.PrivateKey:
        return "Ed25519"
    case ed25519.PublicKey:
        return "Ed25519"
    default:
        return ""
    }
}