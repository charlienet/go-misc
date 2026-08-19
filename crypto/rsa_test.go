package crypto

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 包级共享的 2048 位测试密钥对：懒生成一次，供多个用例复用，
// 避免每个用例各生成一把 2048 位密钥拖慢测试。密钥随机生成，不打印明文。
var (
	testRSAPairOnce sync.Once
	testRSAPairPrv  *rsa.PrivateKey
	testRSAPairErr  error
)

// getTestRSAPair 返回懒生成的 2048 位测试私钥。
func getTestRSAPair(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	testRSAPairOnce.Do(func() {
		testRSAPairPrv, testRSAPairErr = rsa.GenerateKey(rand.Reader, 2048)
	})
	if testRSAPairErr != nil {
		t.Fatalf("生成测试 RSA 密钥失败: %v", testRSAPairErr)
	}
	return testRSAPairPrv
}

// newTestRSAAlgo 以共享 2048 位密钥构造 Asymmetric 实例。
func newTestRSAAlgo(t *testing.T) Asymmetric {
	t.Helper()
	prv := getTestRSAPair(t)

	prkBytes, err := x509.MarshalPKCS8PrivateKey(prv)
	if err != nil {
		t.Fatal(err)
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&prv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewAsymmetric("RSA",
		WithPrivateKey(base64.StdEncoding.EncodeToString(prkBytes)),
		WithPublicKey(base64.StdEncoding.EncodeToString(pubBytes)),
	)
	if err != nil {
		t.Fatalf("构造 RSA 算法实例失败: %v", err)
	}
	return s
}

// TestRSASignVerify：密钥生成 → Sign → Verify 往返成功（标准 PSS）。
func TestRSASignVerify(t *testing.T) {
	s := newTestRSAAlgo(t)

	msg := []byte("hello, standard RSASSA-PSS")
	sig, err := s.Sign(msg)
	assert.NoError(t, err)
	assert.NotEmpty(t, sig)

	assert.True(t, s.Verify(msg, sig), "合法签名验证失败")
}

// TestRSAVerifyTampered：篡改签名或消息均应验证失败。
func TestRSAVerifyTampered(t *testing.T) {
	s := newTestRSAAlgo(t)

	msg := []byte("integrity check")
	sig, err := s.Sign(msg)
	assert.NoError(t, err)

	// 篡改签名中的一个字节
	tamperedSig := append([]byte(nil), sig...)
	tamperedSig[len(tamperedSig)/2] ^= 0x01
	assert.False(t, s.Verify(msg, tamperedSig), "篡改后的签名被接受")

	// 篡改消息
	assert.False(t, s.Verify(append(msg, 'x'), sig), "篡改后的消息被接受")
}

// TestRSAStandardPSSInterop：证明签名是标准 RSASSA-PSS，可跨实现互通。
// 用标准库 rsa.SignPSS（独立实现）对同一消息摘要签名，项目 Verify 应验证通过；
// 反向，项目签名也应被标准库 rsa.VerifyPSS 接受。
func TestRSAStandardPSSInterop(t *testing.T) {
	prv := getTestRSAPair(t)
	s := newTestRSAAlgo(t)

	msg := []byte("standard PSS interop across implementations")

	h := crypto.SHA256.New()
	h.Write(msg)
	digest := h.Sum(nil)

	// 独立实现签名：标准库直接对消息摘要做 PSS
	stdSig, err := rsa.SignPSS(rand.Reader, prv, crypto.SHA256, digest, nil)
	assert.NoError(t, err, "标准库 SignPSS 失败")

	// 项目 Verify 应能验证标准库生成的签名
	assert.True(t, s.Verify(msg, stdSig), "项目 Verify 拒绝标准库 rsa.SignPSS 生成的签名")

	// 反向验证：项目签名应被标准库 VerifyPSS 接受
	projSig, err := s.Sign(msg)
	assert.NoError(t, err, "项目 Sign 失败")

	err = rsa.VerifyPSS(&prv.PublicKey, crypto.SHA256, digest, projSig, nil)
	assert.NoError(t, err, "标准库 VerifyPSS 拒绝项目签名")
}

// TestRSAWeakKeyRejected：1024 位弱密钥构造应被拒绝（A 库 fail-fast 语义）。
func TestRSAWeakKeyRejected(t *testing.T) {
	weakKey, err := rsa.GenerateKey(rand.Reader, 1024)
	assert.NoError(t, err, "生成 1024 位测试密钥失败")
	assert.Equal(t, 1024, weakKey.N.BitLen(), "预期 1024 位密钥")

	prkB64 := base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(weakKey))

	// A 库 fail-fast 语义：1024 位弱私钥在构造期应被拒绝
	_, err = NewAsymmetric("RSA", WithPrivateKey(prkB64))
	assert.Error(t, err, "1024 位弱私钥应被拒绝，实际被接受")
	assert.True(t, strings.Contains(err.Error(), "too weak"),
		"弱密钥错误信息不符合预期: %v", err)
}

