package locker

import "sync"

var empty = &emptyLocker{}

type Locker struct {
	mu locker
}

func (l *Locker) Synchronize() *Locker {
	if l.mu == nil || l.mu == empty {
		l.mu = &sync.Mutex{}
	}

	return l
}

func (l *Locker) Lock() {
	if l.mu == nil {
		return
	}

	l.mu.Lock()
}

func (l *Locker) Unlock() {
	if l.mu == nil {
		return
	}

	l.mu.Unlock()
}

func (l *Locker) TryLock() bool {
	if l.mu == nil {
		return true
	}

	return l.mu.TryLock()
}

type RWLocker struct {
	mu rwLocker
}

func (w *RWLocker) Synchronize() *RWLocker {
	if w.mu == nil || w.mu == empty {
		w.mu = &sync.RWMutex{}
	}

	return w
}

func (w *RWLocker) Lock() {
	if w.mu == nil {
		return
	}

	w.mu.Lock()
}

func (w *RWLocker) TryLock() bool {
	if w.mu == nil {
		return true
	}

	return w.mu.TryLock()
}

func (w *RWLocker) Unlock() {
	if w.mu == nil {
		return
	}

	w.mu.Unlock()
}

func (w *RWLocker) RLock() {
	if w.mu == nil {
		return
	}

	w.mu.RLock()
}

func (w *RWLocker) TryRLock() bool {
	if w.mu == nil {
		return true
	}

	return w.mu.TryRLock()
}

func (w *RWLocker) RUnlock() {
	if w.mu == nil {
		return
	}
	w.mu.RUnlock()
}
