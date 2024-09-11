package maps

import (
	"maps"

	"github.com/charlienet/misc/locker"
)

type hashmap[M ~map[K]V, K comparable, V any] struct {
	m    M
	l    locker.RWLocker
	sync bool
}

func NewHashMap[M ~map[K]V, K comparable, V any](mm ...M) *hashmap[M, K, V] {
	m := make(M)
	for _, v := range mm {
		maps.Copy(m, v)
	}

	return &hashmap[M, K, V]{
		m: m,
	}
}

func (h *hashmap[M, K, V]) Synchronize() *hashmap[M, K, V] {
	h.l.Synchronize()
	h.sync = true

	return h
}

func (h *hashmap[M, K, V]) Set(k K, v V) {
	h.l.Lock()
	h.m[k] = v
	h.l.Unlock()
}

func (h *hashmap[M, K, V]) Get(key K) (V, bool) {
	h.l.RLock()

	v, ok := h.m[key]

	h.l.RUnlock()
	return v, ok
}
