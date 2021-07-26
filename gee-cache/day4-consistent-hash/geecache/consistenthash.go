package geecache

import (
	"hash/crc32"
	"sort"
	"strconv"
)

/* 实现一致性哈希 */

type Hash func(data []byte) uint32

type Map struct {
	hash     Hash           // hash方法
	replicas int            // 有多少个副本，副本就是虚拟节点
	keys     []int          // 哈希环，保存的是虚拟节点的的哈希值
	hashMap  map[int]string // 虚拟节点与真实节点的映射，保存的是真实节点
}

func New(repplicas int, fn Hash) *Map {
	m := &Map{
		replicas: repplicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加真实节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		//  对每一个真实节点 key，对应创建 m.replicas 个虚拟节点
		for i := 0; i < m.replicas; i++ {
			// 计算key的hash值
			// 通过添加编号的方式区分不同虚拟节点。
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 将虚拟节点添加到环
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// 根据key，计算哈希值，然后顺时针找到第一个匹配的虚拟节点的下标idx
// 从m.keys中获取到对应的哈希值
// 然后得到真实的节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	// 二分查找，找到第一个大于等于key的hash的虚拟节点的下标
	// idx最极端情况下返回len(m.keys)
	// 所以要取模
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	// 通过虚拟节点的哈希值找到真实的节点。
	return m.hashMap[m.keys[idx%len(m.keys)]]
}

