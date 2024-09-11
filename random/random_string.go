package random

import "github.com/charlienet/go-misc/bytesconv"

const (
	uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercase = "abcdefghijklmnopqrstuvwxyz"
	digit     = "0123456789"
	nomix     = "BCDFGHJKMPQRTVWXY2346789"
	letter    = uppercase + lowercase
	allChars  = uppercase + lowercase + digit
	hex       = digit + "ABCDEF"
	_         = allChars + "/+"
)

var (
	rng             = &fastRandGenerator{}
	FastGenerator   = &fastRandGenerator{}
	SecureGenerator = &secureRandGenerator{}
	NormalGenerator = NewRandGenerator()
)

type charScope struct {
	bytes  []byte
	length int
	max    int
	bits   int
	mask   int
}

var (
	Uppercase = StringScope(uppercase) // 大写字母
	Lowercase = StringScope(lowercase) // 小写字母
	Digit     = StringScope(digit)     // 数字
	Nomix     = StringScope(nomix)     // 不混淆字符
	Letter    = StringScope(letter)    // 字母
	Hex       = StringScope(hex)       // 十六进制字符
	AllChars  = StringScope(allChars)  // 所有字符
)

func StringScope(str string) *charScope {
	len := len(str)

	scope := &charScope{
		bytes:  bytesconv.StringToBytes(str),
		length: len,
		bits:   1,
	}

	for scope.mask < len {
		scope.bits++
		scope.mask = 1<<scope.bits - 1
	}

	scope.max = scope.mask / scope.bits

	return scope
}

// 生成指定长度的随机字符串
func (scope *charScope) Generate(length int) string {
	n := length
	ret := make([]byte, n)

	for i, cache, remain := n-1, rng.Int63(), scope.max; i >= 0; {
		if remain == 0 {
			cache, remain = rng.Int63(), scope.max
		}

		if idx := int(cache & int64(scope.mask)); idx < scope.length {
			ret[i] = scope.bytes[idx]
			i--
		}

		cache >>= int64(scope.bits)
		remain--
	}

	return bytesconv.BytesToString(ret)
}
