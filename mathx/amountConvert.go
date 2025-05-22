package mathx

import (
	"strconv"

	"github.com/shopspring/decimal"
)

func CentToDollar(cent int) string {
	d := decimal.New(1, 2)

	result := decimal.NewFromInt32(int32(cent)).DivRound(d, 2).StringFixedBank(2)

	return result
}

// 元转换为分
func DollarToCent(dollar string) int {
	p, _ := strconv.ParseFloat(dollar, 64)
	d := decimal.New(1, 2)

	df := decimal.NewFromFloat(p).Mul(d).IntPart()

	return int(df)
}
