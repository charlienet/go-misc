package symmetric

// symmetric.New 薄包装测试：一行转发 crypto.NewEncryptor，
// 覆盖子包直接工厂构造可复用对象的往返、属性查询与构造错误路径。

import (
	"testing"

	"github.com/charlienet/go-misc/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_RoundTrip 六模式经子包 New 构造后多轮往返。
func TestNew_RoundTrip(t *testing.T) {
	key := make([]byte, 16)
	pt := []byte("wrap new roundtrip payload")
	modes := []crypto.Mode{crypto.GCM, crypto.CBC, crypto.ECB, crypto.CFB, crypto.OFB, crypto.CTR}

	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			opts := []crypto.Option{WithKey(key)}
			if mode == crypto.ECB {
				opts = append(opts, crypto.WithInsecureAlgorithms())
			}
			e, err := New(crypto.AES128, mode, opts...)
			require.NoError(t, err)
			require.NotNil(t, e)

			ct, err := e.Encrypt(pt)
			require.NoError(t, err)
			got, err := e.Decrypt(ct)
			require.NoError(t, err)
			assert.Equal(t, pt, got)

			assert.Equal(t, crypto.AES128, e.Algorithm())
			assert.Equal(t, mode, e.Mode())
		})
	}
}

// TestNew_Options 子包重导出的选项构造器可直接使用。
func TestNew_Options(t *testing.T) {
	e, err := New(crypto.AES128, crypto.GCM, WithKeyPassword("0123456789abcdef"))
	require.NoError(t, err)
	ct, err := e.Encrypt([]byte("option re-export"))
	require.NoError(t, err)
	got, err := e.Decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, []byte("option re-export"), got)
}

// TestNew_ConstructionError 构造错误经子包 New 透传根包哨兵。
func TestNew_ConstructionError(t *testing.T) {
	_, err := New(crypto.AES128, crypto.GCM) // 缺密钥
	assert.ErrorIs(t, err, crypto.ErrKeyRequired)

	_, err = New(crypto.DES, crypto.GCM, WithKey(make([]byte, 8)), crypto.WithInsecureAlgorithms())
	assert.ErrorIs(t, err, crypto.ErrIncompatibleAlgorithmMode)
}
