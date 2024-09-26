package pool

type Pool[T any] struct {
	c   chan T
	new func() T
}

func New[T any](size int, f func() T) *Pool[T] {
	return &Pool[T]{
		c:   make(chan T, size),
		new: f,
	}
}

func (p *Pool[T]) Get() (b T) {
	select {
	case b = <-p.c:
	default:
		b = p.new()
	}
	return
}

func (p *Pool[T]) Put(b T) {
	select {
	case p.c <- b:
	default:
	}
}
