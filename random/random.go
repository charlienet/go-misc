package random

import (
	"crypto/rand"
	"io"
)

type scopeConstraint interface {
	~int | ~int32 | ~int64 | ~uint32
}

func Int[T scopeConstraint]() T {
	return T(rng.Int31())
}

// 生成区间 n >= 0, n < max
func Intn[T scopeConstraint](max T) T {
	n := rng.Int63n(int64(max))
	return T(n) % max
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
