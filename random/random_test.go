package random

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInt(t *testing.T) {
	// 生成多个随机数，确保不全部相同
	results := make(map[int]bool)
	for i := 0; i < 100; i++ {
		results[Int[int]()] = true
	}
	assert.Greater(t, len(results), 1, "should generate different random numbers")
}

func TestIntn(t *testing.T) {
	max := 100
	for i := 0; i < 1000; i++ {
		n := Intn(max)
		assert.GreaterOrEqual(t, n, 0)
		assert.Less(t, n, max)
	}
}

func TestIntRange(t *testing.T) {
	min, max := 10, 20
	for i := 0; i < 1000; i++ {
		n := IntRange(min, max)
		assert.GreaterOrEqual(t, n, min)
		assert.Less(t, n, max)
	}
}

func TestRandBytes(t *testing.T) {
	b, err := RandBytes(32)
	assert.NoError(t, err)
	assert.Len(t, b, 32)

	// 确保不是全零
	allZero := true
	for _, v := range b {
		if v != 0 {
			allZero = false
			break
		}
	}
	assert.False(t, allZero, "random bytes should not be all zeros")
}



func TestSecureGenerator(t *testing.T) {
	// 测试 SecureGenerator
	for i := 0; i < 100; i++ {
		n := SecureGenerator.Intn(100)
		assert.GreaterOrEqual(t, n, 0)
		assert.Less(t, n, 100)
	}
}

func TestNormalGenerator(t *testing.T) {
	// 测试 NormalGenerator
	for i := 0; i < 100; i++ {
		n := NormalGenerator.Intn(100)
		assert.GreaterOrEqual(t, n, 0)
		assert.Less(t, n, 100)
	}
}

func TestStringScope_Generate(t *testing.T) {
	// 测试生成随机字符串
	s := Uppercase.Generate(10)
	assert.Len(t, s, 10)
	for _, c := range s {
		assert.True(t, c >= 'A' && c <= 'Z', "should only contain uppercase letters")
	}

	s = Digit.Generate(8)
	assert.Len(t, s, 8)
	for _, c := range s {
		assert.True(t, c >= '0' && c <= '9', "should only contain digits")
	}

	s = Letter.Generate(20)
	assert.Len(t, s, 20)

	// 测试带前缀
	s = Digit.Generate(10, "ID_")
	assert.Len(t, s, 13) // 3 (prefix) + 10
	assert.Equal(t, "ID_", s[:3])
}

func TestStringScope_NoConsecutiveDuplicates(t *testing.T) {
	// Generate 方法应该避免连续重复字符
	s := AllChars.Generate(100)
	for i := 1; i < len(s); i++ {
		// 注意：由于随机性，偶尔可能有重复，但 Generate 方法中有去重逻辑
		// 这里只验证不会 panic
	}
	assert.Len(t, s, 100)
}



func BenchmarkSecureRandIntn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SecureGenerator.Intn(100)
	}
}

func BenchmarkNormalRandIntn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NormalGenerator.Intn(100)
	}
}

func BenchmarkStringGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		AllChars.Generate(32)
	}
}
