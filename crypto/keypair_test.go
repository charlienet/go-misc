package crypto

import (
    "crypto/ecdsa"
    "crypto/ed25519"
    "crypto/rsa"
    "os"
    "path/filepath"
    "testing"
    
    "github.com/emmansun/gmsm/sm2"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// --- GenerateKeyPair 测试 ---

func TestGenerateKeyPair_RSA(t *testing.T) {
    kp, err := GenerateKeyPair("RSA")
    require.NoError(t, err)
    assert.NotNil(t, kp.PrivateKey)
    assert.NotNil(t, kp.PublicKey)
    assert.IsType(t, &rsa.PrivateKey{}, kp.PrivateKey)
    assert.IsType(t, &rsa.PublicKey{}, kp.PublicKey)
}

func TestGenerateKeyPair_RSA_WithKeySize(t *testing.T) {
    kp, err := GenerateKeyPair("RSA", WithKeySize(4096))
    require.NoError(t, err)
    rsaKey := kp.PrivateKey.(*rsa.PrivateKey)
    assert.Equal(t, 4096, rsaKey.N.BitLen())
}

func TestGenerateKeyPair_SM2(t *testing.T) {
    kp, err := GenerateKeyPair("SM2")
    require.NoError(t, err)
    assert.IsType(t, &sm2.PrivateKey{}, kp.PrivateKey)
}

func TestGenerateKeyPair_ECDSA(t *testing.T) {
    kp, err := GenerateKeyPair("ECDSA")
    require.NoError(t, err)
    assert.IsType(t, &ecdsa.PrivateKey{}, kp.PrivateKey)
}

func TestGenerateKeyPair_ECDSA_WithCurve(t *testing.T) {
    kp, err := GenerateKeyPair("ECDSA", WithCurve("P384"))
    require.NoError(t, err)
    ecKey := kp.PrivateKey.(*ecdsa.PrivateKey)
    assert.Equal(t, "P-384", ecKey.Curve.Params().Name)
}

func TestGenerateKeyPair_Ed25519(t *testing.T) {
    kp, err := GenerateKeyPair("Ed25519")
    require.NoError(t, err)
    assert.IsType(t, ed25519.PrivateKey{}, kp.PrivateKey)
    assert.IsType(t, ed25519.PublicKey{}, kp.PublicKey)
}

func TestGenerateKeyPair_Unsupported(t *testing.T) {
    _, err := GenerateKeyPair("UNKNOWN")
    assert.Error(t, err)
}

// --- Marshal/Unmarshal 测试 ---

func TestKeyPair_MarshalPublicKey_AllFormats(t *testing.T) {
    algorithms := []string{"RSA", "SM2", "ECDSA", "Ed25519"}
    formats := []KeyFormat{KeyFormatBase64, KeyFormatPEM, KeyFormatHex, KeyFormatRaw}
    
    for _, algo := range algorithms {
        kp, err := GenerateKeyPair(algo)
        require.NoError(t, err)
        
        for _, format := range formats {
            data, err := kp.MarshalPublicKey(format)
            require.NoError(t, err, "algorithm=%s format=%d", algo, format)
            assert.NotEmpty(t, data, "algorithm=%s format=%d", algo, format)
            
            // 反向解析
            kp2 := &KeyPair{}
            err = kp2.UnmarshalPublicKey(data, format)
            require.NoError(t, err, "algorithm=%s format=%d", algo, format)
            assert.NotNil(t, kp2.PublicKey)
        }
    }
}

func TestKeyPair_MarshalPrivateKey_AllFormats(t *testing.T) {
    algorithms := []string{"RSA", "SM2", "ECDSA", "Ed25519"}
    formats := []KeyFormat{KeyFormatBase64, KeyFormatPEM, KeyFormatHex, KeyFormatRaw}
    
    for _, algo := range algorithms {
        kp, err := GenerateKeyPair(algo)
        require.NoError(t, err)
        
        for _, format := range formats {
            data, err := kp.MarshalPrivateKey(format)
            require.NoError(t, err, "algorithm=%s format=%d", algo, format)
            assert.NotEmpty(t, data, "algorithm=%s format=%d", algo, format)
            
            // 反向解析（自动提取公钥）
            kp2 := &KeyPair{}
            err = kp2.UnmarshalPrivateKey(data, format)
            require.NoError(t, err, "algorithm=%s format=%d", algo, format)
            assert.NotNil(t, kp2.PrivateKey)
            assert.NotNil(t, kp2.PublicKey, "公钥应自动提取")
        }
    }
}

func TestKeyPair_MarshalPrivateKey_PKCS1(t *testing.T) {
    kp, err := GenerateKeyPair("RSA")
    require.NoError(t, err)
    
    // PKCS#1 格式
    data, err := kp.MarshalPrivateKey(KeyFormatPEM, WithRSAKeyFormat(RSAKeyFormatPKCS1))
    require.NoError(t, err)
    assert.Contains(t, string(data), "RSA PRIVATE KEY")
    
    // PKCS#8 格式（默认）
    data8, err := kp.MarshalPrivateKey(KeyFormatPEM)
    require.NoError(t, err)
    assert.Contains(t, string(data8), "PRIVATE KEY")
    assert.NotContains(t, string(data8), "RSA PRIVATE KEY")
}

func TestKeyPair_MarshalNilKey(t *testing.T) {
    kp := &KeyPair{}
    
    _, err := kp.MarshalPublicKey(KeyFormatPEM)
    assert.Error(t, err)
    
    _, err = kp.MarshalPrivateKey(KeyFormatPEM)
    assert.Error(t, err)
}

// --- 文件读写测试 ---

func TestKeyPair_SaveLoadPublicKey(t *testing.T) {
    dir := t.TempDir()
    formats := []struct {
        name   string
        format KeyFormat
        ext    string
    }{
        {"PEM", KeyFormatPEM, ".pem"},
        {"Base64", KeyFormatBase64, ".b64"},
        {"Hex", KeyFormatHex, ".hex"},
        {"Raw", KeyFormatRaw, ".der"},
    }
    
    for _, algo := range []string{"RSA", "SM2", "ECDSA", "Ed25519"} {
        kp, err := GenerateKeyPair(algo)
        require.NoError(t, err)
        
        for _, f := range formats {
            filename := filepath.Join(dir, algo+"_pub"+f.ext)
            
            err = kp.SavePublicKey(filename, f.format)
            require.NoError(t, err)
            
            // 验证文件权限
            info, err := os.Stat(filename)
            require.NoError(t, err)
            assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
            
            // 加载并验证
            kp2 := &KeyPair{}
            err = kp2.LoadPublicKey(filename, f.format)
            require.NoError(t, err)
            assert.NotNil(t, kp2.PublicKey)
        }
    }
}

func TestKeyPair_SaveLoadPrivateKey(t *testing.T) {
    dir := t.TempDir()
    
    kp, err := GenerateKeyPair("RSA")
    require.NoError(t, err)
    
    filename := filepath.Join(dir, "priv.pem")
    err = kp.SavePrivateKey(filename, KeyFormatPEM)
    require.NoError(t, err)
    
    // 验证文件权限（私钥应该是 0600）
    info, err := os.Stat(filename)
    require.NoError(t, err)
    assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
    
    // 加载
    kp2 := &KeyPair{}
    err = kp2.LoadPrivateKey(filename, KeyFormatPEM)
    require.NoError(t, err)
    assert.NotNil(t, kp2.PrivateKey)
    assert.NotNil(t, kp2.PublicKey, "加载私钥后应自动提取公钥")
}

// --- 包级加载函数测试 ---

func TestLoadKeyPair_PrivateKey(t *testing.T) {
    dir := t.TempDir()
    
    kp, _ := GenerateKeyPair("RSA")
    filename := filepath.Join(dir, "priv.pem")
    kp.SavePrivateKey(filename, KeyFormatPEM)
    
    loaded, err := LoadKeyPair(filename, KeyFormatPEM)
    require.NoError(t, err)
    assert.NotNil(t, loaded.PrivateKey)
    assert.NotNil(t, loaded.PublicKey)
}

func TestLoadKeyPair_PublicKey(t *testing.T) {
    dir := t.TempDir()
    
    kp, _ := GenerateKeyPair("RSA")
    filename := filepath.Join(dir, "pub.pem")
    kp.SavePublicKey(filename, KeyFormatPEM)
    
    loaded, err := LoadKeyPair(filename, KeyFormatPEM)
    require.NoError(t, err)
    assert.Nil(t, loaded.PrivateKey)
    assert.NotNil(t, loaded.PublicKey)
}

func TestParseKeyPair(t *testing.T) {
    kp, _ := GenerateKeyPair("RSA")
    
    // 解析私钥
    data, _ := kp.MarshalPrivateKey(KeyFormatPEM)
    parsed, err := ParseKeyPair(data, KeyFormatPEM)
    require.NoError(t, err)
    assert.NotNil(t, parsed.PrivateKey)
    
    // 解析公钥
    data, _ = kp.MarshalPublicKey(KeyFormatPEM)
    parsed, err = ParseKeyPair(data, KeyFormatPEM)
    require.NoError(t, err)
    assert.Nil(t, parsed.PrivateKey)
    assert.NotNil(t, parsed.PublicKey)
}

// --- Reset 测试 ---

func TestKeyPair_Reset(t *testing.T) {
    kp, _ := GenerateKeyPair("RSA")
    assert.NotNil(t, kp.PrivateKey)
    assert.NotNil(t, kp.PublicKey)
    
    kp.Reset()
    assert.Nil(t, kp.PrivateKey)
    assert.Nil(t, kp.PublicKey)
}

func TestKeyPair_Reset_AllTypes(t *testing.T) {
    algorithms := []string{"RSA", "SM2", "ECDSA", "Ed25519"}
    
    for _, algo := range algorithms {
        kp, err := GenerateKeyPair(algo)
        require.NoError(t, err)
        
        kp.Reset()
        assert.Nil(t, kp.PrivateKey, "algorithm=%s", algo)
        assert.Nil(t, kp.PublicKey, "algorithm=%s", algo)
    }
}

// --- 序列化禁止测试 ---

func TestKeyPair_MarshalJSON_Forbidden(t *testing.T) {
    kp, _ := GenerateKeyPair("RSA")
    
    _, err := kp.MarshalJSON()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "cannot be serialized")
}

