package bloom

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/bits-and-blooms/bitset"
	"github.com/charlienet/gadget/redis"
	"github.com/spaolacci/murmur3"

	sredis "github.com/redis/go-redis/v9"
)

type BloomFilter interface {
	// Add 添加元素到布隆过滤器
	Add(ctx context.Context, data []byte) error
	// AddBatch 批量添加元素到布隆过滤器，内部单管道批量执行
	AddBatch(ctx context.Context, data [][]byte) error
	// Test 检查元素是否可能存在
	Test(ctx context.Context, data []byte) (bool, error)
	// Exists 检查元素是否存在（Test的别名）
	Exists(ctx context.Context, data []byte) (bool, error)
	// Clear 清空布隆过滤器
	Clear(ctx context.Context) error
	// Cap 返回布隆过滤器的容量
	Cap() uint
	// K 返回哈希函数的数量
	K() uint
	// EstimateFalsePositiveRate 估计误判率
	EstimateFalsePositiveRate(n uint) float64
}

type opt struct {
	n     uint
	p     float64
	key   string
	redis redis.Client
}

type Option func(*opt)

func WithRedis(redis redis.Client, key string) Option {
	return func(o *opt) {
		o.redis = redis
		o.key = key
	}
}

func New(expectedInsertions uint, fpp float64, opts ...Option) BloomFilter {
	o := &opt{
		n: expectedInsertions,
		p: fpp,
	}

	for _, opt := range opts {
		opt(o)
	}

	m, k := optimalMK(o.n, o.p)

	baseBloomFilter := baseBloomFilter{
		expectedInsertions: expectedInsertions,
		fpp:                fpp,
		k:                  k,
		m:                  m,
		hashFuncs:          generateHashFunctions(k),
	}

	if o.redis != nil {
		useNativeBF := detectNativeBloomSupport(o.redis)

		filter := &redisBloomFilter{
			baseBloomFilter: baseBloomFilter,
			rdb:             o.redis,
			key:             o.key,
			useNativeBF:     useNativeBF,
		}

		filter.build(context.Background())

		return filter
	}

	return &localBloomFilter{
		baseBloomFilter: baseBloomFilter,
		bitset:          bitset.New(m),
	}
}

type baseBloomFilter struct {
	expectedInsertions uint
	fpp                float64
	m                  uint                         // 位数组大小
	k                  uint                         // 哈希函数数量
	hashFuncs          []func(data []byte) []uint64 // 哈希函数
}

type redisBloomFilter struct {
	baseBloomFilter
	key         string
	useNativeBF bool
	rdb         redis.Client
}

type localBloomFilter struct {
	baseBloomFilter
	bitset *bitset.BitSet
}

