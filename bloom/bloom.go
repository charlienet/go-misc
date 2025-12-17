package bloom

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math"

	"github.com/bits-and-blooms/bitset"
	"github.com/charlienet/gadget/redis"

	sredis "github.com/redis/go-redis/v9"
)

type BloomFilter interface {
	// Add 添加元素到布隆过滤器
	Add(ctx context.Context, data []byte) error
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
		k:     k,
		m:     m,
		seed1: 0xdeadbeef,
		seed2: 0xcafebabe,
	}

	if o.redis != nil {
		return &redisBloomFilter{
			baseBloomFilter: baseBloomFilter,
			rdb:             o.redis,
		}
	}

	return &localBloomFilter{
		baseBloomFilter: baseBloomFilter,
		bitset:          bitset.New(m),
	}
}

type baseBloomFilter struct {
	m     uint // 位数组大小
	k     uint // 哈希函数数量
	seed1 uint64
	seed2 uint64
}

type redisBloomFilter struct {
	baseBloomFilter
	key string
	rdb redis.Client
}

type localBloomFilter struct {
	baseBloomFilter
	bitset *bitset.BitSet
}

func (bf *redisBloomFilter) Add(ctx context.Context, data []byte) error {
	locations := bf.getLocations(data)

	pipe := bf.rdb.Pipeline()
	for _, location := range locations {
		pipe.SetBit(ctx, bf.key, int64(location), 1)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (bf *redisBloomFilter) Test(ctx context.Context, data []byte) (bool, error) {
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

func (bf *localBloomFilter) Add(ctx context.Context, data []byte) error {
	locations := bf.getLocations(data)

	for _, location := range locations {
		bf.bitset.Set(location)
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
	h1, h2 := hash(data, bf.seed1, bf.seed2)

	locations := make([]uint, bf.k)
	u64m := uint64(bf.m)

	for i := uint(0); i < bf.k; i++ {
		location := (h1 + uint64(i)*h2) % u64m
		locations[i] = uint(location)
	}

	return locations
}

// hash 哈希函数，返回两个哈希值（双重哈希法）
func hash(data []byte, seed1, seed2 uint64) (uint64, uint64) {
	// 使用SHA256哈希，确保良好的分布性
	h := sha256.New()

	// 写入第一个种子
	binary.Write(h, binary.LittleEndian, seed1)
	// 写入数据
	h.Write(data)
	hash1 := h.Sum(nil)

	// 重新初始化哈希，使用第二个种子
	h.Reset()
	binary.Write(h, binary.LittleEndian, seed2)
	h.Write(data)
	hash2 := h.Sum(nil)

	// 取前8字节作为哈希值
	return binary.LittleEndian.Uint64(hash1[:8]), binary.LittleEndian.Uint64(hash2[:8])
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
