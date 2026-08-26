package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSM2EncryptDecrypt：固定密钥构造 → GenerateKey → Encrypt/Decrypt 往返。
func TestSM2EncryptDecrypt(t *testing.T) {
	// test-only：固定测试密钥，仅用于本测试，不得用于生产
	prv := `MIGTAgEAMBMGByqGSM49AgEGCCqBHM9VAYItBHkwdwIBAQQg7nHbhssWUlVg0Q0z9cYSL00bYdgl7RPhVfKqln7b8j+gCgYIKoEcz1UBgi2hRANCAATLAFa0PaGJuCdAN8iHlPhGWwheohe4SINFlZOmEe2MUxHrlutXyhnPOOLsUt3G9r8wxHDXYt8c5tUUzMQ5aAci`
	pub := `MFkwEwYHKoZIzj0CAQYIKoEcz1UBgi0DQgAEywBWtD2hibgnQDfIh5T4RlsIXqIXuEiDRZWTphHtjFMR65brV8oZzzji7FLdxva/MMRw12LfHObVFMzEOWgHIg==`

	s, err := NewAsymmetric("SM2", WithPrivateKey(prv), WithPublicKey(pub))
	assert.NoError(t, err)

	keyPart, err := s.GenerateKey()
	assert.NoError(t, err)
	// 仅打印密钥类型，避免测试日志泄露密钥明文
	t.Logf("generated private key type: %T", keyPart.PrivateKey)
	t.Logf("generated public key type: %T", keyPart.PublicKey)

	encrypted, err := s.Encrypt([]byte("hello world"))
	assert.NoError(t, err)
	t.Logf("encrypted length: %d bytes", len(encrypted))

	decrypted, err := s.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(decrypted))
	t.Log("decrypt ok")
}

// TestSM2LegacyDERCompatibility：存量 tjfoc/gmsm DER 样本向后兼容验证。
//
// 以下样本均来自 tjfoc/gmsm v1.4.1 生成的 PKCS#8/SPKI DER 数据。
// 验证 emmansun/gmsm v0.44.1 可正确解析、加解密、签名。
func TestSM2LegacyDERCompatibility(t *testing.T) {
	// tjfoc/gmsm 生成的 PKCS#8 私钥 DER（base64）
	legacyPrivDER := `MIGTAgEAMBMGByqGSM49AgEGCCqBHM9VAYItBHkwdwIBAQQgYdEfj4uNxi5U4WOlRC3/K8BLQBEAv/jrYJ50RMX29XqgCgYIKoEcz1UBgi2hRANCAASnEzPWL4n/XDWcEjqHC9kbTAE0Xw1NIidI4r0kN8SQDfpb3ZT0rpYypdlyryNt/Of5DJe01+03lArAtpXlZn1/`

	// tjfoc/gmsm 生成的 SPKI 公钥 DER（base64）
	legacyPubDER := `MFkwEwYHKoZIzj0CAQYIKoEcz1UBgi0DQgAEpxMz1i+J/1w1nBI6hwvZG0wBNF8NTSInSOK9JDfEkA36W92U9K6WMqXZcq8jbfzn+QyXtNftN5QKwLaV5WZ9fw==`

	// tjfoc 加密的存量密文（ASN.1，明文 "legacy sample plaintext"）
	legacyCiphertext := `MIGBAiEAhzoWH9Aq0/AH5UihbwlGu5lSijhRU+FC4oK55zc/pMwCIQCum529pjcB3CX1fQN4bUBK6mMttfImPVA1+7uN5//RDAQgr0X3LfVUyIpE0VqaR3jwPWQ/tbKZkSY/f/rYXvsqLNYEF3k0zVuqL8E+Pd0Agg4VQ0d32GVn8vdb`

	t.Run("NewAsymmetric_with_legacy_keys", func(t *testing.T) {
		s, err := NewAsymmetric("SM2", WithPrivateKey(legacyPrivDER), WithPublicKey(legacyPubDER))
		assert.NoError(t, err)
		assert.Equal(t, "SM2", s.Name())
	})

	t.Run("EncryptDecrypt_roundtrip", func(t *testing.T) {
		s, err := NewAsymmetric("SM2", WithPrivateKey(legacyPrivDER), WithPublicKey(legacyPubDER))
		assert.NoError(t, err)

		encrypted, err := s.Encrypt([]byte("test legacy key"))
		assert.NoError(t, err)
		assert.NotEmpty(t, encrypted)

		decrypted, err := s.Decrypt(encrypted)
		assert.NoError(t, err)
		assert.Equal(t, "test legacy key", string(decrypted))
	})

	t.Run("Decrypt_legacy_ciphertext", func(t *testing.T) {
		s, err := NewAsymmetric("SM2", WithPrivateKey(legacyPrivDER), WithPublicKey(legacyPubDER))
		assert.NoError(t, err)

		cipherBytes, err := base64.StdEncoding.DecodeString(legacyCiphertext)
		assert.NoError(t, err)

		decrypted, err := s.Decrypt(cipherBytes)
		assert.NoError(t, err)
		assert.Equal(t, "legacy sample plaintext", string(decrypted))
	})

	t.Run("SignVerify_roundtrip", func(t *testing.T) {
		s, err := NewAsymmetric("SM2", WithPrivateKey(legacyPrivDER), WithPublicKey(legacyPubDER))
		assert.NoError(t, err)

		msg := []byte("msg")
		sig, err := s.Sign(msg)
		assert.NoError(t, err)
		assert.True(t, s.Verify(msg, sig))
	})
}

