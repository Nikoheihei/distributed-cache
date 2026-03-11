package lru

import (
	"container/list"
)

type Cache struct {
	maxBytes  int64
	nBytes    int64
	ll        *list.List
	cache     map[string]*list.Element
	OnEvicted func(key string, value Value)
}

// list库里的Value字段是接口，需要自己定义。
// entry是双向链表节点的数据类型
// entry包含key是因为在删除map中的记录时需要key。
type entry struct {
	key   string
	value Value //Value是一个接口，为了通用性，允许不同的类型。
}

type Value interface {
	Len() int
}

// 初始化一个Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted, //类似于钩子函数
	}
}

// 实现查找功能：1.从字典查到对应节点。2.该节点移动到队首
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok { //这里ele的类型是*list.Element
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry) //ele.Value的类型是interface{}，获取value需要断言。不是entry因为是ele的类型是指针类型。
		return kv.value, true    //而且Go 的思维是：我拿到的是一个节点指针，节点里装的是 interface{}，所以我几乎总是存一个结构体指针
	}
	return
}

// 实现删除功能
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		kv := ele.Value.(*entry)
		c.ll.Remove(ele)
		delete(c.cache, kv.key)
		c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 实现添加功能
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nBytes += int64(len(key) + value.Len())
	}
	//其实这里体现了编程一个重要思维，那就是：先把事情做完，再考虑清理工作。
	//也就是说，先把新节点加进去，再考虑要不要淘汰旧节点。因为不管咋样都要添加节点。
	for c.maxBytes != 0 && c.maxBytes < c.nBytes { //如果maxBytes为0表示不限制缓存大小
		c.RemoveOldest()
	}
}

// 一个方便测试加的函数
func (c *Cache) Len() int {
	return c.ll.Len()
}