func TestKeyPair_UnmarshalJSON_Forbidden(t *testing.T) {
    kp := &KeyPair{}
    
    err := kp.UnmarshalJSON([]byte("{}"))
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "cannot be deserialized")
}

// --- 并发安全测试 ---

func TestKeyPair_Concurrent_MarshalPublicKey(t *testing.T) {
    kp, _ := GenerateKeyPair("RSA")
    
    done := make(chan bool, 20)
    formats := []KeyFormat{KeyFormatBase64, KeyFormatPEM, KeyFormatHex, KeyFormatRaw}
    
    for i := 0; i < 20; i++ {
        go func(idx int) {
            format := formats[idx%len(formats)]
            data, err := kp.MarshalPublicKey(format)
            assert.NoError(t, err)
            assert.NotEmpty(t, data)
            done <- true
        }(i)
    }
    
    for i := 0; i < 20; i++ {
        <-done
    }
}

func TestKeyPair_Concurrent_MarshalPrivateKey(t *testing.T) {
    kp, _ := GenerateKeyPair("RSA")
    
    done := make(chan bool, 20)
    formats := []KeyFormat{KeyFormatBase64, KeyFormatPEM, KeyFormatHex, KeyFormatRaw}
    
    for i := 0; i < 20; i++ {
        go func(idx int) {
            format := formats[idx%len(formats)]
            data, err := kp.MarshalPrivateKey(format)
            assert.NoError(t, err)
            assert.NotEmpty(t, data)
            done <- true
        }(i)
    }
    
    for i := 0; i < 20; i++ {
        <-done
    }
}

