package currency

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCentToDollar(t *testing.T) {
	tests := []struct {
		cent     int32
		expected string
	}{
		{100, "1.00"},
		{1, "0.01"},
		{0, "0.00"},
		{999, "9.99"},
		{12345, "123.45"},
		{-100, "-1.00"},
	}

	for _, tt := range tests {
		result := CentToDollar(tt.cent)
		assert.Equal(t, tt.expected, result)
	}
}

func TestDollarToCent(t *testing.T) {
	tests := []struct {
		dollar   string
		expected int64
	}{
		{"1.00", 100},
		{"0.01", 1},
		{"0", 0},
		{"9.99", 999},
		{"123.45", 12345},
		{"-1.00", -100},
	}

	for _, tt := range tests {
		result := DollarToCent(tt.dollar)
		assert.Equal(t, tt.expected, result)
	}
}

func TestFenToYuan(t *testing.T) {
	tests := []struct {
		fen      int
		expected string
	}{
		{100, "1.00"},
		{1, "0.01"},
		{0, "0.00"},
		{999, "9.99"},
	}

	for _, tt := range tests {
		result := FenToYuan(tt.fen)
		assert.Equal(t, tt.expected, result)
	}
}

func TestYuanToFen(t *testing.T) {
	tests := []struct {
		yuan     string
		expected int64
	}{
		{"1.00", 100},
		{"0.01", 1},
		{"0", 0},
		{"9.99", 999},
	}

	for _, tt := range tests {
		result := YuanToFen(tt.yuan)
		assert.Equal(t, tt.expected, result)
	}
}

func TestCentToDollar_RoundTrip(t *testing.T) {
	// 测试分转元再转分的往返一致性
	original := int32(12345)
	dollar := CentToDollar(original)
	backToCent := DollarToCent(dollar)
	assert.Equal(t, int64(original), backToCent)
}
