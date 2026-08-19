package crypto

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== AES-GCM ====================

func TestSymmetric_AES_GCM(t *testing.T) {
	key := []byte("0123456789abcdef") // 16 bytes for AES-128
	plaintext := []byte("hello world")

	cipher, err := NewCipher("AES", key)
	assert.NoError(t, err)

	// 使用 GCM 模式，随机 nonce
	gcm, err := cipher.NewGCMWithRandomNonce()
	assert.NoError(t, err)

	// 加密
	encrypted, err := gcm.Encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEqual(t, plaintext, []byte(encrypted))

	// 解密
	decrypted, err := gcm.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

// ==================== AES-CBC ====================

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
	encrypted, err := cbc.Encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEqual(t, plaintext, []byte(encrypted))

	// 解密
	decrypted, err := cbc.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

// ==================== AES-CTR ====================

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
	assert.NotEqual(t, plaintext, encrypted)

	// 重新创建 CTR（CTR 模式需要重新初始化才能解密）
	ctr2 := cipher.NewCTR(iv)
	decrypted := ctr2.XORKeyStream(encrypted)
	assert.Equal(t, plaintext, decrypted)
}

// ==================== SM4-GCM ====================

func TestSymmetric_SM4(t *testing.T) {
	key := []byte("0123456789abcdef") // 16 bytes for SM4
	plaintext := []byte("hello world")

	cipher, err := NewCipher("SM4", key)
	assert.NoError(t, err)

	// 使用 GCM 模式
	gcm, err := cipher.NewGCMWithRandomNonce()
	assert.NoError(t, err)

	// 加密
	encrypted, err := gcm.Encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEqual(t, plaintext, []byte(encrypted))

	// 解密
	decrypted, err := gcm.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

// ==================== 错误路径 ====================

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

// ==================== 密钥生成 ====================

func TestGenerateKey(t *testing.T) {
	key, iv, nonce, err := GenerateKey("AES")
	assert.NoError(t, err)
	assert.Len(t, key, 16) // AES-128
	assert.Len(t, iv, 16)
	assert.Len(t, nonce, 12)
}

// ==================== RSA（A 库原有） ====================

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

// ==================== SM2（A 库原有） ====================

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

// ==================== PKCS7 填充 ====================

func TestPKCS7Padding(t *testing.T) {
	key := []byte("0123456789abcdef")
	cipher, err := NewCipher("AES", key)
	assert.NoError(t, err)

	sym := cipher.(*symmetric)
	block := sym.Block()

	// 测试填充
	data := []byte("hello")
	padded := pkcs7Padding(block, data)
	assert.Len(t, padded, 16) // 5 + 11 bytes padding

	// 测试去填充
	unpadded, err := pkcs7UnPadding(block, padded)
	assert.NoError(t, err)
	assert.Equal(t, data, unpadded)

	// 测试边界情况：数据长度正好是块大小
	data = []byte("0123456789abcdef") // 16 bytes
	padded = pkcs7Padding(block, data)
	assert.Len(t, padded, 32) // 16 + 16 bytes padding

	unpadded, err = pkcs7UnPadding(block, padded)
	assert.NoError(t, err)
	assert.Equal(t, data, unpadded)
}

// ==================== GCM Nonce 大小 ====================

func TestGCM_NonceSize(t *testing.T) {
	key := []byte("0123456789abcdef")
	c, err := NewCipher("AES", key)
	assert.NoError(t, err)

	gcm, err := c.NewGCMWithRandomNonce()
	assert.NoError(t, err)

	// 加密一个空消息来验证 nonce 大小
	encrypted, err := gcm.Encrypt([]byte{})
	assert.NoError(t, err)
	// 对于空消息，长度应该是 nonceSize + tagSize (12 + 16 = 28)
	assert.Equal(t, 28, len(encrypted))
}

// ==================== CBC IV 长度校验 ====================

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

// ==================== 包级 BlockSize 函数 ====================

func TestBlockSize(t *testing.T) {
	blockSize, ivSize, err := BlockSize("AES")
	assert.NoError(t, err)
	assert.Equal(t, 16, blockSize)
	assert.Equal(t, 16, ivSize)

	blockSize, ivSize, err = BlockSize("SM4")
	assert.NoError(t, err)
	assert.Equal(t, 16, blockSize)
	assert.Equal(t, 16, ivSize)

	// 未知算法应返回错误
	_, _, err = BlockSize("INVALID")
	assert.Error(t, err)
}

// ==================== ECB 模式 ====================

func TestECB(t *testing.T) {
	key := []byte("0123456789abcdef")
	cipher, err := NewCipher("AES", key)
	assert.NoError(t, err)

	ecb, err := cipher.NewECB()
	assert.NoError(t, err)

	plaintext := []byte("hello world!!!") // 填充后对齐块
	encrypted, err := ecb.Encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEqual(t, plaintext, []byte(encrypted))

	decrypted, err := ecb.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

// ==================== 并发安全 ====================

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
			encrypted, err := gcm.Encrypt(plaintext)
			assert.NoError(t, err)
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

// ==================== B 库迁入：SM4-CBC + EmbedIV ====================

func TestSM4CBC_EmbedIV(t *testing.T) {
	key := []byte("1234567890abcdef")
	iv := []byte("1234567890abcdef")
	plain := []byte("hello world")

	c, err := NewCipher("SM4", key)
	assert.NoError(t, err)

	cbc, err := c.NewCBC(iv, EmbedIV())
	assert.NoError(t, err)

	// 加密
	encrypted, err := cbc.Encrypt(plain)
	assert.NoError(t, err)
	assert.NotEqual(t, plain, []byte(encrypted))
	t.Logf("加密结果: %X", encrypted)

	// 解密
	decrypted, err := cbc.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plain, []byte(decrypted))
	t.Logf("解密结果: %s", string(decrypted))
}

// ==================== AES-192 回归测试 ====================

func TestSymmetric_AES192_BlockSize(t *testing.T) {
	key := []byte("0123456789abcdef01234567") // 24 bytes for AES-192

	cipher, err := NewCipher("AES-192", key)
	assert.NoError(t, err)

	// 方法级 BlockSize() 应返回块大小 16（不是密钥长度 24）
	assert.Equal(t, 16, cipher.BlockSize())

	// IVSize() 应返回 16
	assert.Equal(t, 16, cipher.IVSize())
}

func TestSymmetric_AES192_GCM(t *testing.T) {
	key := []byte("0123456789abcdef01234567") // 24 bytes for AES-192
	plaintext := []byte("hello world")

	cipher, err := NewCipher("AES-192", key)
	assert.NoError(t, err)

	// GCM 往返
	gcm, err := cipher.NewGCMWithRandomNonce()
	assert.NoError(t, err)

	encrypted, err := gcm.Encrypt(plaintext)
	assert.NoError(t, err)

	decrypted, err := gcm.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

func TestSymmetric_AES192_CBC(t *testing.T) {
	key := []byte("0123456789abcdef01234567") // 24 bytes
	iv := []byte("abcdef0123456789")          // 16 bytes IV
	plaintext := []byte("hello world")

	cipher, err := NewCipher("AES-192", key)
	assert.NoError(t, err)

	// AES-192 用 16 字节 IV 不应报错（块大小为 16，修复前的 bug：会错误拒绝 16B IV）
	cbc, err := cipher.NewCBC(iv)
	assert.NoError(t, err)

	encrypted, err := cbc.Encrypt(plaintext)
	assert.NoError(t, err)

	decrypted, err := cbc.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

// ==================== AES-256 回归测试 ====================

func TestSymmetric_AES256_BlockSize(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes for AES-256

	cipher, err := NewCipher("AES-256", key)
	assert.NoError(t, err)

	// 方法级 BlockSize() 应返回块大小 16（不是密钥长度 32）
	assert.Equal(t, 16, cipher.BlockSize())

	// IVSize() 应返回 16
	assert.Equal(t, 16, cipher.IVSize())
}

func TestSymmetric_AES256_GCM(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes for AES-256
	plaintext := []byte("hello world")

	cipher, err := NewCipher("AES-256", key)
	assert.NoError(t, err)

	// GCM 往返
	gcm, err := cipher.NewGCMWithRandomNonce()
	assert.NoError(t, err)

	encrypted, err := gcm.Encrypt(plaintext)
	assert.NoError(t, err)

	decrypted, err := gcm.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

func TestSymmetric_AES256_CBC(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	iv := []byte("abcdef0123456789")                  // 16 bytes IV
	plaintext := []byte("hello world")

	cipher, err := NewCipher("AES-256", key)
	assert.NoError(t, err)

	// AES-256 用 16 字节 IV 不应报错（块大小为 16，修复前的 bug：会错误拒绝 16B IV）
	cbc, err := cipher.NewCBC(iv)
	assert.NoError(t, err)

	encrypted, err := cbc.Encrypt(plaintext)
	assert.NoError(t, err)

	decrypted, err := cbc.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

// ==================== EmbedIV 行为测试 ====================

func TestEmbedIV_Behavior(t *testing.T) {
	key := []byte("0123456789abcdef")
	iv := []byte("abcdef0123456789")
	plaintext := []byte("hello world")

	c, err := NewCipher("AES", key)
	assert.NoError(t, err)

	// 使用 EmbedIV 的 CBC
	cbcEmbed, err := c.NewCBC(iv, EmbedIV())
	assert.NoError(t, err)

	encryptedEmbed, err := cbcEmbed.Encrypt(plaintext)
	assert.NoError(t, err)

	// EmbedIV 后密文前缀应为 IV（前 16 字节）
	assert.Equal(t, iv, []byte(encryptedEmbed)[:16], "EmbedIV 后密文前缀应为 IV")

	// 不使用 EmbedIV 的 CBC
	cbcNoEmbed, err := c.NewCBC(iv)
	assert.NoError(t, err)

	encryptedNoEmbed, err := cbcNoEmbed.Encrypt(plaintext)
	assert.NoError(t, err)

	// EmbedIV 密文比非 EmbedIV 密文长一个块大小（16 字节 = IV 长度）
	assert.Equal(t, len(encryptedNoEmbed)+16, len(encryptedEmbed),
		"EmbedIV 密文应比非 EmbedIV 密文长一个块大小")

	// 验证使用 EmbedIV 后仍能正确解密
	decrypted, err := cbcEmbed.Decrypt(encryptedEmbed)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

// ==================== EmbedNonce 行为测试 ====================

func TestEmbedNonce_Behavior(t *testing.T) {
	key := []byte("0123456789abcdef")
	nonce := []byte("0123456789ab") // 12 bytes
	plaintext := []byte("hello world")

	c, err := NewCipher("AES", key)
	assert.NoError(t, err)

	// 使用 EmbedNonce 的 GCM
	gcmEmbed, err := c.NewGCM(nonce, EmbedNonce())
	assert.NoError(t, err)

	encryptedEmbed, err := gcmEmbed.Encrypt(plaintext)
	assert.NoError(t, err)

	// EmbedNonce 后密文前缀应为 nonce（前 12 字节）
	assert.Equal(t, nonce, []byte(encryptedEmbed)[:12], "EmbedNonce 后密文前缀应为 nonce")

	// 验证使用 EmbedNonce 后仍能正确解密
	decrypted, err := gcmEmbed.Decrypt(encryptedEmbed)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

// ==================== StreamCipher.Stream ====================

func TestStreamCipher_Stream(t *testing.T) {
	key := []byte("0123456789abcdef")
	iv := []byte("abcdef0123456789")
	plaintext := []byte("hello world, streaming test!")

	c, err := NewCipher("AES", key)
	assert.NoError(t, err)

	// 加密：XORKeyStream
	enc := c.NewCTR(iv)
	ciphertext := enc.XORKeyStream(plaintext)

	// 解密：Stream(io.Reader) 返回的 reader 读出明文
	dec := c.NewCTR(iv)
	got, err := io.ReadAll(dec.Stream(bytes.NewReader(ciphertext)))
	assert.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

// ==================== GCM NonceSize ====================

func TestGCM_NonceSize_Method(t *testing.T) {
	key := []byte("0123456789abcdef")
	c, err := NewCipher("AES", key)
	assert.NoError(t, err)

	gcm, err := c.NewGCMWithRandomNonce()
	assert.NoError(t, err)
	// NonceSize 不在 CipherMode 接口上，通过类型断言访问
	type nonceSizer interface{ NonceSize() int }
	assert.Equal(t, 12, gcm.(nonceSizer).NonceSize())
}

// ==================== GCM 固定 nonce（embednonce=false 路径）====================

func TestGCM_FixedNonce(t *testing.T) {
	key := []byte("0123456789abcdef")
	nonce := []byte("0123456789ab")
	plaintext := []byte("fixed nonce test")

	c, err := NewCipher("AES", key)
	assert.NoError(t, err)

	// 不 EmbedNonce、固定 nonce → copy(nonce, a.nonce) 路径
	gcm, err := c.NewGCM(nonce)
	assert.NoError(t, err)

	encrypted, err := gcm.Encrypt(plaintext)
	assert.NoError(t, err)
	assert.Equal(t, len(plaintext)+16, len(encrypted)) // ciphertext + tag

	decrypted, err := gcm.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, []byte(decrypted))
}

// ==================== GCM Decrypt 密文过短 ====================

func TestGCM_Decrypt_TooShort(t *testing.T) {
	key := []byte("0123456789abcdef")
	c, err := NewCipher("AES", key)
	assert.NoError(t, err)

	gcm, err := c.NewGCMWithRandomNonce() // embednonce=true
	assert.NoError(t, err)

	_, err = gcm.Decrypt([]byte("short"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

// ==================== NewCBC WithAAD 返回错误 ====================

func TestCBC_WithAAD_Rejected(t *testing.T) {
	key := []byte("0123456789abcdef")
	iv := []byte("abcdef0123456789")
	c, err := NewCipher("AES", key)
	assert.NoError(t, err)

	_, err = c.NewCBC(iv, WithAAD([]byte("aad")))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WithAAD 仅支持 GCM")
}

// ==================== pkcs7UnPadding 非法填充错误 ====================

func TestPKCS7UnPadding_Errors(t *testing.T) {
	key := []byte("0123456789abcdef")
	c, err := NewCipher("AES", key)
	assert.NoError(t, err)

	sym := c.(*symmetric)
	block := sym.Block()

	// unpadding > BlockSize（17 > 16）
	bad1 := make([]byte, 16)
	bad1[15] = 17
	_, err = pkcs7UnPadding(block, bad1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unpadding > BlockSize")

	// unpadding == 0
	bad2 := make([]byte, 16)
	bad2[15] = 0
	_, err = pkcs7UnPadding(block, bad2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unpadding == 0")

	// pad 字节不匹配：最后字节声明 2 字节填充，但倒数第二字节不对
	bad3 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 0xFF, 2}
	_, err = pkcs7UnPadding(block, bad3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pad[i] != unpadding")
}

// ==================== CBC Decrypt EmbedIV 密文过短 ====================

func TestCBC_Decrypt_EmbedIV_TooShort(t *testing.T) {
	key := []byte("0123456789abcdef")
	iv := []byte("abcdef0123456789")
	c, err := NewCipher("AES", key)
	assert.NoError(t, err)

	cbc, err := c.NewCBC(iv, EmbedIV())
	assert.NoError(t, err)

	_, err = cbc.Decrypt([]byte("short"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}
