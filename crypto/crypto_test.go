package crypto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSymmetric_AES_GCM(t *testing.T) {
	key := []byte("0123456789abcdef") // 16 bytes for AES-128
	plaintext := []byte("hello world")

	cipher, err := NewCipher("AES", key)
	assert.NoError(t, err)

	// 使用 GCM 模式，随机 nonce
	gcm, err := cipher.NewGCMWithRandomNonce()
	assert.NoError(t, err)

	// 加密
	encrypted := gcm.Encrypt(plaintext)
	assert.NotEqual(t, plaintext, []byte(encrypted))

	// 解密
	decrypted, err := gcm.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

func TestSymmetric_AES_CBC(t *testing.T) {
	key := []byte("0123456789abcdef")
	iv := []byte("abcdef0123456789") // 16 bytes IV
	plaintext := []byte("hello world")

	cipher, err := NewCipher("AES", key)
	assert.NoError(t, err)

	// 使用 CBC 模式
	cbc, err := cipher.NewCBC(iv)
	assert.NoError(t, err)

	// 加密
	encrypted := cbc.Encrypt(plaintext)
	assert.NotEqual(t, plaintext, []byte(encrypted))

	// 解密
	decrypted, err := cbc.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

func TestSymmetric_AES_CTR(t *testing.T) {
	key := []byte("0123456789abcdef")
	iv := []byte("abcdef0123456789")
	plaintext := []byte("hello world")

	cipher, err := NewCipher("AES", key)
	assert.NoError(t, err)

	// 使用 CTR 模式
	ctr := cipher.NewCTR(iv)

	// 加密
	encrypted := ctr.XORKeyStream(plaintext)
	assert.NotEqual(t, plaintext, []byte(encrypted))

	// 重新创建 CTR（CTR 模式需要重新初始化才能解密）
	ctr2 := cipher.NewCTR(iv)
	decrypted := ctr2.XORKeyStream(encrypted)
	assert.Equal(t, plaintext, []byte(decrypted))
}

func TestSymmetric_SM4(t *testing.T) {
	key := []byte("0123456789abcdef") // 16 bytes for SM4
	plaintext := []byte("hello world")

	cipher, err := NewCipher("SM4", key)
	assert.NoError(t, err)

	// 使用 GCM 模式
	gcm, err := cipher.NewGCMWithRandomNonce()
	assert.NoError(t, err)

	// 加密
	encrypted := gcm.Encrypt(plaintext)
	assert.NotEqual(t, plaintext, []byte(encrypted))

	// 解密
	decrypted, err := gcm.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

func TestSymmetric_InvalidAlgorithm(t *testing.T) {
	key := []byte("0123456789abcdef")

	_, err := NewCipher("INVALID", key)
	assert.Error(t, err)
}

func TestSymmetric_InvalidKeyLength(t *testing.T) {
	key := []byte("short") // 太短

	_, err := NewCipher("AES", key)
	assert.Error(t, err)
}

func TestGenerateKey(t *testing.T) {
	key, iv, nonce, err := GenerateKey("AES")
	assert.NoError(t, err)
	assert.Len(t, key, 16) // AES-128
	assert.Len(t, iv, 16)
	assert.Len(t, nonce, 12)
}

func TestRSA_SignAndVerify(t *testing.T) {
	// 生成密钥对
	rsaAlgo, err := NewAsymmetric("RSA")
	assert.NoError(t, err)

	keyPair, err := rsaAlgo.GenerateKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, keyPair.PrivateKey)
	assert.NotEmpty(t, keyPair.PublicKey)

	// 创建新的实例并设置私钥用于签名
	signer, err := NewAsymmetric("RSA", WithPrivateKey(keyPair.PrivateKey))
	assert.NoError(t, err)

	// 签名
	message := []byte("test message")
	signature, err := signer.Sign(message)
	assert.NoError(t, err)
	assert.NotEmpty(t, signature)

	// 创建新实例设置公钥用于验证
	verifier, err := NewAsymmetric("RSA", WithPublicKey(keyPair.PublicKey))
	assert.NoError(t, err)

	// 验证
	valid := verifier.Verify(message, signature)
	assert.True(t, valid)

	// 验证错误消息
	invalid := verifier.Verify([]byte("wrong message"), signature)
	assert.False(t, invalid)
}

func TestRSA_EncryptAndDecrypt(t *testing.T) {
	// 生成密钥对
	rsaAlgo, err := NewAsymmetric("RSA")
	assert.NoError(t, err)

	keyPair, err := rsaAlgo.GenerateKey()
	assert.NoError(t, err)

	// 创建新实例设置密钥
	algo, err := NewAsymmetric("RSA", WithPrivateKey(keyPair.PrivateKey), WithPublicKey(keyPair.PublicKey))
	assert.NoError(t, err)

	// 加密
	plaintext := []byte("secret message")
	ciphertext, err := algo.Encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEqual(t, plaintext, []byte(ciphertext))

	// 解密
	decrypted, err := algo.Decrypt(ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

func TestSM2_SignAndVerify(t *testing.T) {
	// 生成密钥对
	sm2Algo, err := NewAsymmetric("SM2")
	assert.NoError(t, err)

	keyPair, err := sm2Algo.GenerateKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, keyPair.PrivateKey)
	assert.NotEmpty(t, keyPair.PublicKey)

	// 创建新实例设置私钥用于签名
	signer, err := NewAsymmetric("SM2", WithPrivateKey(keyPair.PrivateKey))
	assert.NoError(t, err)

	// 签名
	message := []byte("test message")
	signature, err := signer.Sign(message)
	assert.NoError(t, err)
	assert.NotEmpty(t, signature)

	// 创建新实例设置公钥用于验证
	verifier, err := NewAsymmetric("SM2", WithPublicKey(keyPair.PublicKey))
	assert.NoError(t, err)

	// 验证
	valid := verifier.Verify(message, signature)
	assert.True(t, valid)
}

func TestSM2_EncryptAndDecrypt(t *testing.T) {
	// 生成密钥对
	sm2Algo, err := NewAsymmetric("SM2")
	assert.NoError(t, err)

	keyPair, err := sm2Algo.GenerateKey()
	assert.NoError(t, err)

	// 创建新实例设置密钥
	algo, err := NewAsymmetric("SM2", WithPrivateKey(keyPair.PrivateKey), WithPublicKey(keyPair.PublicKey))
	assert.NoError(t, err)

	// 加密
	plaintext := []byte("secret message")
	ciphertext, err := algo.Encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEqual(t, plaintext, []byte(ciphertext))

	// 解密
	decrypted, err := algo.Decrypt(ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

func TestInvalidAlgorithm(t *testing.T) {
	_, err := NewAsymmetric("INVALID")
	assert.Error(t, err)
}

func TestPKCS7Padding(t *testing.T) {
	key := []byte("0123456789abcdef")
	cipher, err := NewCipher("AES", key)
	assert.NoError(t, err)

	sym := cipher.(*symmetric)

	// 测试填充
	data := []byte("hello")
	padded := sym.pkcs7Padding(data)
	assert.Len(t, padded, 16) // 5 + 11 bytes padding

	// 测试去填充
	unpadded, err := sym.pkcs7UnPadding(padded)
	assert.NoError(t, err)
	assert.Equal(t, data, unpadded)

	// 测试边界情况：数据长度正好是块大小
	data = []byte("0123456789abcdef") // 16 bytes
	padded = sym.pkcs7Padding(data)
	assert.Len(t, padded, 32) // 16 + 16 bytes padding

	unpadded, err = sym.pkcs7UnPadding(padded)
	assert.NoError(t, err)
	assert.Equal(t, data, unpadded)
}

func TestGCM_NonceSize(t *testing.T) {
	key := []byte("0123456789abcdef")
	c, err := NewCipher("AES", key)
	assert.NoError(t, err)

	gcm, err := c.NewGCMWithRandomNonce()
	assert.NoError(t, err)

	// GCM nonce 大小应该是 12 bytes
	// 通过加密一个空消息来验证 nonce 大小
	encrypted := gcm.Encrypt([]byte{})
	// 加密结果包含 nonce + 密文 + tag
	// 对于空消息，长度应该是 nonceSize + tagSize (12 + 16 = 28)
	assert.Equal(t, 28, len(encrypted))
}

func TestCBC_IVLength(t *testing.T) {
	key := []byte("0123456789abcdef")
	cipher, err := NewCipher("AES", key)
	assert.NoError(t, err)

	// IV 长度不正确
	iv := []byte("short")
	_, err = cipher.NewCBC(iv)
	assert.Error(t, err)

	// IV 长度正确
	iv = []byte("0123456789abcdef")
	_, err = cipher.NewCBC(iv)
	assert.NoError(t, err)
}

func TestSymmetric_Concurrent(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := []byte("hello world")

	cipher, err := NewCipher("AES", key)
	assert.NoError(t, err)

	gcm, err := cipher.NewGCMWithRandomNonce()
	assert.NoError(t, err)

	// 并发加密解密
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			encrypted := gcm.Encrypt(plaintext)
			decrypted, err := gcm.Decrypt(encrypted)
			assert.NoError(t, err)
			assert.True(t, bytes.Equal(plaintext, decrypted))
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
