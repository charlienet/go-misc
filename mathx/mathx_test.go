package mathx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeduction(t *testing.T) {
	tests := []struct {
		amount   int64
		rate     float64
		min      int64
		max      int64
		expected int64
	}{
		{10000, 1.0, 0, 0, 100},      // 1% of 10000 = 100
		{10000, 0.5, 0, 0, 50},       // 0.5% of 10000 = 50
		{100, 10.0, 0, 0, 10},        // 10% of 100 = 10
		{10000, 0.01, 100, 0, 100},   // 低于最小值，返回 min
		{10000, 50.0, 0, 1000, 1000}, // 超过最大值，返回 max
		{-10000, 1.0, 0, 0, -100},    // 负数金额
	}

	for _, tt := range tests {
		result := Deduction(tt.amount, tt.rate, tt.min, tt.max)
		assert.Equal(t, tt.expected, result, "amount=%d, rate=%f, min=%d, max=%d", tt.amount, tt.rate, tt.min, tt.max)
	}
}

func TestRound(t *testing.T) {
	tests := []struct {
		v         float64
		precision int
		expected  float64
	}{
		{3.14159, 0, 3},
		{3.14159, 2, 3.14},
		{3.14159, 4, 3.1416},
		{3.5, 0, 4},
		{2.5, 0, 3},
		{-3.14159, 2, -3.14},
		{1234.567, -2, 1200},   // 四舍五入到百位
		{1250, -2, 1300},       // 边界：1250 → 1300
		{-1234.567, -2, -1200}, // 负数到百位
	}

	for _, tt := range tests {
		result := Round(tt.v, tt.precision)
		assert.InDelta(t, tt.expected, result, 0.0001)
	}
}
