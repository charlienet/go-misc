package compiledbuffer

import "sync"

type CompiledBuffer[T any] struct {
	buf         map[string]T
	compileFunc func(string) (T, error)
	mu          sync.RWMutex
}

func NewCompiledBuffer[T any](fn func(string) (T, error)) *CompiledBuffer[T] {
	return &CompiledBuffer[T]{
		buf:         make(map[string]T),
		compileFunc: fn,
	}
}

func (x *CompiledBuffer[T]) Put(s string) (T, error) {
	p, err := x.compileFunc(s)
	if err != nil {
		return p, err
	}

	x.mu.Lock()
	x.buf[s] = p
	x.mu.Unlock()

	return p, nil
}

func (x *CompiledBuffer[T]) Get(s string) (T, error) {
	x.mu.RLock()
	if p, ok := x.buf[s]; ok {
		x.mu.RUnlock()
		return p, nil
	}
	x.mu.RUnlock()

	return x.Put(s)
}

func (x *CompiledBuffer[T]) Clear() {
	x.mu.Lock()
	x.buf = make(map[string]T)
	x.mu.Unlock()
}
