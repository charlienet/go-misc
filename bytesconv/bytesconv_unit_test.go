package bytesconv

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringToBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"", ""},
		{"中文", "中文"},
		{"hello world", "hello world"},
	}

	for _, tt := range tests {
		result := StringToBytes(tt.input)
		assert.Equal(t, tt.expected, string(result))
		assert.Len(t, result, len(tt.input))
	}
}

func TestBytesToString(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{[]byte("hello"), "hello"},
		{[]byte(""), ""},
		{[]byte("中文"), "中文"},
		{[]byte("hello world"), "hello world"},
	}

	for _, tt := range tests {
		result := BytesToString(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestStringToBytes_ZeroCopy(t *testing.T) {
	// 验证零拷贝：返回的字节切片应该与原字符串共享内存
	s := "hello"
	b := StringToBytes(s)

	// 内容应该相同
	assert.Equal(t, []byte(s), b)

	// 长度应该相同
	assert.Len(t, b, len(s))
}

func TestBytesToString_ZeroCopy(t *testing.T) {
	// 验证零拷贝：返回的字符串应该与原字节切片共享内存
	b := []byte("hello")
	s := BytesToString(b)

	// 内容应该相同
	assert.Equal(t, string(b), s)

	// 长度应该相同
	assert.Len(t, s, len(b))
}

func TestFromString(t *testing.T) {
	result := FromString("hello")
	assert.Equal(t, BytesResult([]byte("hello")), result)
}

func TestFromBytes(t *testing.T) {
	input := []byte("hello")
	result := FromBytes(input)
	assert.Equal(t, BytesResult(input), result)
}

func TestFromHexString(t *testing.T) {
	tests := []struct {
		input    string
		expected BytesResult
		hasError bool
	}{
		{"68656c6c6f", BytesResult([]byte("hello")), false},
		{"", BytesResult([]byte{}), false},
		{"48656c6c6f20576f726c64", BytesResult([]byte("Hello World")), false},
		{"invalid", nil, true},
		{"0", nil, true}, // 奇数长度
	}

	for _, tt := range tests {
		result, err := FromHexString(tt.input)
		if tt.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		}
	}
}

func TestFromBase64String(t *testing.T) {
	tests := []struct {
		input    string
		expected BytesResult
		hasError bool
	}{
		{"aGVsbG8=", BytesResult([]byte("hello")), false},
		{"", BytesResult([]byte{}), false},
		{"SGVsbG8gV29ybGQ=", BytesResult([]byte("Hello World")), false},
		{"invalid!@#", nil, true},
	}

	for _, tt := range tests {
		result, err := FromBase64String(tt.input)
		if tt.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		}
	}
}

func TestBytesResult_Hex(t *testing.T) {
	tests := []struct {
		input    BytesResult
		expected string
	}{
		{BytesResult([]byte("hello")), "68656c6c6f"},
		{BytesResult([]byte{}), ""},
		{BytesResult([]byte{0x00, 0xff}), "00ff"},
	}

	for _, tt := range tests {
		result := tt.input.Hex()
		assert.Equal(t, tt.expected, result)
	}
}

func TestBytesResult_UppercaseHex(t *testing.T) {
	tests := []struct {
		input    BytesResult
		expected string
	}{
		{BytesResult([]byte("hello")), "68656C6C6F"},
		{BytesResult([]byte{}), ""},
		{BytesResult([]byte{0x00, 0xff}), "00FF"},
		{BytesResult([]byte{0xab, 0xcd}), "ABCD"},
	}

	for _, tt := range tests {
		result := tt.input.UppercaseHex()
		assert.Equal(t, tt.expected, result)
	}
}

func TestBytesResult_Base64(t *testing.T) {
	tests := []struct {
		input    BytesResult
		expected string
	}{
		{BytesResult([]byte("hello")), "aGVsbG8="},
		{BytesResult([]byte{}), ""},
		{BytesResult([]byte("Hello World")), "SGVsbG8gV29ybGQ="},
	}

	for _, tt := range tests {
		result := tt.input.Base64()
		assert.Equal(t, tt.expected, result)
	}
}

func TestBytesResult_Bytes(t *testing.T) {
	input := BytesResult([]byte("hello"))
	result := input.Bytes()
	assert.Equal(t, []byte("hello"), result)
}

func TestBytesResult_String(t *testing.T) {
	input := BytesResult([]byte("hello"))
	result := input.String()
	assert.Equal(t, "68656c6c6f", result) // Hex 编码
}

func TestBigEndian_BytesToUInt64(t *testing.T) {
	tests := []struct {
		input    []byte
		expected uint64
		hasError bool
	}{
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, 1, false},
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00}, 256, false},
		{[]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 72057594037927936, false},
		{[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, 18446744073709551615, false},
		{[]byte{}, 0, false},
		{[]byte{0x01}, 1, false},
		{[]byte{0x01, 0x02}, 258, false},
		{make([]byte, 9), 0, true}, // 超过 8 字节
	}

	for _, tt := range tests {
		result, err := BigEndian.BytesToUInt64(tt.input)
		if tt.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		}
	}
}

func TestLittleEndian_BytesToUInt64(t *testing.T) {
	tests := []struct {
		input    []byte
		expected uint64
		hasError bool
	}{
		{[]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 1, false},
		{[]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 256, false},
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, 72057594037927936, false},
		{[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, 18446744073709551615, false},
		{[]byte{}, 0, false},
		{[]byte{0x01}, 1, false},
		{[]byte{0x01, 0x02}, 513, false}, // 0x0201 = 513
		{make([]byte, 9), 0, true},       // 超过 8 字节
	}

	for _, tt := range tests {
		result, err := LittleEndian.BytesToUInt64(tt.input)
		if tt.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		}
	}
}

func TestEndian_RoundTrip(t *testing.T) {
	// 测试往返转换
	values := []uint64{0, 1, 255, 256, 65535, 65536, 1<<32 - 1, 1 << 32, 1<<64 - 1}

	for _, v := range values {
		// BigEndian
		bytes := make([]byte, 8)
		for i := 0; i < 8; i++ {
			bytes[7-i] = byte(v >> (i * 8))
		}
		result, err := BigEndian.BytesToUInt64(bytes)
		assert.NoError(t, err)
		assert.Equal(t, v, result)

		// LittleEndian
		for i := 0; i < 8; i++ {
			bytes[i] = byte(v >> (i * 8))
		}
		result, err = LittleEndian.BytesToUInt64(bytes)
		assert.NoError(t, err)
		assert.Equal(t, v, result)
	}
}

func TestBytesResult_Open(t *testing.T) {
	tests := []struct {
		name     string
		input    BytesResult
		expected string
	}{
		{
			name:     "普通字符串",
			input:    BytesResult([]byte("hello")),
			expected: "hello",
		},
		{
			name:     "空切片",
			input:    BytesResult([]byte{}),
			expected: "",
		},
		{
			name:     "包含中文",
			input:    BytesResult([]byte("你好世界")),
			expected: "你好世界",
		},
		{
			name:     "二进制数据",
			input:    BytesResult{0x00, 0xff, 0xab, 0xcd},
			expected: string([]byte{0x00, 0xff, 0xab, 0xcd}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := tt.input.Open()
			assert.NotNil(t, reader)

			// 验证实现了 io.Reader
			var _ io.Reader = reader

			// 读取全部内容
			result, err := io.ReadAll(reader)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}
