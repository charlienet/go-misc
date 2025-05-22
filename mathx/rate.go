package mathx

import "math"

func Deduction(amount int64, rate float64, min, max int64) int64 {
	isPlus := amount > 0
	fee := int64(math.Abs(math.Round(float64(amount) * rate / 100)))

	// 计算结果低于最小值
	if min > 0 && fee < min {
		return plusOrMinus(isPlus, min)
	}

	// 手续费指定了封顶值并且计算结果大于封顶值
	if max > 0 && fee > max {
		return plusOrMinus(isPlus, max)
	}

	return plusOrMinus(isPlus, fee)
}

func plusOrMinus(isPlus bool, fee int64) int64 {
	if isPlus {
		return fee
	}

	return -fee
}
