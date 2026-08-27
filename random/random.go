package random

import (
	"crypto/rand"
	"io"
)

var (
	rng             = NormalGenerator
	SecureGenerator = &secureRandGenerator{}
	NormalGenerator = NewRandGenerator()
)

type scopeConstraint interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func Int[T scopeConstraint]() T {
	return T(rng.Int31())
}

// 生成区间 n >= 0, n < max
func Intn[T scopeConstraint](max T) T {
	n := rng.Int63n(int64(max))
	return T(n)
}

// 生成区间 n >= min, n < max
func IntRange[T scopeConstraint](min, max T) T {
	n := Intn(max - min)
	return T(n + min)
}

func RandBytes(len int) ([]byte, error) {
	r := make([]byte, len)
	_, err := io.ReadFull(rand.Reader, r)
	return r, err
}
