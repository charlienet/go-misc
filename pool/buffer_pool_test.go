package pool

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBufferPool(t *testing.T) {
	var size = 4
	var allocated = 1024

	p := NewAllocatedBufferPool(size, allocated)
	b := p.Get()
	assert.Equal(t, allocated, cap(b.Bytes()))

	b.Grow(allocated * 3)
	p.Put(b)

	for i := 0; i < size; i++ {
		p.Put(bytes.NewBuffer(make([]byte, 0, p.a*2)))
	}

	assert.LessOrEqual(t, len(p.c), size)
}

func BenchmarkGetPut(b *testing.B) {
	var size = 4
	var allocated = 1024
	p := NewAllocatedBufferPool(size, allocated)

	b.Run("t", func(b *testing.B) {
		for range b.N {
			b := p.Get()
			b.Write([]byte{23})
			_ = b.Bytes()

			p.Put(b)
		}
	})

	b.Run("t", func(b *testing.B) {
		for range b.N {
			b := bytes.NewBuffer(make([]byte, 0, 1024))

			b.Write([]byte{23})
			_ = b.Bytes()

			p.Put(b)
		}
	})
}
