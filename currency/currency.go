package currency

import (
	"strconv"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/constraints"
)

// @title 分转换为元
// @param cent 金额分
// @return 字符串表示的元
func CentToDollar(cent int32) string {
	d := decimal.New(1, 2)

	result := decimal.NewFromInt32(cent).DivRound(d, 2).StringFixedBank(2)

	return result
}

// 元转换为分
func DollarToCent(dollar string) int64 {
	p, _ := strconv.ParseFloat(dollar, 64)
	d := decimal.New(1, 2)

	df := decimal.NewFromFloat(p).Mul(d).IntPart()

	return df
}

func FenToYuan[T constraints.Integer](v T) string {
	d := decimal.New(1, 2)

	return decimal.NewFromInt(int64(v)).DivRound(d, 2).StringFixedBank(2)
}

func YuanToFen(yuan string) int64 {
	p, _ := decimal.NewFromString(yuan)

	d := decimal.New(1, 2)
	return p.Mul(d).IntPart()
}
