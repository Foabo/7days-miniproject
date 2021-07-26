package main

import "fmt"

func main() {
	a := get(GetterFunc(test), "a")
	b := get(GetterFunc(func(k string) int {
		if v, ok := mp[k]; ok {
			return v
		}
		return 0
	}), "b")
	fmt.Println(a)
	fmt.Println(b)
}

var mp = map[string]int{
	"a": 1,
	"b": 2,
}

type Getter interface {
	Get(k string) int
}
type GetterFunc func(k string) int

func (f GetterFunc) Get(k string) int {
	return f(k)
}
func test(k string) int {
	if v, ok := mp[k]; ok {
		return v
	}
	return 0
}
func get(getter Getter, k string) int {
	return getter.Get(k)
}
