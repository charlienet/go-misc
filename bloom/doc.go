/*
Package bloom 提供布隆过滤器的实现，支持本地内存和 Redis 后端存储。

主要功能：
- 本地内存布隆过滤器实现
- Redis 后端支持（支持原生 RedisBloom 模块和 bitmap 方式）
- 支持批量添加操作
- 提供误判率估算功能

导出类型和函数：
- BloomFilter: 布隆过滤器接口
  - Add(ctx context.Context, data []byte): 添加单个元素
  - AddBatch(ctx context.Context, data [][]byte): 批量添加元素
  - Test(ctx context.Context, data []byte): 测试元素是否存在
  - Exists(ctx context.Context, data []byte): 检查元素是否存在（Test 的别名）
  - Clear(ctx context.Context): 清空过滤器
  - Cap(): 返回过滤器容量
  - K(): 返回哈希函数数量
  - EstimateFalsePositiveRate(n uint): 估算误判率

- Option: 配置选项类型
  - WithRedis(redis Client, key string): 指定 Redis 客户端和键名

- New(expectedInsertions uint, fpp float64, opts ...Option): 创建布隆过滤器实例

使用示例：

	// 示例1: 创建本地内存布隆过滤器
	bf := bloom.New(1000, 0.01) // 预期插入 1000 个元素，误判率 1%
	
	ctx := context.Background()
	
	// 添加元素
	err := bf.Add(ctx, []byte("user1"))
	if err != nil {
		log.Fatal(err)
	}
	
	// 测试元素是否存在
	exists, err := bf.Test(ctx, []byte("user1"))
	if err != nil {
		log.Fatal(err)
	}
	if exists {
		fmt.Println("user1 可能存在于集合中")
	}

	// 示例2: 批量添加元素
	data := [][]byte{
		[]byte("item1"),
		[]byte("item2"),
		[]byte("item3"),
	}
	err = bf.AddBatch(ctx, data)
	if err != nil {
		log.Fatal(err)
	}

	// 示例3: 使用 Redis 后端
	rdb := redis.New(redis.WithAddr("localhost:6379"))
	bf = bloom.New(10000, 0.001, bloom.WithRedis(rdb, "my_bloom_filter"))
	
	// 添加元素到 Redis
	err = bf.Add(ctx, []byte("redis_item"))
	if err != nil {
		log.Fatal(err)
	}

	// 示例4: 估算误判率
	rate := bf.EstimateFalsePositiveRate(5000) // 估算在已有 5000 个元素时的误判率
	fmt.Printf("误判率: %.4f\n", rate)

Redis 后端支持：
- 自动检测 Redis 是否支持原生 Bloom Filter 模块（RedisBloom）
- 如果支持原生模块，则使用 BF.ADD/BF.MADD 等原生命令
- 如果不支持，则使用 SETBIT/GETBIT 命令模拟布隆过滤器
- 支持大批量操作的自动分片（避免单次管道命令过多）

注意事项：
- 布隆过滤器存在误判（假阳性），但不会有假阴性
- 误判率越低，所需空间越大
- Redis 后端的性能优于本地内存，特别是对于大量数据
- 使用 Redis 时建议根据实际数据量调整 Redis 配置
- 批量添加操作会自动分片，单次最大处理 5000 个元素
*/
package bloom