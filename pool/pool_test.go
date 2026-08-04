package pool

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPool_Basic(t *testing.T) {
	p := New[int](10, func() int { return 0 })

	// Get 应该返回默认值
	v := p.Get()
	assert.Equal(t, 0, v)

	// Put 后再 Get
	p.Put(42)
	v = p.Get()
	assert.Equal(t, 42, v)
}

func TestPool_Overflow(t *testing.T) {
	p := New[int](2, func() int { return 99 })

	// 填满池
	p.Put(1)
	p.Put(2)
	p.Put(3) // 超出容量，应该被丢弃

	// 池中的值
	v1 := p.Get()
	v2 := p.Get()
	assert.Contains(t, []int{1, 2}, v1)
	assert.Contains(t, []int{1, 2}, v2)

	// 第三个 Get 应该返回默认值
	v3 := p.Get()
	assert.Equal(t, 99, v3)
}

func TestPool_Concurrent(t *testing.T) {
	p := New[int](10, func() int { return 0 })
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			p.Put(val)
			_ = p.Get()
		}(i)
	}

	wg.Wait()
}

func TestBufferPool_NewAllocatedBufferPool(t *testing.T) {
	bp := NewAllocatedBufferPool(4, 1024)
	assert.NotNil(t, bp)

	b := bp.Get()
	assert.NotNil(t, b)
	assert.Equal(t, 1024, cap(b.Bytes()))
}

func TestBufferPool_NewBufferPool(t *testing.T) {
	bp := NewBufferPool(4)
	assert.NotNil(t, bp)

	b := bp.Get()
	assert.NotNil(t, b)
}

func TestBufferPool_PutReset(t *testing.T) {
	bp := NewAllocatedBufferPool(4, 16)

	b := bp.Get()
	b.WriteString("hello world")
	assert.Greater(t, b.Len(), 0)

	bp.Put(b)

	// 取回后应该是空的（Reset）
	b2 := bp.Get()
	assert.Equal(t, 0, b2.Len())
}

func TestBufferPool_ShrinkLargeBuffer(t *testing.T) {
	bp := NewAllocatedBufferPool(4, 16)

	b := bp.Get()
	// 增长到超过分配大小
	b.Grow(1024)
	assert.Greater(t, cap(b.Bytes()), 16)

	bp.Put(b)

	// 取回后应该被替换为新的合适大小的 buffer
	b2 := bp.Get()
	assert.LessOrEqual(t, cap(b2.Bytes()), 32) // 应该是新分配的
}

func TestBufferPool_Concurrent(t *testing.T) {
	bp := NewAllocatedBufferPool(10, 64)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b := bp.Get()
			b.WriteString("test data")
			_ = b.Bytes()
			bp.Put(b)
		}()
	}

	wg.Wait()
}

func TestPool_GenericTypes(t *testing.T) {
	// 测试不同类型的池
	stringPool := New[string](5, func() string { return "default" })
	s := stringPool.Get()
	assert.Equal(t, "default", s)
	stringPool.Put("hello")
	s = stringPool.Get()
	assert.Equal(t, "hello", s)

	bufPool := New[*bytes.Buffer](5, func() *bytes.Buffer { return bytes.NewBuffer(nil) })
	buf := bufPool.Get()
	assert.NotNil(t, buf)
	buf.WriteString("test")
	bufPool.Put(buf)
	buf2 := bufPool.Get()
	assert.Equal(t, "test", buf2.String())
}
