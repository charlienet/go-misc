package locker

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResourceLocker_Basic(t *testing.T) {
	rl := NewResourceLocker()

	rl.Lock("key1")
	rl.Unlock("key1")
}

func TestResourceLocker_MutualExclusion(t *testing.T) {
	rl := NewResourceLocker()
	var counter int32
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.Lock("counter")
			atomic.AddInt32(&counter, 1)
			rl.Unlock("counter")
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(100), counter)
}

func TestResourceLocker_MultipleKeys(t *testing.T) {
	rl := NewResourceLocker()
	var wg sync.WaitGroup

	results := make(map[string]int32)
	var mu sync.Mutex

	for k := 0; k < 5; k++ {
		key := string(rune('a' + k))
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(key string) {
				defer wg.Done()
				rl.Lock(key)
				mu.Lock()
				results[key]++
				mu.Unlock()
				rl.Unlock(key)
			}(key)
		}
	}

	wg.Wait()
	for k := 0; k < 5; k++ {
		key := string(rune('a' + k))
		assert.Equal(t, int32(20), results[key])
	}
}

func TestResourceLocker_TryLock(t *testing.T) {
	rl := NewResourceLocker()

	// 第一次 TryLock 应该成功
	ok := rl.TryLock("key1")
	assert.True(t, ok)

	// 第二次 TryLock 应该失败（已被锁定）
	done := make(chan bool, 1)
	go func() {
		ok := rl.TryLock("key1")
		done <- ok
	}()

	select {
	case result := <-done:
		assert.False(t, result)
	case <-time.After(time.Second):
		t.Fatal("TryLock should not block")
	}

	rl.Unlock("key1")
}

func TestResourceLocker_UnlockNonExistent(t *testing.T) {
	rl := NewResourceLocker()

	// 解锁不存在的资源不应导致 panic 或错误
	rl.Unlock("nonexistent") // 应该静默返回
}

func TestChanSourceLocker_Basic(t *testing.T) {
	locker := NewChanSourceLocker()

	ok, ch := locker.Lock("key1")
	assert.True(t, ok)
	assert.NotNil(t, ch)

	// 重复锁定同一个 key 应该失败
	ok2, ch2 := locker.Lock("key1")
	assert.False(t, ok2)
	assert.NotNil(t, ch2)

	locker.Unlock("key1")
}

func TestChanSourceLocker_WaitForUnlock(t *testing.T) {
	locker := NewChanSourceLocker()

	ok, ch := locker.Lock("key1")
	assert.True(t, ok)

	// 等待解锁
	done := make(chan bool, 1)
	go func() {
		<-ch
		done <- true
	}()

	locker.Unlock("key1")

	select {
	case <-done:
		// 成功
	case <-time.After(time.Second):
		t.Fatal("should have been notified of unlock")
	}
}

func TestChanSourceLocker_MultipleKeys(t *testing.T) {
	locker := NewChanSourceLocker()

	ok1, _ := locker.Lock("key1")
	ok2, _ := locker.Lock("key2")
	ok3, _ := locker.Lock("key3")

	assert.True(t, ok1)
	assert.True(t, ok2)
	assert.True(t, ok3)

	locker.Unlock("key1")
	locker.Unlock("key2")
	locker.Unlock("key3")
}

func TestLocker_Synchronize(t *testing.T) {
	l := &Locker{}

	// 未调用 Synchronize 时，Lock/Unlock 应该是空操作
	l.Lock()
	l.Unlock()

	// 调用 Synchronize 后，应该变成真正的锁
	l.Synchronize()
	l.Lock()
	l.Unlock()
}

func TestLocker_TryLock(t *testing.T) {
	l := &Locker{}

	// 未同步时 TryLock 应该返回 true
	assert.True(t, l.TryLock())

	l.Synchronize()
	assert.True(t, l.TryLock())
	l.Unlock()
}

func TestRWLocker_Synchronize(t *testing.T) {
	l := &RWLocker{}

	// 未调用 Synchronize 时
	l.Lock()
	l.Unlock()
	l.RLock()
	l.RUnlock()

	// 调用 Synchronize 后
	l.Synchronize()
	l.Lock()
	l.Unlock()
	l.RLock()
	l.RUnlock()
}

func TestRWLocker_TryLock(t *testing.T) {
	l := &RWLocker{}

	assert.True(t, l.TryLock())
	l.Unlock()

	l.Synchronize()
	assert.True(t, l.TryLock())
	l.Unlock()
}

func TestRWLocker_TryRLock(t *testing.T) {
	l := &RWLocker{}

	assert.True(t, l.TryRLock())
	l.RUnlock()

	l.Synchronize()
	assert.True(t, l.TryRLock())
	l.RUnlock()
}

func TestRWLocker_ConcurrentRead(t *testing.T) {
	l := &RWLocker{}
	l.Synchronize()

	var counter int32
	var wg sync.WaitGroup

	// 多个读锁应该可以并发
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.RLock()
			atomic.AddInt32(&counter, 1)
			time.Sleep(10 * time.Millisecond)
			l.RUnlock()
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(10), counter)
}

func TestEmptyLocker(t *testing.T) {
	l := &emptyLocker{}

	// 所有操作应该是空操作
	l.Lock()
	l.Unlock()
	l.RLock()
	l.RUnlock()
	assert.True(t, l.TryLock())
	assert.True(t, l.TryRLock())
}
