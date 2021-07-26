package geecache

/* 缓存值的抽象与封装 */

// 抽象一个只读数据结构 ByteView 用来表示缓存值，是 GeeCache 主要的数据结构之一
// b存储真实的缓存值
type ByteView struct{
	b []byte
}

// 返回ByteView实例的长度
// 被缓存对象必须实现Value接口
func (v ByteView)Len()int{
	return len(v.b)
}

// 返回一个ByteView的切片副本
func (v ByteView)ByteSlice()[]byte{
	return cloneBytes(v.b)
}

// 返回字节数组转成的字符串
func (v ByteView)String()string{
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte,len(b))
	copy(c,b)
	return c
}