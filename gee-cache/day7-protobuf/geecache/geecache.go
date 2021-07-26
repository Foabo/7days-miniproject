package geecache

import (
	"fmt"
	pb "geecache/geecachepb"
	"geecache/singleflight"
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
	peers     PeerPicker
	// use singleflight.Group to make sure that
	// each key is only fetched once
	loader *singleflight.SGFGroup
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
		loader: &singleflight.SGFGroup{},
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
// Group也实现了Getter接口
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

// 缓存没有就来load()函数找
func (g *Group) load(key string) (value ByteView, err error) {
	// each key is only fetched once (either locally or remotely)
	// regardless of the number of concurrent callers.
	// Do返回key对应的value
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			// peer返回的是http客户端，能够请求远程节点
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err := g.getFromPeer(peer, key); err == nil {
					return value, nil
				} else {
					log.Println("[GeeCache] Failed to get from peer", err)

				}
			}
		}
		return g.getLocally(key)
	})

	if err == nil {
		// 将viewi强转成ByteView
		return viewi.(ByteView), nil
	}
	return

}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	// 请求远程节点获取数据到resp中
	resp := &pb.Response{}
	err := peer.Get(req, resp)

	if err != nil {
		return ByteView{}, err
	}
	return ByteView{resp.Value}, nil
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

func (g *Group) RegisterPeer(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}