func TestGenerateKey_Concurrent(t *testing.T) {
    done := make(chan bool, 20)
    algorithms := []string{"RSA", "SM2", "ECDSA", "Ed25519"}
    
    for i := 0; i < 20; i++ {
        go func(idx int) {
            algo := algorithms[idx%len(algorithms)]
            kp, err := GenerateKeyPair(algo)
            assert.NoError(t, err)
            assert.NotNil(t, kp)
            assert.NotNil(t, kp.PrivateKey)
            assert.NotNil(t, kp.PublicKey)
            done <- true
        }(i)
    }
    
    for i := 0; i < 20; i++ {
        <-done
    }
}

func TestAsymmetric_Concurrent_SignVerify(t *testing.T) {
    algorithms := []string{"RSA", "SM2", "ECDSA", "Ed25519"}
    
    for _, algo := range algorithms {
        kp, err := GenerateKeyPair(algo)
        require.NoError(t, err)
        
        signer, err := NewAsymmetric(algo, WithPrivateKeyObject(kp.PrivateKey))
        require.NoError(t, err)
        
        verifier, err := NewAsymmetric(algo, WithPublicKeyObject(kp.PublicKey))
        require.NoError(t, err)
        
        done := make(chan bool, 10)
        data := []byte("concurrent sign verify test")
        
        for i := 0; i < 10; i++ {
            go func() {
                sig, err := signer.Sign(data)
                assert.NoError(t, err)
                assert.True(t, verifier.Verify(data, sig))
                done <- true
            }()
        }
        
        for i := 0; i < 10; i++ {
            <-done
        }
    }
}