// ==================== RSA Name ====================

func TestRSA_Name(t *testing.T) {
	s, err := NewAsymmetric("RSA")
	assert.NoError(t, err)
	assert.Equal(t, "RSA", s.Name())
}

// ==================== RSA ExportPublicKey ====================

func TestRSA_ExportPublicKey(t *testing.T) {
	s, err := NewAsymmetric("RSA")
	assert.NoError(t, err)

	kp, err := s.GenerateKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, kp.PrivateKey)

	// 仅设置私钥
	signer, err := NewAsymmetric("RSA", WithPrivateKey(kp.PrivateKey))
	assert.NoError(t, err)

	pubB64, err := signer.ExportPublicKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, pubB64)

	// 用导出的公钥回读并验证签名
	verifier, err := NewAsymmetric("RSA", WithPublicKey(pubB64))
	assert.NoError(t, err)

	msg := []byte("test export roundtrip")
	sig, err := signer.Sign(msg)
	assert.NoError(t, err)
	assert.True(t, verifier.Verify(msg, sig))
}

// ==================== RSA nil key paths ====================

func TestRSA_NilKeyPaths(t *testing.T) {
	s, err := NewAsymmetric("RSA")
	assert.NoError(t, err)

	_, err = s.Encrypt([]byte("test"))
	assert.Error(t, err)

	_, err = s.Decrypt([]byte("test"))
	assert.Error(t, err)

	_, err = s.Sign([]byte("test"))
	assert.Error(t, err)

	assert.False(t, s.Verify([]byte("test"), []byte("sig")))
}

// ==================== RSA WithPrivateKey 无效 base64 ====================

func TestRSA_WithPrivateKey_InvalidBase64(t *testing.T) {
	_, err := NewAsymmetric("RSA", WithPrivateKey("!!!not-base64!!!"))
	assert.Error(t, err)
}

// ==================== RSA WithPrivateKey 非 RSA 密钥 ====================

func TestRSA_WithPrivateKey_NonRSAKey(t *testing.T) {
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	prkBytes, err := x509.MarshalPKCS8PrivateKey(ecdsaKey)
	assert.NoError(t, err)

	_, err = NewAsymmetric("RSA", WithPrivateKey(base64.StdEncoding.EncodeToString(prkBytes)))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an RSA private key")
}

// ==================== RSA WithPublicKey 无效 base64 ====================

func TestRSA_WithPublicKey_InvalidBase64(t *testing.T) {
	_, err := NewAsymmetric("RSA", WithPublicKey("!!!not-base64!!!"))
	assert.Error(t, err)
}

// ==================== RSA WithPublicKey 解析错误 ====================

func TestRSA_WithPublicKey_ParserError(t *testing.T) {
	_, err := NewAsymmetric("RSA", WithPublicKey(base64.StdEncoding.EncodeToString([]byte("not-valid-spki"))))
	assert.Error(t, err)
}

// ==================== RSA WithPublicKey 非 RSA 密钥 ====================

func TestRSA_WithPublicKey_NotRSAKey(t *testing.T) {
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	pubBytes, err := x509.MarshalPKIXPublicKey(&ecdsaKey.PublicKey)
	assert.NoError(t, err)

	_, err = NewAsymmetric("RSA", WithPublicKey(base64.StdEncoding.EncodeToString(pubBytes)))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an RSA public key")
}

// ==================== RSA WithPublicKey 弱密钥 ====================

func TestRSA_WithPublicKey_WeakKey(t *testing.T) {
	weakKey, err := rsa.GenerateKey(rand.Reader, 1024)
	assert.NoError(t, err)

	pubBytes, err := x509.MarshalPKIXPublicKey(&weakKey.PublicKey)
	assert.NoError(t, err)

	_, err = NewAsymmetric("RSA", WithPublicKey(base64.StdEncoding.EncodeToString(pubBytes)))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too weak")
}
