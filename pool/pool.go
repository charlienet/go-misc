package pool

import "sync"

type Pool[T any] interface {
	Get() T
	Put(T)
}

type pool[T any] struct {
	i *sync.Pool
}

func New[T any](new func() T) pool[T] {
	return pool[T]{
		i: &sync.Pool{
			New: func() any {
				return new()
			},
		},
	}
}

func (p pool[T]) Get() T {
	return p.i.Get().(T)
}

func (p pool[T]) Put(t T) {
	p.i.Put(t)
}
