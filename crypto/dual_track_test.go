package crypto_test

// 双轨一致性测试（阶段 5）：根包协议入口（Encrypt/Decrypt/NewEncryptor/
// NewAsymmetric/NewKeyAgreement/GenerateKeyPair，经注册表分发）与子包
// 直接工厂/函数（symmetric.New / asym.New / agreement.New /
// keymgr.GenerateKeyPair）输出互解、行为一致，验证两条 API 轨道最终落到
// 同一实现。

import (
	"testing"

	"github.com/charlienet/go-misc/crypto"

	"github.com/charlienet/go-misc/crypto/agreement"
	"github.com/charlienet/go-misc/crypto/asym" // import 即注册非对称引擎（协议入口依赖）
	"github.com/charlienet/go-misc/crypto/keymgr"
	"github.com/charlienet/go-misc/crypto/symmetric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDualTrack_Encryptor_Interop 根包 Encrypt/Decrypt 输出 ↔ 子包
// symmetric.New Encryptor 输出互解（AES128×GCM/CBC + SM4×GCM 代表性组合）。
// 相同输入各自独立加密：A 轨密文 B 轨可解，B 轨密文 A 轨可解。
func TestDualTrack_Encryptor_Interop(t *testing.T) {
	combos := []struct {
		name string
		alg  crypto.Algorithm
		mode crypto.Mode
	}{
		{"AES128×GCM", crypto.AES128, crypto.GCM},
		{"AES128×CBC", crypto.AES128, crypto.CBC},
		{"SM4×GCM", crypto.SM4, crypto.GCM},
	}

	for _, tc := range combos {
		t.Run(tc.name, func(t *testing.T) {
			key := make([]byte, tc.alg.KeySize())
			pt := []byte("dual track consistency payload")

			// 轨道 A：根包协议入口 Encrypt（经注册表分发，使用即弃）
			ctA, err := crypto.Encrypt(tc.alg, tc.mode, pt, crypto.WithKey(key))
			require.NoError(t, err)

			// 轨道 B：子包直接工厂 symmetric.New
			e, err := symmetric.New(tc.alg, tc.mode, symmetric.WithKey(key))
			require.NoError(t, err)

			// A 加密 → B 解密
			got, err := e.Decrypt(ctA)
			require.NoError(t, err)
			assert.Equal(t, pt, got, "根包 Encrypt 输出应能被子包 Encryptor 解密")

			// B 加密 → A 解密
			ctB, err := e.Encrypt(pt)
			require.NoError(t, err)
			got, err = crypto.Decrypt(tc.alg, tc.mode, ctB, crypto.WithKey(key))
			require.NoError(t, err)
			assert.Equal(t, pt, got, "子包 Encryptor 输出应能被根包 Decrypt 解密")
		})
	}
}

// TestDualTrack_Encryptor_SameType 编译期断言：crypto.NewEncryptor 与
// symmetric.New 返回同一对象类型（*crypto.Encryptor）。两条轨道的返回值
// 均赋给 *crypto.Encryptor：若任一轨道返回类型不一致，编译即失败。
func TestDualTrack_Encryptor_SameType(t *testing.T) {
	key := make([]byte, 16)

	var root, sub *crypto.Encryptor

	var err error
	root, err = crypto.NewEncryptor(crypto.AES128, crypto.GCM, crypto.WithKey(key))
	require.NoError(t, err)
	sub, err = symmetric.New(crypto.AES128, crypto.GCM, symmetric.WithKey(key))
	require.NoError(t, err)

	// 同一类型下对象层面互换：根包对象加密 → 子包对象解密
	ct, err := root.Encrypt([]byte("same object type"))
	require.NoError(t, err)
	got, err := sub.Decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, []byte("same object type"), got)
}

// TestDualTrack_Asymmetric 根包协议入口与子包直接工厂 asym.New 行为一致
// （构造成功、Name 一致）。子包直接工厂直发包内构造器，不经注册表；
// 根包 NewAsymmetric 经注册表分发到同一引擎实现。
func TestDualTrack_Asymmetric(t *testing.T) {
	a, err := crypto.NewAsymmetric(crypto.RSA)
	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, "RSA", a.Name())

	a2, err := asym.New(crypto.RSA)
	require.NoError(t, err)
	assert.Equal(t, "RSA", a2.Name(), "子包直接工厂构造的实例 Name 应与协议入口一致")
}

// TestDualTrack_KeyAgreement 根包协议入口与子包直接工厂 agreement.New
// 行为一致（构造成功、Name 一致）。
func TestDualTrack_KeyAgreement(t *testing.T) {
	ka, err := crypto.NewKeyAgreement(crypto.ECDH)
	require.NoError(t, err)
	require.NotNil(t, ka)
	assert.Equal(t, "ECDH", ka.Name())

	ka2, err := agreement.New(crypto.ECDH)
	require.NoError(t, err)
	assert.Equal(t, "ECDH", ka2.Name(), "子包直接工厂构造的实例 Name 应与协议入口一致")
}

// TestDualTrack_KeyPair 根包协议入口与 keymgr 直接函数都能生成 SM2 密钥对，
// 且返回同一数据契约类型（*crypto.KeyPair）。
func TestDualTrack_KeyPair(t *testing.T) {
	kp1, err := crypto.GenerateKeyPair(crypto.SM2)
	require.NoError(t, err)
	require.NotNil(t, kp1)

	kp2, err := keymgr.GenerateKeyPair(crypto.SM2)
	require.NoError(t, err)
	require.NotNil(t, kp2)

	// 编译期断言：两条轨道返回同一类型
	var _ *crypto.KeyPair = kp1
	var _ *crypto.KeyPair = kp2
}
