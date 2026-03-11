package geecache

import (
	"GopherStore/geecache/lru"
	"sync"
)

// cache结构体对lru进行了封装，并添加了互斥锁以保证并发安全
type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64 //允许的最大缓存值
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//延迟初始化，该对象的创建会延迟到第一次使用它的时候。用于提高性能，减少程序内存要求
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	v, ok := c.lru.Get(key)
	if ok {
		return v.(ByteView), ok //add方法中存的是ByteView类型，所以这里可以断言为ByteView类型
	}
	return
}
