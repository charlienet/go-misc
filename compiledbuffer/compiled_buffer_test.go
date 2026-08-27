package compiledbuffer

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompiledBuffer_PutAndGet(t *testing.T) {
	cb := NewCompiledBuffer(func(s string) (int, error) {
		return len(s), nil
	})

	// 测试 Put
	result, err := cb.Put("hello")
	assert.NoError(t, err)
	assert.Equal(t, 5, result)

	// 测试 Get（已缓存）
	result, err = cb.Get("hello")
	assert.NoError(t, err)
	assert.Equal(t, 5, result)

	// 测试 Get（未缓存，自动编译）
	result, err = cb.Get("world!")
	assert.NoError(t, err)
	assert.Equal(t, 6, result)
}

func TestCompiledBuffer_CompileError(t *testing.T) {
	cb := NewCompiledBuffer(func(s string) (int, error) {
		if s == "error" {
			return 0, errors.New("compile error")
		}
		return len(s), nil
	})

	// 测试编译错误
	result, err := cb.Put("error")
	assert.Error(t, err)
	assert.Equal(t, 0, result)

	// 错误结果不应被缓存
	result, err = cb.Get("error")
	assert.Error(t, err)
	assert.Equal(t, 0, result)
}

func TestCompiledBuffer_Clear(t *testing.T) {
	cb := NewCompiledBuffer(func(s string) (int, error) {
		return len(s), nil
	})

	// 添加一些数据
	cb.Put("hello")
	cb.Put("world")

	// 清空
	cb.Clear()

	// 再次获取应该重新编译
	result, err := cb.Get("hello")
	assert.NoError(t, err)
	assert.Equal(t, 5, result)
}

func TestCompiledBuffer_Concurrent(t *testing.T) {
	cb := NewCompiledBuffer(func(s string) (int, error) {
		return len(s), nil
	})

	var wg sync.WaitGroup
	concurrency := 100

	// 并发写入
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := "key"
			result, err := cb.Put(key)
			assert.NoError(t, err)
			assert.Equal(t, 3, result)
		}(i)
	}

	// 并发读取
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := "key"
			result, err := cb.Get(key)
			assert.NoError(t, err)
			assert.Equal(t, 3, result)
		}(i)
	}

	wg.Wait()
}
