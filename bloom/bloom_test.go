package bloom

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/charlienet/gadget/redis"
	"github.com/stretchr/testify/assert"
)

func TestLocalBloomFilter(t *testing.T) {
	// 创建一个预期插入 1000 个元素，误判率为 0.01 的布隆过滤器
	bf := New(1000, 0.01)

	ctx := context.Background()

	// 测试添加和查询
	data1 := []byte("hello")
	data2 := []byte("world")
	data3 := []byte("not_exist")

	// 添加元素
	err := bf.Add(ctx, data1)
	assert.NoError(t, err)

	err = bf.Add(ctx, data2)
	assert.NoError(t, err)

	// 测试存在的元素
	exists, err := bf.Test(ctx, data1)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = bf.Test(ctx, data2)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 测试不存在的元素（可能误判）
	exists, err = bf.Test(ctx, data3)
	assert.NoError(t, err)
	// 不存在的元素可能返回 false（正确）或 true（误判）
	// 这里只验证不报错

	// 测试 Exists 方法（Test 的别名）
	exists, err = bf.Exists(ctx, data1)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 测试 Cap 和 K
	assert.Greater(t, bf.Cap(), uint(0))
	assert.Greater(t, bf.K(), uint(0))

	// 测试误判率估计
	rate := bf.EstimateFalsePositiveRate(1000)
	assert.Greater(t, rate, 0.0)
	assert.Less(t, rate, 1.0)

	// 测试清空
	err = bf.Clear(ctx)
	assert.NoError(t, err)

	// 清空后查询应该返回 false
	exists, err = bf.Test(ctx, data1)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestLocalBloomFilterAddBatch(t *testing.T) {
	bf := New(1000, 0.01)
	ctx := context.Background()

	// 批量添加
	data := make([][]byte, 100)
	for i := range data {
		data[i] = []byte(fmt.Sprintf("batch_item_%d", i))
	}

	err := bf.AddBatch(ctx, data)
	assert.NoError(t, err)

	// 验证所有元素都存在
	for i, d := range data {
		exists, err := bf.Test(ctx, d)
		assert.NoError(t, err)
		assert.True(t, exists, "item %d should exist after AddBatch", i)
	}

	// 验证不存在的元素
	exists, err := bf.Test(ctx, []byte("not_in_batch"))
	assert.NoError(t, err)
	assert.False(t, exists)

	// 空批量添加
	err = bf.AddBatch(ctx, nil)
	assert.NoError(t, err)

	err = bf.AddBatch(ctx, [][]byte{})
	assert.NoError(t, err)
}

func TestLocalBloomFilterAddBatchLarge(t *testing.T) {
	// 测试超过 maxBatchPerExec 的批量添加（验证分片逻辑）
	bf := New(20000, 0.01)
	ctx := context.Background()

	data := make([][]byte, 12000)
	for i := range data {
		data[i] = []byte(fmt.Sprintf("large_batch_%d", i))
	}

	err := bf.AddBatch(ctx, data)
	assert.NoError(t, err)

	// 抽样验证
	for i := 0; i < len(data); i += 100 {
		exists, err := bf.Test(ctx, data[i])
		assert.NoError(t, err)
		assert.True(t, exists, "item %d should exist", i)
	}
}

func TestOptimalMK(t *testing.T) {
	tests := []struct {
		n uint
		p float64
	}{
		{100, 0.01},
		{1000, 0.001},
		{10000, 0.0001},
		{1, 0.1},
	}

	for _, tt := range tests {
		m, k := optimalMK(tt.n, tt.p)
		assert.GreaterOrEqual(t, m, uint(64), "m should be at least 64")
		assert.GreaterOrEqual(t, k, uint(1), "k should be at least 1")
		assert.LessOrEqual(t, k, uint(50), "k should be at most 50")
	}
}

func TestBloomFilterFalsePositiveRate(t *testing.T) {
	// 测试误判率是否在预期范围内
	n := uint(1000)
	fpp := 0.01
	bf := New(n, fpp)

	ctx := context.Background()

	// 添加 n 个元素
	for i := 0; i < int(n); i++ {
		data := []byte{byte(i), byte(i >> 8)}
		err := bf.Add(ctx, data)
		assert.NoError(t, err)
	}

	// 测试 10000 个不存在的元素，统计误判数
	falsePositives := 0
	testCount := 10000
	for i := 10000; i < 10000+testCount; i++ {
		data := []byte{byte(i), byte(i >> 8)}
		exists, err := bf.Test(ctx, data)
		assert.NoError(t, err)
		if exists {
			falsePositives++
		}
	}

	// 实际误判率
	actualRate := float64(falsePositives) / float64(testCount)
	// 允许一定的误差范围
	assert.Less(t, actualRate, fpp*3, "actual false positive rate should be within acceptable range")
}

// newMiniRedisClient 创建基于 miniredis 的测试客户端
func newMiniRedisClient(t *testing.T) (*miniredis.Miniredis, redis.Client) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)

	rdb := redis.New(redis.WithAddr(mr.Addr()))
	return mr, rdb
}