// ==================== SM2 ExportPublicKey ====================

func TestSM2_ExportPublicKey(t *testing.T) {
	s, err := NewAsymmetric("SM2")
	assert.NoError(t, err)

	kp, err := s.GenerateKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, kp.PrivateKey)

	// 仅设置私钥
	signer, err := NewAsymmetric("SM2", WithPrivateKeyObject(kp.PrivateKey))
	assert.NoError(t, err)

	pubB64, err := signer.ExportPublicKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, pubB64)

	// 用导出的公钥回读并验证签名
	verifier, err := NewAsymmetric("SM2", WithPublicKey(pubB64))
	assert.NoError(t, err)

	msg := []byte("test export roundtrip")
	sig, err := signer.Sign(msg)
	assert.NoError(t, err)
	assert.True(t, verifier.Verify(msg, sig))
}

// ==================== SM2 nil key paths ====================

func TestSM2_NilKeyPaths(t *testing.T) {
	s, err := NewAsymmetric("SM2")
	assert.NoError(t, err)

	_, err = s.Encrypt([]byte("test"))
	assert.Error(t, err)

	_, err = s.Decrypt([]byte("test"))
	assert.Error(t, err)

	_, err = s.Sign([]byte("test"))
	assert.Error(t, err)

	assert.False(t, s.Verify([]byte("test"), []byte("sig")))
}

// ==================== SM2 WithPrivateKey 无效 base64 ====================

func TestSM2_WithPrivateKey_InvalidBase64(t *testing.T) {
	_, err := NewAsymmetric("SM2", WithPrivateKey("!!!not-base64!!!"))
	assert.Error(t, err)
}

// ==================== SM2 WithPublicKey 无效 base64 ====================

func TestSM2_WithPublicKey_InvalidBase64(t *testing.T) {
	_, err := NewAsymmetric("SM2", WithPublicKey("!!!not-base64!!!"))
	assert.Error(t, err)
}

// ==================== SM2 WithPrivateKey 非 SM2 密钥 ====================

func TestSM2_WithPrivateKey_NonSM2Key(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	prkBytes, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	assert.NoError(t, err)

	_, err = NewAsymmetric("SM2", WithPrivateKey(base64.StdEncoding.EncodeToString(prkBytes)))
	// smx509 解析后类型断言失败或直接解析失败
	assert.Error(t, err)
}

// ==================== SM2 WithPublicKey 非 SM2 密钥 ====================

func TestSM2_WithPublicKey_NonSM2Key(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	pubBytes, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	assert.NoError(t, err)

	_, err = NewAsymmetric("SM2", WithPublicKey(base64.StdEncoding.EncodeToString(pubBytes)))
	// smx509 解析后类型断言失败或直接解析失败
	assert.Error(t, err)
}
