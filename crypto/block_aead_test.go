package crypto

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 固定 16 字节 SM4 测试密钥。
var testKey = []byte("0123456789abcdef")

// testNonce 生成随机 12 字节 baseNonce。
func testNonce(t testing.TB) []byte {
	t.Helper()
	nonce := make([]byte, blockNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		t.Fatal(err)
	}
	return nonce
}

func mustCipher(t testing.TB) Cipher {
	t.Helper()
	c, err := NewCipher("SM4", testKey)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// randBytes 生成 n 字节伪随机数据。
func randBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		t.Fatal(err)
	}
	return b
}

func encryptToBytes(t *testing.T, plain []byte, nonce []byte, totalSize int64) ([]byte, error) {
	t.Helper()
	er, err := NewEncryptingReader(bytes.NewReader(plain), mustCipher(t), nonce, totalSize)
	if err != nil {
		t.Fatal(err)
	}
	return io.ReadAll(er)
}

func decryptToBytes(t *testing.T, ct []byte, nonce []byte, totalSize int64) ([]byte, error) {
	t.Helper()
	dr, err := NewDecryptingReader(bytes.NewReader(ct), mustCipher(t), nonce, totalSize)
	if err != nil {
		t.Fatal(err)
	}
	return io.ReadAll(dr)
}

// 1. 往返：各边界大小 Encrypt→Decrypt 全等。
func TestBlockAeadRoundTrip(t *testing.T) {
	sizes := []int{0, 1, 4095, 4096, 4097, 1024 * 1024}
	for _, size := range sizes {
		plain := randBytes(t, size)
		nonce := testNonce(t)

		ct, err := encryptToBytes(t, plain, nonce, int64(size))
		if err != nil {
			t.Fatalf("size=%d encrypt: %v", size, err)
		}
		got, err := decryptToBytes(t, ct, nonce, int64(size))
		if err != nil {
			t.Fatalf("size=%d decrypt: %v", size, err)
		}
		if !bytes.Equal(got, plain) {
			t.Fatalf("size=%d roundtrip mismatch", size)
		}
	}
}

// 2. 篡改密文任意一字节 → 解密失败。
func TestBlockAeadTamperCiphertext(t *testing.T) {
	plain := randBytes(t, 64*1024)
	nonce := testNonce(t)
	ct, err := encryptToBytes(t, plain, nonce, int64(len(plain)))
	if err != nil {
		t.Fatal(err)
	}

	// 翻转第 2 块密文区中间一字节（非 tag 区）。
	tampered := append([]byte(nil), ct...)
	tampered[5000] ^= 0xFF

	if _, err := decryptToBytes(t, tampered, nonce, int64(len(plain))); err == nil {
		t.Fatal("tampered ciphertext decrypted successfully")
	}
}

// 3. 篡改 TAG 任意一字节 → 解密失败。
func TestBlockAeadTamperTag(t *testing.T) {
	plain := randBytes(t, 64*1024)
	nonce := testNonce(t)
	ct, err := encryptToBytes(t, plain, nonce, int64(len(plain)))
	if err != nil {
		t.Fatal(err)
	}

	// 翻转最后一个块的认证标签最后一字节。
	tampered := append([]byte(nil), ct...)
	tampered[len(tampered)-1] ^= 0x01

	if _, err := decryptToBytes(t, tampered, nonce, int64(len(plain))); err == nil {
		t.Fatal("tampered tag decrypted successfully")
	}
}

// 4. 块重排（交换前两块）→ 失败（AAD 块号绑定）。
func TestBlockAeadReorderedBlocks(t *testing.T) {
	plain := randBytes(t, 64*1024) // 16 块
	nonce := testNonce(t)
	ct, err := encryptToBytes(t, plain, nonce, int64(len(plain)))
	if err != nil {
		t.Fatal(err)
	}

	blockLen := ChunkSize + TagSize
	reordered := make([]byte, 0, len(ct))
	reordered = append(reordered, ct[blockLen:2*blockLen]...) // 原第 2 块
	reordered = append(reordered, ct[:blockLen]...)           // 原第 1 块
	reordered = append(reordered, ct[2*blockLen:]...)         // 其余块

	if _, err := decryptToBytes(t, reordered, nonce, int64(len(plain))); err == nil {
		t.Fatal("reordered blocks decrypted successfully")
	}
}