// TestRedisBloomFilterBitmap 测试 Redis bitmap 路径（useNativeBF=false）
func TestRedisBloomFilterBitmap(t *testing.T) {
	mr, rdb := newMiniRedisClient(t)
	defer mr.Close()
	defer rdb.Close()

	ctx := context.Background()
	key := "test:bloom:bitmap"

	// 创建基于 bitmap 的布隆过滤器
	bf := New(1000, 0.01, WithRedis(rdb, key))

	// 测试添加和查询
	data1 := []byte("hello")
	data2 := []byte("world")
	data3 := []byte("not_exist")

	// 添加元素
	err := bf.Add(ctx, data1)
	assert.NoError(t, err)

	err = bf.Add(ctx, data2)
	assert.NoError(t, err)

	// 测试存在的元素
	exists, err := bf.Test(ctx, data1)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = bf.Test(ctx, data2)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 测试不存在的元素
	exists, err = bf.Test(ctx, data3)
	assert.NoError(t, err)
	// 不存在的元素可能返回 false（正确）或 true（误判）

	// 测试 Exists 方法
	exists, err = bf.Exists(ctx, data1)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 测试 Cap 和 K
	assert.Greater(t, bf.Cap(), uint(0))
	assert.Greater(t, bf.K(), uint(0))

	// 测试清空
	err = bf.Clear(ctx)
	assert.NoError(t, err)

	// 清空后查询应该返回 false
	exists, err = bf.Test(ctx, data1)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestRedisBloomFilterBitmapAddBatch 测试 Redis bitmap 路径的批量添加
func TestRedisBloomFilterBitmapAddBatch(t *testing.T) {
	mr, rdb := newMiniRedisClient(t)
	defer mr.Close()
	defer rdb.Close()

	ctx := context.Background()
	key := "test:bloom:bitmap:batch"

	bf := New(1000, 0.01, WithRedis(rdb, key))

	// 批量添加
	data := make([][]byte, 100)
	for i := range data {
		data[i] = []byte(fmt.Sprintf("batch_item_%d", i))
	}

	err := bf.AddBatch(ctx, data)
	assert.NoError(t, err)

	// 验证所有元素都存在
	for i, d := range data {
		exists, err := bf.Test(ctx, d)
		assert.NoError(t, err)
		assert.True(t, exists, "item %d should exist after AddBatch", i)
	}

	// 验证不存在的元素
	exists, err := bf.Test(ctx, []byte("not_in_batch"))
	assert.NoError(t, err)
	assert.False(t, exists)

	// 空批量添加
	err = bf.AddBatch(ctx, nil)
	assert.NoError(t, err)

	err = bf.AddBatch(ctx, [][]byte{})
	assert.NoError(t, err)
}

// TestRedisBloomFilterBitmapAddBatchLarge 测试大批量添加（验证分片逻辑）
func TestRedisBloomFilterBitmapAddBatchLarge(t *testing.T) {
	mr, rdb := newMiniRedisClient(t)
	defer mr.Close()
	defer rdb.Close()

	ctx := context.Background()
	key := "test:bloom:bitmap:batch:large"

	bf := New(20000, 0.01, WithRedis(rdb, key))

	data := make([][]byte, 12000)
	for i := range data {
		data[i] = []byte(fmt.Sprintf("large_batch_%d", i))
	}

	err := bf.AddBatch(ctx, data)
	assert.NoError(t, err)

	// 抽样验证
	for i := 0; i < len(data); i += 100 {
		exists, err := bf.Test(ctx, data[i])
		assert.NoError(t, err)
		assert.True(t, exists, "item %d should exist", i)
	}
}

// TestRedisBloomFilterNativeBF 测试 Redis native BF 路径（需要 Redis Stack）
// 运行方式: REDIS_STACK_URL=redis://localhost:6379 go test -run TestRedisBloomFilterNativeBF
func TestRedisBloomFilterNativeBF(t *testing.T) {
	url := os.Getenv("REDIS_STACK_URL")
	if url == "" {
		t.Skip("REDIS_STACK_URL not set, skipping native BF test")
	}

	rdb, err := redis.NewWithUrl(url)
	assert.NoError(t, err)
	defer rdb.Close()

	ctx := context.Background()
	key := fmt.Sprintf("test:bloom:native:%d", time.Now().UnixNano())

	// 创建基于 native BF 的布隆过滤器
	bf := New(1000, 0.01, WithRedis(rdb, key))
	defer rdb.Del(ctx, key)

	// 测试添加和查询
	data1 := []byte("hello")
	data2 := []byte("world")
	data3 := []byte("not_exist")

	err = bf.Add(ctx, data1)
	assert.NoError(t, err)

	err = bf.Add(ctx, data2)
	assert.NoError(t, err)

	exists, err := bf.Test(ctx, data1)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = bf.Test(ctx, data2)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = bf.Test(ctx, data3)
	assert.NoError(t, err)

	exists, err = bf.Exists(ctx, data1)
	assert.NoError(t, err)
	assert.True(t, exists)

	assert.Greater(t, bf.Cap(), uint(0))
	assert.Greater(t, bf.K(), uint(0))

	err = bf.Clear(ctx)
	assert.NoError(t, err)

	exists, err = bf.Test(ctx, data1)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestRedisBloomFilterNativeBFAddBatch 测试 native BF 路径的批量添加
func TestRedisBloomFilterNativeBFAddBatch(t *testing.T) {
	url := os.Getenv("REDIS_STACK_URL")
	if url == "" {
		t.Skip("REDIS_STACK_URL not set, skipping native BF batch test")
	}

	rdb, err := redis.NewWithUrl(url)
	assert.NoError(t, err)
	defer rdb.Close()

	ctx := context.Background()
	key := fmt.Sprintf("test:bloom:native:batch:%d", time.Now().UnixNano())

	bf := New(1000, 0.01, WithRedis(rdb, key))
	defer rdb.Del(ctx, key)

	data := make([][]byte, 100)
	for i := range data {
		data[i] = []byte(fmt.Sprintf("batch_item_%d", i))
	}

	err = bf.AddBatch(ctx, data)
	assert.NoError(t, err)

	for i, d := range data {
		exists, err := bf.Test(ctx, d)
		assert.NoError(t, err)
		assert.True(t, exists, "item %d should exist after AddBatch", i)
	}

	exists, err := bf.Test(ctx, []byte("not_in_batch"))
	assert.NoError(t, err)
	assert.False(t, exists)

	err = bf.AddBatch(ctx, nil)
	assert.NoError(t, err)

	err = bf.AddBatch(ctx, [][]byte{})
	assert.NoError(t, err)
}
