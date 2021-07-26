package geecache

import (
	"fmt"
	"log"
	"testing"
)

/*测试单机并发缓存*/

// 用一个 map 模拟耗时的数据库。
// 想象成mysql吧，（虽然用map模拟）
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// 测试Get方法
// 需要测试缓存命中，未命中，key为空三种？
func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	gee := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}

			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		// 缓存为空的情况，调用回调函数从数据源获取数据
		// 即从实现了Getter接口的数据源获取数据
		// err != nil  获取不到数据
		// view.String() != v说明缓存和数据库不一致
		if view, err := gee.Get(k); err != nil || view.String() != v {
			t.Fatalf("failed to get value of %s", k)
		} // load from callback function

		// 缓存已经存在的情况
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}
}