func (bf *redisBloomFilter) Add(ctx context.Context, data []byte) error {
	if bf.useNativeBF {
		return bf.rdb.BFAdd(ctx, bf.key, data).Err()
	}

	locations := bf.getLocations(data)

	pipe := bf.rdb.Pipeline()
	for _, location := range locations {
		pipe.SetBit(ctx, bf.key, int64(location), 1)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// maxBatchPerExec 单次管道批量写入的数据条数上限；每条含 k≈10 个 SETBIT，即单管道 ≤5 万命令
const maxBatchPerExec = 5000

func (bf *redisBloomFilter) AddBatch(ctx context.Context, data [][]byte) error {
	for start := 0; start < len(data); start += maxBatchPerExec {
		end := start + maxBatchPerExec
		if end > len(data) {
			end = len(data)
		}
		if err := bf.addBatchOnce(ctx, data[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (bf *redisBloomFilter) addBatchOnce(ctx context.Context, data [][]byte) error {
	if bf.useNativeBF {
		elements := make([]interface{}, len(data))
		for i, d := range data {
			elements[i] = d
		}
		return bf.rdb.BFMAdd(ctx, bf.key, elements...).Err()
	}

	pipe := bf.rdb.Pipeline()
	for _, d := range data {
		for _, location := range bf.getLocations(d) {
			pipe.SetBit(ctx, bf.key, int64(location), 1)
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (bf *redisBloomFilter) Test(ctx context.Context, data []byte) (bool, error) {
	if bf.useNativeBF {
		return bf.rdb.BFExists(context.Background(), bf.key, data).Result()
	}

	locations := bf.getLocations(data)

	pipe := bf.rdb.Pipeline()
	cmds := make([]*sredis.IntCmd, len(locations))

	for i, location := range locations {
		cmds[i] = pipe.GetBit(ctx, bf.key, int64(location))
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	// 检查所有位是否都为1
	for _, cmd := range cmds {
		if cmd.Val() == 0 {
			return false, nil
		}
	}

	return true, nil
}

func (bf *redisBloomFilter) Exists(ctx context.Context, data []byte) (bool, error) {
	return bf.Test(ctx, data)
}

func (bf *redisBloomFilter) Clear(ctx context.Context) error {
	if bf.useNativeBF {
		if err := bf.rdb.Del(ctx, bf.key).Err(); err != nil {
			return err
		}

		return bf.rdb.BFReserve(ctx, bf.key, bf.fpp, int64(bf.expectedInsertions)).Err()
	}

	return bf.rdb.Del(ctx, bf.key).Err()
}

func (bf *redisBloomFilter) Cap() uint {
	return bf.m
}

func (bf *redisBloomFilter) K() uint {
	return bf.k
}

func (bf *redisBloomFilter) EstimateFalsePositiveRate(n uint) float64 {
	// 误判率计算公式: (1 - e^(-k*n/m))^k
	kf := float64(bf.k)
	nf := float64(n)
	mf := float64(bf.m)

	return math.Pow(1-math.Exp(-kf*nf/mf), kf)
}

func (bf *redisBloomFilter) build(ctx context.Context) error {
	if bf.useNativeBF {
		exist, err := bf.rdb.Exists(ctx, bf.key).Result()
		if err != nil {
			return err
		}

		if exist == 0 {
			return bf.rdb.BFReserve(ctx, bf.key, bf.fpp, int64(bf.expectedInsertions)).Err()
		}
	}

	return nil
}

func (bf *localBloomFilter) Add(ctx context.Context, data []byte) error {
	locations := bf.getLocations(data)

	for _, location := range locations {
		bf.bitset.Set(location)
	}

	return nil
}

func (bf *localBloomFilter) AddBatch(ctx context.Context, data [][]byte) error {
	for _, d := range data {
		locations := bf.getLocations(d)
		for _, location := range locations {
			bf.bitset.Set(location)
		}
	}
	return nil
}

func (bf *localBloomFilter) Test(ctx context.Context, data []byte) (bool, error) {
	locations := bf.getLocations(data)

	for _, location := range locations {
		if !bf.bitset.Test(location) {
			return false, nil
		}
	}

	return true, nil
}

func (bf *localBloomFilter) Exists(ctx context.Context, data []byte) (bool, error) {
	return bf.Test(ctx, data)
}

func (bf *localBloomFilter) Clear(ctx context.Context) error {
	bf.bitset.ClearAll()
	return nil
}

func (bf *localBloomFilter) Cap() uint {
	return bf.m
}

func (bf *localBloomFilter) K() uint {
	return bf.k
}

func (bf *localBloomFilter) EstimateFalsePositiveRate(n uint) float64 {
	kf := float64(bf.k)
	nf := float64(n)
	mf := float64(bf.m)

	return math.Pow(1-math.Exp(-kf*nf/mf), kf)
}

func (bf *baseBloomFilter) getLocations(data []byte) []uint {
	locations := make([]uint, len(bf.hashFuncs))
	m64 := uint64(bf.m)

	for i, hashFunc := range bf.hashFuncs {
		hashes := hashFunc(data)

		// 使用双重哈希法：location = (h1 + i * h2) % m
		h1 := hashes[0]
		h2 := hashes[1]

		// 确保非零
		if h2 == 0 {
			h2 = 1
		}

		location := (h1 + uint64(i)*h2) % m64
		locations[i] = uint(location)
	}

	return locations
}

func optimalMK(n uint, p float64) (m, k uint) {
	// m = - (n * ln(p)) / (ln(2)^2)
	m = uint(math.Ceil(-float64(n) * math.Log(p) / (math.Ln2 * math.Ln2)))

	// k = (m / n) * ln(2)
	k = uint(math.Ceil(float64(m) / float64(n) * math.Ln2))

	// 确保最小值
	if m < 64 {
		m = 64
	}
	if k < 1 {
		k = 1
	}
	if k > 50 {
		k = 50
	}

	return
}

// 生成k个哈希函数
func generateHashFunctions(k uint) []func(data []byte) []uint64 {
	funcs := make([]func(data []byte) []uint64, k)

	for i := range k {
		seed := uint32(i)
		funcs[i] = func(data []byte) []uint64 {
			return murmur3Hash(data, seed)
		}
	}

	return funcs
}

func murmur3Hash(data []byte, seed uint32) []uint64 {
	// 使用murmur3生成多个哈希值
	h1, h2 := murmur3.Sum128WithSeed(data, seed)
	return []uint64{h1, h2}
}

func detectNativeBloomSupport(rdb redis.Client) bool {
	// 使用随机 key 避免与业务数据冲突
	testKey := fmt.Sprintf("__bloom_test_%d__", time.Now().UnixNano())
	testValue := "test_value"

	ctx := context.Background()
	defer rdb.Del(ctx, testKey) // 确保清理

	err := rdb.BFAdd(ctx, testKey, testValue).Err()

	return err == nil
}
