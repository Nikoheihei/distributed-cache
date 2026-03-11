package geecache

// 用一个只读数据结构byteview来表示缓存值
// byteview提供具体实现和安全性保证。
// Value和ByteView的关系就像抽象层和实现层的关系【之前zinx就是先定义interface，用interface实现核心逻辑/算法层，实现层实现业务逻辑关系】
type ByteView struct {
	b []byte //选择byte是为了支持任何数据类型的存储，如字符串，图片
}

// 被缓存对象必须实现Value接口
func (v ByteView) Len() int {
	return len(v.b)
}

func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b) //防止切片被外部程序修改。如果直接返回b，切片传递的是指向底层数据的指针
	return c
}