func TestSymmetric_CFB_OFB_Concurrent(t *testing.T) {
    key := make([]byte, 16)
    iv := make([]byte, 16)
    
    c, _ := NewCipher("AES-128", key)
    
    done := make(chan bool, 20)
    
    // CFB 并发
    for i := 0; i < 10; i++ {
        go func() {
            cfb, _ := c.NewCFB(iv)
            plaintext := []byte("concurrent CFB test data")
            encrypted, err := cfb.Encrypt(plaintext)
            assert.NoError(t, err)
            decrypted, err := cfb.Decrypt(encrypted)
            assert.NoError(t, err)
            assert.Equal(t, plaintext, []byte(decrypted))
            done <- true
        }()
    }
    
    // OFB 并发
    for i := 0; i < 10; i++ {
        go func() {
            ofb, _ := c.NewOFB(iv)
            plaintext := []byte("concurrent OFB test data")
            encrypted, err := ofb.Encrypt(plaintext)
            assert.NoError(t, err)
            decrypted, err := ofb.Decrypt(encrypted)
            assert.NoError(t, err)
            assert.Equal(t, plaintext, []byte(decrypted))
            done <- true
        }()
    }
    
    for i := 0; i < 20; i++ {
        <-done
    }
}

// --- Race 检测 ---

func TestKeyPair_Race_MarshalUnmarshal(t *testing.T) {
    kp, _ := GenerateKeyPair("RSA")
    
    done := make(chan bool, 10)
    
    // 并发 Marshal（只读，应安全）
    for i := 0; i < 5; i++ {
        go func() {
            _, _ = kp.MarshalPublicKey(KeyFormatPEM)
            _, _ = kp.MarshalPrivateKey(KeyFormatPEM)
            done <- true
        }()
    }
    
    // 并发 Unmarshal（修改，不应与 Marshal 并发）
    // 这里只测试 Marshal 的并发安全性
    for i := 0; i < 5; i++ {
        <-done
    }
}