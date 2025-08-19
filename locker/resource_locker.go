package locker

import "sync"

const lockPoolSize = 128

type lockPool struct {
	pool sync.Pool
}

func newLockPool() *lockPool {
	return &lockPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(sync.Mutex)
			},
		},
	}
}

func (p *lockPool) get() *sync.Mutex {
	return p.pool.Get().(*sync.Mutex)
}

func (p *lockPool) put(l *sync.Mutex) {
	p.pool.Put(l)
}

type ResourceLocker struct {
	locks sync.Map
	pool  *lockPool
}

func NewResourceLocker() *ResourceLocker {
	return &ResourceLocker{
		pool: newLockPool(),
	}
}

func (rl *ResourceLocker) Lock(key string) {
	lock, _ := rl.locks.LoadOrStore(key, rl.pool.get())
	mutex := lock.(*sync.Mutex)
	mutex.Lock()
}

func (rl *ResourceLocker) Unlock(key string) {
	lock, exists := rl.locks.Load(key)
	if !exists {
		panic("unlocking a non-locked resource")
	}

	mu := lock.(*sync.Mutex)
	mu.Unlock()
	rl.cleanupLock(key, mu)
}

func (rl *ResourceLocker) cleanupLock(key string, mu *sync.Mutex) {
	lock, exist := rl.locks.Load(key)
	if !exist {
		return
	}

	if lock.(*sync.Mutex) != mu {
		if rl.locks.CompareAndDelete(key, mu) {
			rl.pool.put(mu)
		}
	}
}
