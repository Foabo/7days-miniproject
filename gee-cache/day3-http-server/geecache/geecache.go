package geecache

import (
	"fmt"
	"log"
	"sync"
)

/* 负责与外部交互，控制缓存存储和获取的主流程 */

// 定义接口 Getter 和 回调函数 Get
type Getter interface {
	Get(key string) ([]byte, error)
}

// 定义函数类型 GetterFunc，并实现 Getter 接口的 Get 方法。
type GetterFunc func(key string) ([]byte, error)

// 接口型函数
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// 一个 Group 可以认为是一个缓存的命名空间，
// 每个 Group 拥有一个唯一的名称 name
// getter 缓存未命中时候的回调函数
// cache是并发缓存
type Group struct {
	name      string
	getter    Getter
	mainCache cache
}

// 全局变量
var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:   name,
		getter: getter,
		mainCache: cache{
			cacheBytes: cacheBytes,
		},
	}
	groups[name] = g
	return g
}

// 根据所给名称返回Group
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// 最核心的方法Get
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("OHHHHHHHHH,[GeeCache] hit")
		return v, nil
	}
	// 值没有被缓存,通过回调函数去找这个值
	return g.load(key)
}

func (g *Group) load(key string) (ByteView, error) {
	return g.getLocally(key)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	// 因为是单机并发，Getter接口获取到数据源
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil

}

// 将key放到lru链表最前面
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}


