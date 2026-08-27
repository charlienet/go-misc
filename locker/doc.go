/*
Package locker 提供多种锁实现，适用于不同的并发控制场景。

主要功能：
- ResourceLocker: 基于资源键的互斥锁，允许多个 goroutine 对不同资源进行并发访问
- ChanSourceLocker: 基于 Channel 的锁实现，支持等待通知机制
- Locker/RWLocker: 支持懒初始化的锁实现，可以根据需要启用同步

导出类型和函数：
- ResourceLocker: 基于键值的资源锁
  - NewResourceLocker(): 创建新的资源锁实例
  - Lock(key string): 锁定指定键的资源
  - Unlock(key string): 解锁指定键的资源
  - TryLock(key string) bool: 尝试锁定资源（非阻塞）

- ChanSourceLocker: Channel 源锁
  - NewChanSourceLocker(): 创建新的 Channel 源锁实例
  - Lock(key string) (ok bool, ch <-chan int): 锁定资源并返回通知通道
  - Unlock(key string): 解锁资源

- Locker: 懒初始化的互斥锁
  - Synchronize(): 启用同步功能
  - Lock()/Unlock(): 加锁/解锁
  - TryLock(): 尝试加锁（非阻塞）

- RWLocker: 懒初始化的读写锁
  - Synchronize(): 启用同步功能
  - Lock()/Unlock(): 写锁操作
  - RLock()/RUnlock(): 读锁操作
  - TryLock()/TryRLock(): 尝试获取锁（非阻塞）

使用示例：

	// 示例1: 使用 ResourceLocker 控制对不同资源的并发访问
	rl := locker.NewResourceLocker()
	
	// 并发访问不同资源（不会相互阻塞）
	go func() {
		rl.Lock("resource1")
		// 处理 resource1
		rl.Unlock("resource1")
	}()
	
	go func() {
		rl.Lock("resource2")
		// 处理 resource2
		rl.Unlock("resource2")
	}()

	// 示例2: 使用 ChanSourceLocker 实现等待通知
	locker := locker.NewChanSourceLocker()
	
	ok, ch := locker.Lock("key1")
	if ok {
		// 第一次锁定成功
		go func() {
			// 做一些工作...
			locker.Unlock("key1") // 释放锁并通知等待者
		}()
	} else {
		// 锁已被占用，等待解锁
		<-ch // 等待解锁通知
	}

	// 示例3: 使用懒初始化锁
	l := &locker.Locker{}
	// 初始状态，Lock/Unlock 是空操作
	l.Lock()
	l.Unlock()
	
	// 调用 Synchronize() 后启用实际锁功能
	l.Synchronize()
	l.Lock()
	// 执行临界区代码
	l.Unlock()

适用场景：
- ResourceLocker: 适合需要对多个不同资源进行细粒度并发控制的场景
- ChanSourceLocker: 适合需要等待通知机制的锁场景
- Locker/RWLocker: 适合可选的同步场景，当不确定是否需要锁时使用

注意事项：
- ResourceLocker 的 Unlock 操作必须与 Lock 配对，且使用相同的键
- ChanSourceLocker 的 Lock 返回的通道可用于等待解锁事件
- 懒初始化锁在未调用 Synchronize() 前不会提供同步保证
*/
package locker