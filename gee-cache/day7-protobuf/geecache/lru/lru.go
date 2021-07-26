package lru

import "container/list"

// Cache is a LRU cache. It is not safe for concurrent access.
type Cache struct {
	maxBytes int64                    // 允许使用的最大内存
	nbytes   int64                    // 当前已经使用的内存
	ll       *list.List               // 双向链表
	cache    map[string]*list.Element // 值是双向链表的节点指针，Element的Value是一个interface{}
	// optional and executed when an entry is purged.
	OnEvicted func(key string, value Value) // 某条记录被移除时的回调函数，可以为 nil。
}

// 双向链表的数据类型
type entry struct {
	key   string
	value Value
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int // 字节数量
}

// 实例化Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 查找
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// MoveToFront将ele放到ll的root的后面，root是个哑结点
		c.ll.MoveToFront(ele)
		// ele.Value原本是个空指针，代表任意值，转成*entry的类型
		// 这里强转可能导致错误，但是cache里有值的话，这个值一定存在的
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// 缓存淘汰
func (c *Cache) RemoveOldest() {
	// 链表长度为0才返回空
	// 否则返回root的前一个节点
	// 然后将其删除
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		// delete针对map的key进行操作
		delete(c.cache, kv.key)
		// 更新内存，c.nbytes减去删掉的key和value的字节长度
		c.nbytes -= (int64(len(kv.key)) + int64(kv.value.Len()))
		// 回调函数不为空，执行回调
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 新增，移到最前面
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		// 如果key在缓存中，更新这个value
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
	}else{
		// key不存在，加入缓存
		ele := c.ll.PushFront(&entry{key,value})
		c.cache[key] = ele
		c.nbytes += int64(len(key))+int64(value.Len())
	}
	//
	for c.maxBytes != 0 && c.maxBytes<c.nbytes{
		c.RemoveOldest()
	}
}

func (c *Cache)Len() int{
	return c.ll.Len()
}
