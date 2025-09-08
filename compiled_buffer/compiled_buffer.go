package compiledbuffer

import "sync"

type compiledbuffer[T any] struct {
	buf         map[string]T
	compileFunc func(string) (T, error)
	mu          sync.RWMutex
}

func NewCompiledBuffer[T any](fn func(string) (T, error)) *compiledbuffer[T] {
	return &compiledbuffer[T]{
		buf:         make(map[string]T),
		compileFunc: fn,
	}
}

func (x *compiledbuffer[T]) Put(s string) (T, error) {
	p, err := x.compileFunc(s)
	if err != nil {
		return p, err
	}

	x.mu.Lock()
	x.buf[s] = p
	x.mu.Unlock()

	return p, nil
}

func (x *compiledbuffer[T]) Get(s string) (T, error) {
	x.mu.RLock()
	if p, ok := x.buf[s]; ok {
		x.mu.RUnlock()
		return p, nil
	}
	x.mu.RUnlock()

	return x.Put(s)
}

func (x *compiledbuffer[T]) Clear() {
	x.mu.Lock()
	x.buf = make(map[string]T)
	x.mu.Unlock()
}