// 5. 删尾块 / 尾部追加块 → 截断 / 过长。
func TestBlockAeadTruncateAndAppend(t *testing.T) {
	plain := randBytes(t, 64*1024)
	nonce := testNonce(t)
	ct, err := encryptToBytes(t, plain, nonce, int64(len(plain)))
	if err != nil {
		t.Fatal(err)
	}
	blockLen := ChunkSize + TagSize

	// 删尾块 → 块数不足 → ErrCipherTruncated
	truncated := ct[:len(ct)-blockLen]
	if _, err := decryptToBytes(t, truncated, nonce, int64(len(plain))); !errors.Is(err, ErrCipherTruncated) {
		t.Fatalf("truncated: got %v, want ErrCipherTruncated", err)
	}

	// 尾部追加一个块 → 超出预期块数 → ErrCipherTooLong
	appended := append(append([]byte(nil), ct...), randBytes(t, blockLen)...)
	if _, err := decryptToBytes(t, appended, nonce, int64(len(plain))); !errors.Is(err, ErrCipherTooLong) {
		t.Fatalf("appended: got %v, want ErrCipherTooLong", err)
	}
}

// 6. 用错误的 totalSize 解密 → 失败（AAD size 绑定）。
func TestBlockAeadWrongTotalSize(t *testing.T) {
	plain := randBytes(t, 5000) // 2 块：4096 + 904
	nonce := testNonce(t)
	ct, err := encryptToBytes(t, plain, nonce, int64(len(plain)))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := decryptToBytes(t, ct, nonce, 4096); err == nil {
		t.Fatal("wrong totalSize decrypted successfully")
	}
	if _, err := decryptToBytes(t, ct, nonce, 10000); err == nil {
		t.Fatal("larger wrong totalSize decrypted successfully")
	}
}

// 7. 篡改 baseNonce → 失败。
func TestBlockAeadWrongNonce(t *testing.T) {
	plain := randBytes(t, 64*1024)
	nonce := testNonce(t)
	ct, err := encryptToBytes(t, plain, nonce, int64(len(plain)))
	if err != nil {
		t.Fatal(err)
	}

	wrong := append([]byte(nil), nonce...)
	wrong[0] ^= 0xFF

	if _, err := decryptToBytes(t, ct, wrong, int64(len(plain))); err == nil {
		t.Fatal("wrong baseNonce decrypted successfully")
	}
}

