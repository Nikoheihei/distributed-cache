package geecache

import (
	"GopherStore/geecache/lru"
	"math/rand"
	"sync"
	"time"
)

// cache结构体对lru进行了封装，并添加了互斥锁以保证并发安全
type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64 //允许的最大缓存值
	ttl        time.Duration
	jitter     time.Duration // TTL抖动范围，防止缓存雪崩
	stopEvict  chan struct{} // 主动过期扫描停止信号
}

func (c *cache) add(key string, value ByteView) {
	c.addWithTTL(key, value, 0)
}

// addWithTTL 添加缓存项，支持自定义TTL（用于空值缓存短TTL，防缓存穿透）
func (c *cache) addWithTTL(key string, value ByteView, overrideTTL time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//延迟初始化，该对象的创建会延迟到第一次使用它的时候。用于提高性能，减少程序内存要求
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}

	ttl := c.ttl
	if overrideTTL > 0 {
		ttl = overrideTTL
	}

	c.lru.Add(key, cachedValue{
		ByteView:  value,
		expireAt:  c.expireAtWithJitter(ttl),
		expirable: ttl > 0,
	})
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	v, ok := c.lru.Get(key)
	if ok {
		entry := v.(cachedValue)
		if entry.expired(time.Now()) {
			c.lru.Remove(key)
			return ByteView{}, false
		}
		return entry.ByteView, true
	}
	return
}

func (c *cache) setTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ttl = ttl
	// 默认 jitter 为 TTL 的 ±10%，防止大量key同时过期导致缓存雪崩
	if ttl > 0 {
		c.jitter = ttl / 10
	}
}

// startEviction 启动主动过期扫描，弥补惰性删除的不足
// 每 scanInterval 扫描一次，清除已过期的缓存项
func (c *cache) startEviction(scanInterval time.Duration) {
	if scanInterval <= 0 || c.ttl <= 0 {
		return
	}
	c.stopEvict = make(chan struct{})
	go func() {
		ticker := time.NewTicker(scanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.evictExpired()
			case <-c.stopEvict:
				return
			}
		}
	}()
}

// stopEviction 停止主动过期扫描
func (c *cache) stopEvictionLoop() {
	if c.stopEvict != nil {
		close(c.stopEvict)
		c.stopEvict = nil
	}
}

// evictExpired 主动清除已过期的缓存项
func (c *cache) evictExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	now := time.Now()
	// LRU 的链表尾部是最久未访问的，从尾部开始检查过期
	// 最多扫描 100 个，避免长时间持锁
	scanned := 0
	for {
		ele := c.lru.Back()
		if ele == nil {
			break
		}
		kv := ele.Value.(*lru.Entry)
		cv, ok := kv.Value.(cachedValue)
		if !ok {
			break
		}
		if !cv.expired(now) {
			break
		}
		c.lru.Remove(kv.Key)
		IncEvictions()
		scanned++
		if scanned >= 100 {
			break
		}
	}
}

// expireAtWithJitter 在TTL基础上加随机抖动，防止大量key同时过期（缓存雪崩）
func (c *cache) expireAtWithJitter(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	jitter := time.Duration(0)
	// jitter不超过TTL的10%，且不超过TTL本身（防止短TTL被jitter覆盖）
	if c.jitter > 0 && c.jitter < ttl {
		// 随机偏移 [-jitter, +jitter]
		jitter = time.Duration(rand.Int63n(int64(c.jitter)*2) - int64(c.jitter))
	}
	return time.Now().Add(ttl + jitter)
}

type cachedValue struct {
	ByteView
	expireAt  time.Time
	expirable bool
}

func (v cachedValue) expired(now time.Time) bool {
	return v.expirable && !v.expireAt.IsZero() && !now.Before(v.expireAt)
}
