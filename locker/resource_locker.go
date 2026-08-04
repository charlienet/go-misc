package locker

import (
	"errors"
	"sync"
)

type lockEntry struct {
	mu       *sync.Mutex
	refCount int
}

type ResourceLocker struct {
	globalMu sync.Mutex
	locks    sync.Map
	pool     sync.Pool
}

func NewResourceLocker() *ResourceLocker {
	return &ResourceLocker{
		pool: sync.Pool{
			New: func() any {
				return &lockEntry{
					mu: &sync.Mutex{},
				}
			},
		},
	}
}

func (rl *ResourceLocker) Lock(key string) {
	rl.globalMu.Lock()

	entry, loaded := rl.locks.Load(key)
	if !loaded {
		entry = rl.pool.Get()
		entry.(*lockEntry).refCount = 0
		rl.locks.Store(key, entry)
	}

	entryPtr := entry.(*lockEntry)
	entryPtr.refCount++

	rl.globalMu.Unlock()
	entryPtr.mu.Lock()
}

func (rl *ResourceLocker) Unlock(key string) error {
	entry, exists := rl.locks.Load(key)
	if !exists {
		return errors.New("unlocking a non-locked resource")
	}

	entryPtr := entry.(*lockEntry)
	entryPtr.mu.Unlock()
	rl.globalMu.Lock()
	defer rl.globalMu.Unlock()

	entryPtr.refCount--
	if entryPtr.refCount == 0 {
		rl.locks.Delete(key)

		entryPtr.refCount = 0
		rl.pool.Put(entry)
	}
	
	return nil
}

func (rl *ResourceLocker) TryLock(key string) bool {
	rl.globalMu.Lock()
	defer rl.globalMu.Unlock()

	// 尝试加载现有条目
	entry, loaded := rl.locks.Load(key)
	if !loaded {
		// 从池中获取新条目
		entry = rl.pool.Get()
		entry.(*lockEntry).refCount = 0
		rl.locks.Store(key, entry)
	}

	entryPtr := entry.(*lockEntry)

	// 尝试获取锁
	if !entryPtr.mu.TryLock() {
		return false
	}

	entryPtr.refCount++
	return true
}