// 8. 流式行为：以 1KB 缓冲逐段读解密输出，与原文一致。
func TestBlockAeadStreamingRead(t *testing.T) {
	plain := randBytes(t, 100*1024)
	nonce := testNonce(t)
	ct, err := encryptToBytes(t, plain, nonce, int64(len(plain)))
	if err != nil {
		t.Fatal(err)
	}

	dr, err := NewDecryptingReader(bytes.NewReader(ct), mustCipher(t), nonce, int64(len(plain)))
	if err != nil {
		t.Fatal(err)
	}

	var got []byte
	buf := make([]byte, 1024)
	for {
		n, err := dr.Read(buf)
		got = append(got, buf[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(got, plain) {
		t.Fatal("streaming read mismatch")
	}
}

// 9. EncryptingReader 实计数 ≠ totalSize → ErrSizeMismatch。
func TestBlockAeadSizeMismatch(t *testing.T) {
	plain := randBytes(t, 1000)
	nonce := testNonce(t)

	if _, err := encryptToBytes(t, plain, nonce, 2000); !errors.Is(err, ErrSizeMismatch) {
		t.Fatalf("got %v, want ErrSizeMismatch", err)
	}
}

// 9b. EncryptingReader.Length() 精确性：对 0/1/4095/4096/4097/1MB 输入，
// Length() 必须等于公式 totalSize + TagSize*ceil(totalSize/ChunkSize)，
// 且与实际读出的密文流长度一致（供 S3 PutObject ContentLength 使用）。
func TestEncryptingReaderLength(t *testing.T) {
	cases := []struct {
		size int
		want int64
	}{
		{0, 0},
		{1, 1 + TagSize},
		{ChunkSize - 1, ChunkSize - 1 + TagSize},
		{ChunkSize, ChunkSize + TagSize},
		{ChunkSize + 1, ChunkSize + 1 + 2*TagSize},
		{1024 * 1024, 1024*1024 + TagSize*(1024*1024/ChunkSize)},
	}
	for _, tc := range cases {
		er, err := NewEncryptingReader(bytes.NewReader(randBytes(t, tc.size)), mustCipher(t), testNonce(t), int64(tc.size))
		if err != nil {
			t.Fatalf("size=%d NewEncryptingReader: %v", tc.size, err)
		}
		if got := er.Length(); got != tc.want {
			t.Errorf("size=%d Length() = %d, want %d", tc.size, got, tc.want)
		}
		// 实际输出的密文流长度必须与 Length() 一致
		ct, err := io.ReadAll(er)
		if err != nil {
			t.Fatalf("size=%d read: %v", tc.size, err)
		}
		if int64(len(ct)) != tc.want {
			t.Errorf("size=%d actual ciphertext len = %d, want %d", tc.size, len(ct), tc.want)
		}
	}
}

// 10. 空文件往返。
func TestBlockAeadEmptyRoundTrip(t *testing.T) {
	nonce := testNonce(t)

	ct, err := encryptToBytes(t, nil, nonce, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ct) != 0 {
		t.Fatalf("empty file produced %d ciphertext bytes", len(ct))
	}

	got, err := decryptToBytes(t, ct, nonce, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("empty file decrypted to %d bytes", len(got))
	}

	// 空密文但声明有内容 → 截断
	if _, err := decryptToBytes(t, nil, nonce, 100); !errors.Is(err, ErrCipherTruncated) {
		t.Fatalf("got %v, want ErrCipherTruncated", err)
	}
}

// deriveNonce 单元行为：大端加法、进位与溢出。
func TestDeriveNonce(t *testing.T) {
	// 0 + 0 = 0
	base := make([]byte, blockNonceSize)
	n, err := deriveNonce(base, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(n, base) {
		t.Fatal("derive 0 mismatch")
	}

	// 低 8 字节大端加法：0xFE + 2 = 0x0100
	base = make([]byte, blockNonceSize)
	base[blockNonceSize-1] = 0xFE
	n, err = deriveNonce(base, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := make([]byte, blockNonceSize)
	want[blockNonceSize-2] = 0x01
	if !bytes.Equal(n, want) {
		t.Fatalf("big-endian add: got %X want %X", n, want)
	}

	// 低 64 位全 FF + 1 → 进位到高 32 位
	base = make([]byte, blockNonceSize)
	for i := 4; i < blockNonceSize; i++ {
		base[i] = 0xFF
	}
	n, err = deriveNonce(base, 1)
	if err != nil {
		t.Fatal(err)
	}
	want = make([]byte, blockNonceSize)
	want[3] = 0x01
	if !bytes.Equal(n, want) {
		t.Fatalf("carry: got %X want %X", n, want)
	}

	// 全 FF + 1 → 溢出
	allFF := bytes.Repeat([]byte{0xFF}, blockNonceSize)
	if _, err := deriveNonce(allFF, 1); !errors.Is(err, ErrNonceOverflow) {
		t.Fatalf("got %v, want ErrNonceOverflow", err)
	}

	// 长度非法
	if _, err := deriveNonce([]byte{1, 2, 3}, 0); !errors.Is(err, ErrInvalidBaseNonce) {
		t.Fatalf("got %v, want ErrInvalidBaseNonce", err)
	}
}

// 快速吞吐基准：SM4-GCM 100MB 数据。
func BenchmarkBlockAeadThroughput(b *testing.B) {
	const size = 100 * 1024 * 1024

	plain := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, plain); err != nil {
		b.Fatal(err)
	}
	nonce := testNonce(b)
	c := mustCipher(b)

	b.SetBytes(size)

	for b.Loop() {
		er, err := NewEncryptingReader(bytes.NewReader(plain), c, nonce, size)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := io.Copy(io.Discard, er); err != nil {
			b.Fatal(err)
		}
	}
}

// errReader 始终返回指定错误的 Reader。
type errReader struct{ err error }

func (r *errReader) Read(p []byte) (int, error) { return 0, r.err }

// ==================== NewEncryptingReader 无效 nonce ====================

func TestNewEncryptingReader_InvalidBaseNonce(t *testing.T) {
	c := mustCipher(t)
	_, err := NewEncryptingReader(bytes.NewReader(nil), c, []byte("short"), 0)
	assert.ErrorIs(t, err, ErrInvalidBaseNonce)
}

// ==================== NewDecryptingReader 无效 nonce ====================

func TestNewDecryptingReader_InvalidBaseNonce(t *testing.T) {
	c := mustCipher(t)
	_, err := NewDecryptingReader(bytes.NewReader(nil), c, []byte("short"), 0)
	assert.ErrorIs(t, err, ErrInvalidBaseNonce)
}

// ==================== EncryptingReader 空缓冲区 Read ====================

func TestEncryptingReader_EmptyBuffer(t *testing.T) {
	er, err := NewEncryptingReader(bytes.NewReader(nil), mustCipher(t), testNonce(t), 0)
	assert.NoError(t, err)

	n, err := er.Read([]byte{})
	assert.Equal(t, 0, n)
	assert.NoError(t, err)
}

// ==================== DecryptingReader 空缓冲区 Read ====================

func TestDecryptingReader_EmptyBuffer(t *testing.T) {
	dr, err := NewDecryptingReader(bytes.NewReader(nil), mustCipher(t), testNonce(t), 0)
	assert.NoError(t, err)

	n, err := dr.Read([]byte{})
	assert.Equal(t, 0, n)
	assert.NoError(t, err)
}

// ==================== EncryptingReader 缓存错误 ====================

func TestEncryptingReader_CachedError(t *testing.T) {
	testErr := errors.New("io error")
	er, err := NewEncryptingReader(&errReader{err: testErr}, mustCipher(t), testNonce(t), 100)
	assert.NoError(t, err)

	buf := make([]byte, 1024)
	_, err = er.Read(buf)
	assert.Error(t, err)

	// 第二次 Read 应返回缓存错误
	_, err = er.Read(buf)
	assert.ErrorIs(t, err, testErr)
}

// ==================== DecryptingReader 缓存错误 ====================

func TestDecryptingReader_CachedError(t *testing.T) {
	testErr := errors.New("io error")
	dr, err := NewDecryptingReader(&errReader{err: testErr}, mustCipher(t), testNonce(t), 100)
	assert.NoError(t, err)

	buf := make([]byte, 1024)
	_, err = dr.Read(buf)
	assert.Error(t, err)

	// 第二次 Read 应返回缓存错误
	_, err = dr.Read(buf)
	assert.ErrorIs(t, err, testErr)
}

// ==================== EncryptingReader nonce 溢出 ====================

func TestEncryptingReader_NonceOverflow(t *testing.T) {
	baseNonce := bytes.Repeat([]byte{0xFF}, blockNonceSize)
	plain := randBytes(t, ChunkSize+1) // 需要 2 块触发 blockIndex=1 溢出

	er, err := NewEncryptingReader(bytes.NewReader(plain), mustCipher(t), baseNonce, int64(len(plain)))
	assert.NoError(t, err)

	// 持续读取直到 nonce 溢出错误
	buf := make([]byte, 1024)
	for {
		_, readErr := er.Read(buf)
		if readErr != nil {
			assert.ErrorIs(t, readErr, ErrNonceOverflow)
			return
		}
	}
}

// ==================== EncryptingReader 读取错误（switch default）====================

func TestEncryptingReader_ReadError(t *testing.T) {
	testErr := errors.New("disk failure")
	er, err := NewEncryptingReader(&errReader{err: testErr}, mustCipher(t), testNonce(t), 100)
	assert.NoError(t, err)

	buf := make([]byte, 1024)
	_, err = er.Read(buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disk failure")
}
