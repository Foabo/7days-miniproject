package singleflight

import "sync"

// call 代表正在进行中，或已经结束的请求。使用 sync.WaitGroup 锁避免重入。
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// singleflight的主数据结构，管理不同的key的请求call
type SGFGroup struct {
	mu sync.Mutex
	m  map[string]*call
}

// Do 方法，接收 2 个参数，第一个参数是 key，第二个参数是一个函数 fn。
// Do 的作用就是，针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。
// 针对的是一瞬间的请求压力
func (g *SGFGroup) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	// 延迟加载
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 有调用的call
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	// 当前是第一次调用，则初试化并加载进SGFGroup
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	// 为了防止键过期和清理内存，将key删除
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err

}
