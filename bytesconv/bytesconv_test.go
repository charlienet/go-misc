package bytesconv_test

import (
	"testing"

	"github.com/charlienet/go-misc/bytesconv"
)

func BenchmarkStringToBytes(b *testing.B) {
	s := "hello world"
	by := []byte(s)

	b.Run("s", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = []byte(s)
		}
	})

	b.Run("s", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = string(by)
		}
	})

	b.Run("StringToBytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = bytesconv.StringToBytes(s)
		}
	})

	b.Run("BytesToString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = bytesconv.BytesToString(by)
		}
	})
}
