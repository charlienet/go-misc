package crypto

import (
	"bytes"
	"crypto/rand"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== gcx1 往返测试 ====================

func TestGCX1_RoundTrip_SM4(t *testing.T) {
	key := []byte("0123456789abcdef") // SM4: 16 bytes
	plaintext := []byte("hello, sm4!")

	envelope, err := Encrypt("SM4", key, plaintext)
	assert.NoError(t, err)
	assert.True(t, bytes.HasPrefix(envelope, []byte(gcx1Magic)))

	decrypted, err := Decrypt(key, envelope)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestGCX1_RoundTrip_AES128(t *testing.T) {
	key := []byte("0123456789abcdef") // AES-128: 16 bytes
	plaintext := []byte("hello, aes128!")

	envelope, err := Encrypt("AES-128", key, plaintext)
	assert.NoError(t, err)

	decrypted, err := Decrypt(key, envelope)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestGCX1_RoundTrip_AES192(t *testing.T) {
	key := []byte("0123456789abcdef01234567") // AES-192: 24 bytes
	plaintext := []byte("hello, aes192!")

	envelope, err := Encrypt("AES-192", key, plaintext)
	assert.NoError(t, err)

	decrypted, err := Decrypt(key, envelope)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestGCX1_RoundTrip_AES256(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // AES-256: 32 bytes
	plaintext := []byte("hello, aes256!")

	envelope, err := Encrypt("AES-256", key, plaintext)
	assert.NoError(t, err)

	decrypted, err := Decrypt(key, envelope)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

// ==================== gcx1 信封结构验证 ====================

func TestGCX1_Structure(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := []byte("test")

	envelope, err := Encrypt("AES-128", key, plaintext)
	assert.NoError(t, err)

	// 验证总长度：header(7) + nonce(12) + ciphertext(4) + tag(16) = 39
	assert.Equal(t, gcx1HeaderLen+gcx1NonceLen+len(plaintext)+TagSize, len(envelope))

	// 验证 header
	assert.Equal(t, byte('g'), envelope[0])
	assert.Equal(t, byte('c'), envelope[1])
	assert.Equal(t, byte('x'), envelope[2])
	assert.Equal(t, byte('1'), envelope[3])
	assert.Equal(t, byte(gcx1Version), envelope[4])
	assert.Equal(t, algIDAES128, envelope[5])
	assert.Equal(t, byte(gcx1NonceLen), envelope[6])
}

// ==================== 空明文往返 ====================

func TestGCX1_EmptyPlaintext(t *testing.T) {
	key := []byte("0123456789abcdef")

	envelope, err := Encrypt("AES-128", key, []byte{})
	assert.NoError(t, err)

	decrypted, err := Decrypt(key, envelope)
	assert.NoError(t, err)
	assert.Empty(t, decrypted)
}

// ==================== NonceLen 不匹配 ====================

func TestGCX1_Tamper_NonceLen(t *testing.T) {
	key := []byte("0123456789abcdef")
	envelope, _ := Encrypt("AES-128", key, []byte("test"))

	tampered := make([]byte, len(envelope))
	copy(tampered, envelope)
	tampered[6] = 0x08 // 错误的 nonceLen（应为 12）

	_, err := Decrypt(key, tampered)
	assert.ErrorIs(t, err, errGcx1NonceLenMismatch)
}

// ==================== EncryptWithAAD 未知算法 ====================

func TestGCX1_EncryptWithAAD_UnknownAlgorithm(t *testing.T) {
	key := []byte("0123456789abcdef")
	_, err := EncryptWithAAD("UNKNOWN", key, []byte("test"), []byte("aad"))
	assert.Error(t, err)
}

// ==================== EncryptWithAAD DES 拒绝 ====================

func TestGCX1_EncryptWithAAD_DES_Rejected(t *testing.T) {
	key := []byte("01234567")
	_, err := EncryptWithAAD("DES", key, []byte("test"), []byte("aad"))
	assert.ErrorIs(t, err, errGcx1DESNotSupported)
}

// ==================== 篡改测试 ====================

func TestGCX1_Tamper_Magic(t *testing.T) {
	key := []byte("0123456789abcdef")
	envelope, _ := Encrypt("AES-128", key, []byte("test"))

	// 篡改魔数
	tampered := make([]byte, len(envelope))
	copy(tampered, envelope)
	tampered[0] = 'X'

	_, err := Decrypt(key, tampered)
	assert.ErrorIs(t, err, errGcx1MagicMismatch)
}

func TestGCX1_Tamper_Version(t *testing.T) {
	key := []byte("0123456789abcdef")
	envelope, _ := Encrypt("AES-128", key, []byte("test"))

	tampered := make([]byte, len(envelope))
	copy(tampered, envelope)
	tampered[4] = 0x02 // 无效版本

	_, err := Decrypt(key, tampered)
	assert.ErrorIs(t, err, errGcx1VersionMismatch)
}

func TestGCX1_Tamper_AlgID(t *testing.T) {
	key := []byte("0123456789abcdef")
	envelope, _ := Encrypt("AES-128", key, []byte("test"))

	tampered := make([]byte, len(envelope))
	copy(tampered, envelope)
	tampered[5] = 0xFF // 未知 algID

	_, err := Decrypt(key, tampered)
	assert.ErrorIs(t, err, errGcx1UnknownAlgID)
}

func TestGCX1_Tamper_Nonce(t *testing.T) {
	key := []byte("0123456789abcdef")
	envelope, _ := Encrypt("AES-128", key, []byte("test"))

	tampered := make([]byte, len(envelope))
	copy(tampered, envelope)
	tampered[7] ^= 0xFF // 篡改 nonce 首字节

	_, err := Decrypt(key, tampered)
	assert.Error(t, err) // GCM Open 失败
}

func TestGCX1_Tamper_Ciphertext(t *testing.T) {
	key := []byte("0123456789abcdef")
	envelope, _ := Encrypt("AES-128", key, []byte("test"))

	tampered := make([]byte, len(envelope))
	copy(tampered, envelope)
	tampered[len(tampered)-3] ^= 0xFF // 篡改密文

	_, err := Decrypt(key, tampered)
	assert.Error(t, err) // GCM Open 失败
}

func TestGCX1_Tamper_Tag(t *testing.T) {
	key := []byte("0123456789abcdef")
	envelope, _ := Encrypt("AES-128", key, []byte("test"))

	tampered := make([]byte, len(envelope))
	copy(tampered, envelope)
	tampered[len(tampered)-1] ^= 0xFF // 篡改 tag 末字节

	_, err := Decrypt(key, tampered)
	assert.Error(t, err) // GCM Open 失败
}

// ==================== AAD 不一致拒绝 ====================

func TestGCX1_AAD_Inconsistent(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := []byte("test aad")
	aad1 := []byte("aad-v1")
	aad2 := []byte("aad-v2")

	envelope, err := EncryptWithAAD("AES-128", key, plaintext, aad1)
	assert.NoError(t, err)

	// 使用不同 AAD 解密，应失败
	_, err = DecryptWithAAD(key, envelope, aad2)
	assert.Error(t, err)
}

func TestGCX1_AAD_Consistent(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := []byte("test aad")
	aad := []byte("consistent-aad")

	envelope, err := EncryptWithAAD("AES-128", key, plaintext, aad)
	assert.NoError(t, err)

	decrypted, err := DecryptWithAAD(key, envelope, aad)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestGCX1_AAD_EncryptWithout_DecryptWith(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := []byte("test")

	// 不使用 AAD 加密
	envelope, err := Encrypt("AES-128", key, plaintext)
	assert.NoError(t, err)

	// 尝试用 AAD 解密，应失败
	_, err = DecryptWithAAD(key, envelope, []byte("extra-aad"))
	assert.Error(t, err)
}

// ==================== 未知算法拒绝 ====================

func TestGCX1_UnknownAlgorithm(t *testing.T) {
	key := []byte("0123456789abcdef")

	_, err := Encrypt("UNKNOWN", key, []byte("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported algorithm")
}

// ==================== DES/3DES 拒绝 ====================

func TestGCX1_DES_Rejected(t *testing.T) {
	key := []byte("01234567") // DES: 8 bytes

	_, err := Encrypt("DES", key, []byte("test"))
	assert.ErrorIs(t, err, errGcx1DESNotSupported)
}

func TestGCX1_3DES_Rejected(t *testing.T) {
	key := []byte("0123456789abcdef01234567") // 3DES: 24 bytes

	_, err := Encrypt("3DES", key, []byte("test"))
	assert.ErrorIs(t, err, errGcx1DESNotSupported)
}

func TestGCX1_DES_AlgoIDRejected(t *testing.T) {
	// 构造一个含有 DES algID 的伪造信封
	key := []byte("01234567")
	fake := make([]byte, 19)
	copy(fake, gcx1Magic)
	fake[4] = gcx1Version
	fake[5] = algIDDES // DES
	fake[6] = gcx1NonceLen

	_, err := Decrypt(key, fake)
	assert.ErrorIs(t, err, errGcx1DESNotSupported)
}

func TestGCX1_3DES_AlgoIDRejected(t *testing.T) {
	key := []byte("0123456789abcdef01234567")
	fake := make([]byte, 19)
	copy(fake, gcx1Magic)
	fake[4] = gcx1Version
	fake[5] = algID3DES
	fake[6] = gcx1NonceLen

	_, err := Decrypt(key, fake)
	assert.ErrorIs(t, err, errGcx1DESNotSupported)
}

// ==================== 长度不足拒绝 ====================

func TestGCX1_TooShort(t *testing.T) {
	key := []byte("0123456789abcdef")

	_, err := Decrypt(key, []byte("short"))
	assert.ErrorIs(t, err, errGcx1TooShort)
}

// ==================== NormalizeAlgorithm ====================

func TestNormalizeAlgorithm_ExactMatch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SM4", AlgorithmSM4},
		{"AES-128", AlgorithmAES128},
		{"AES-192", AlgorithmAES192},
		{"AES-256", AlgorithmAES256},
		{"DES", AlgorithmDES},
		{"3DES", Algorithm3DES},
		{"SM2", AlgorithmSM2},
		{"RSA", AlgorithmRSA},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := NormalizeAlgorithm(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeAlgorithm_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"sm4", AlgorithmSM4},
		{"Sm4", AlgorithmSM4},
		{"aes-128", AlgorithmAES128},
		{"aes-192", AlgorithmAES192},
		{"aes-256", AlgorithmAES256},
		{"des", AlgorithmDES},
		{"3des", Algorithm3DES},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := NormalizeAlgorithm(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeAlgorithm_Alias(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"AES", AlgorithmAES128}, // "AES" 默认归一为 AES-128
		{"aes", AlgorithmAES128},
		{"AES128", AlgorithmAES128}, // 去除连字符匹配
		{"aes128", AlgorithmAES128},
		{"AES192", AlgorithmAES192},
		{"aes192", AlgorithmAES192},
		{"AES256", AlgorithmAES256},
		{"aes256", AlgorithmAES256},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := NormalizeAlgorithm(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeAlgorithm_Unknown(t *testing.T) {
	_, err := NormalizeAlgorithm("BLOWFISH")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported algorithm")
}

// ==================== 并发测试 ====================

func TestGCX1_Concurrent(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := []byte("concurrent test data")

	var wg sync.WaitGroup
	errs := make(chan error, 20)

	for i := 0; i < 10; i++ {
		wg.Add(2)

		// 并发加密
		go func() {
			defer wg.Done()
			_, err := Encrypt("AES-128", key, plaintext)
			errs <- err
		}()

		// 并发加密（带 AAD）
		go func() {
			defer wg.Done()
			_, err := EncryptWithAAD("SM4", key, plaintext, []byte("aad"))
			errs <- err
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		assert.NoError(t, err)
	}
}

func TestGCX1_Concurrent_RoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := []byte("round trip concurrent")

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			envelope, err := Encrypt("AES-128", key, plaintext)
			if err != nil {
				t.Errorf("Encrypt failed: %v", err)
				return
			}

			decrypted, err := Decrypt(key, envelope)
			if err != nil {
				t.Errorf("Decrypt failed: %v", err)
				return
			}
			if !bytes.Equal(plaintext, decrypted) {
				t.Errorf("round trip mismatch: got %s, want %s", decrypted, plaintext)
			}
		}()
	}
	wg.Wait()
}

// ==================== 多算法同密钥加密（确保 nonce 随机性） ====================

func TestGCX1_NonceRandomness(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := []byte("same plaintext")

	// 同一明文加密两次，信封应不同（因为 nonce 随机）
	env1, err := Encrypt("AES-128", key, plaintext)
	assert.NoError(t, err)
	env2, err := Encrypt("AES-128", key, plaintext)
	assert.NoError(t, err)

	// 信封内容应不同（nonce 不同导致密文不同）
	assert.False(t, bytes.Equal(env1, env2), "两次加密应产生不同信封")

	// 但都能正确解密
	d1, err := Decrypt(key, env1)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, d1)
	d2, err := Decrypt(key, env2)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, d2)
}

// ==================== 错误密钥解密 ====================

func TestGCX1_WrongKey(t *testing.T) {
	key1 := []byte("0123456789abcdef")
	key2 := []byte("fedcba9876543210")
	plaintext := []byte("secret")

	envelope, err := Encrypt("AES-128", key1, plaintext)
	assert.NoError(t, err)

	_, err = Decrypt(key2, envelope)
	assert.Error(t, err) // GCM 认证失败
}

// ==================== 大数据量测试 ====================

func TestGCX1_LargePlaintext(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := make([]byte, 1024*1024) // 1MB
	_, err := io.ReadFull(rand.Reader, plaintext)
	assert.NoError(t, err)

	envelope, err := Encrypt("AES-256", key, plaintext)
	assert.NoError(t, err)

	decrypted, err := Decrypt(key, envelope)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
