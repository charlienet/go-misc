package pool

import "bytes"

type BufferPool struct {
	c chan *bytes.Buffer
	a int
}

func NewAllocatedBufferPool(size int, alloc int) *BufferPool {
	return &BufferPool{
		c: make(chan *bytes.Buffer, size),
		a: alloc,
	}
}

func NewBufferPool(size int) (bp *BufferPool) {
	return &BufferPool{
		c: make(chan *bytes.Buffer, size),
	}
}

func (bp *BufferPool) Get() (b *bytes.Buffer) {
	select {
	case b = <-bp.c:
	default:
		b = bytes.NewBuffer(make([]byte, bp.a))
	}
	return
}

func (bp *BufferPool) Put(b *bytes.Buffer) {
	b.Reset()

	if bp.a > 0 && cap(b.Bytes()) > bp.a {
		b = bytes.NewBuffer(make([]byte, 0, bp.a))
	}

	select {
	case bp.c <- b:
	default:
	}
}
