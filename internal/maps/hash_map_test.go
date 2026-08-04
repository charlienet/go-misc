package maps

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHashMap(t *testing.T) {
	m := NewHashMap[map[string]int]()
	assert.NotNil(t, m)

	// 带初始值
	initial := map[string]int{"a": 1, "b": 2}
	m2 := NewHashMap[map[string]int](initial)
	v, ok := m2.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, v)
}

func TestHashMap_SetGet(t *testing.T) {
	m := NewHashMap[map[string]int]()

	m.Set("key1", 100)
	m.Set("key2", 200)

	v, ok := m.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, 100, v)

	v, ok = m.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, 200, v)

	// 不存在的 key
	_, ok = m.Get("nonexistent")
	assert.False(t, ok)
}

func TestHashMap_Overwrite(t *testing.T) {
	m := NewHashMap[map[string]string]()

	m.Set("key", "value1")
	v, _ := m.Get("key")
	assert.Equal(t, "value1", v)

	m.Set("key", "value2")
	v, _ = m.Get("key")
	assert.Equal(t, "value2", v)
}

func TestHashMap_Synchronize(t *testing.T) {
	m := NewHashMap[map[string]int]().Synchronize()
	assert.NotNil(t, m)

	// 并发读写测试
	var wg sync.WaitGroup
	concurrency := 100

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := "key"
			m.Set(key, idx)
			m.Get(key)
		}(i)
	}

	wg.Wait()
}

func TestHashMap_IntKey(t *testing.T) {
	m := NewHashMap[map[int]string]()

	m.Set(1, "one")
	m.Set(2, "two")

	v, ok := m.Get(1)
	assert.True(t, ok)
	assert.Equal(t, "one", v)

	v, ok = m.Get(2)
	assert.True(t, ok)
	assert.Equal(t, "two", v)
}

func TestHashMap_EmptyMap(t *testing.T) {
	m := NewHashMap[map[string]int]()

	_, ok := m.Get("anything")
	assert.False(t, ok)
}

func TestHashMap_MultipleInitialValues(t *testing.T) {
	m1 := map[string]int{"a": 1}
	m2 := map[string]int{"b": 2, "c": 3}

	m := NewHashMap[map[string]int](m1, m2)

	v, ok := m.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, v)

	v, ok = m.Get("b")
	assert.True(t, ok)
	assert.Equal(t, 2, v)

	v, ok = m.Get("c")
	assert.True(t, ok)
	assert.Equal(t, 3, v)
}
