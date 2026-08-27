/*
Package pool provides generic object pooling utilities to reduce garbage collection pressure.

The pool package implements efficient object pools for reusing objects, particularly useful for reducing allocations in high-frequency scenarios.

Exported Types and Functions:
  - Pool[T]: Generic object pool supporting any type T
  - New[T]: Creates a new generic Pool with specified size and factory function
  - (p *Pool[T]) Get(): Retrieves an object from the pool or creates a new one
  - (p *Pool[T]) Put(): Returns an object to the pool for reuse
  - BufferPool: Specialized buffer pool for bytes.Buffer objects
  - NewBufferPool: Creates a new BufferPool with specified size
  - NewAllocatedBufferPool: Creates a BufferPool with pre-allocated buffer capacity
  - (bp *BufferPool) Get(): Retrieves a buffer from the pool
  - (bp *BufferPool) Put(): Returns a buffer to the pool after resetting it

Example Usage:

Using generic Pool:
 type MyObject struct {
     Data string
 }
 
 factory := func() MyObject { return MyObject{} }
 p := pool.New[MyObject](10, factory)
 
 obj := p.Get()           // Get object from pool or create new
 // Use obj...
 p.Put(obj)               // Return object to pool

Using BufferPool:
 bp := pool.NewBufferPool(10)
 buf := bp.Get()          // Get buffer from pool
 buf.WriteString("test")
 // Use buf...
 bp.Put(buf)              // Return buffer to pool (automatically reset)

Capacity Limitations:
  - Both Pool and BufferPool have fixed channel-based capacity limits
  - When pool is full, Put operations become no-ops (objects are discarded)
  - When pool is empty, Get operations create new objects via factory function
*/
package pool