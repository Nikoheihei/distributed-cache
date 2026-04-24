package geecache

import (
	"testing"
	"time"
)

func TestCacheTTLExpiresEntries(t *testing.T) {
	c := cache{cacheBytes: 1 << 10}
	c.setTTL(20 * time.Millisecond)
	c.add("Tom", ByteView{b: []byte("630")})

	if _, ok := c.get("Tom"); !ok {
		t.Fatalf("expected cache hit before TTL expiry")
	}

	time.Sleep(30 * time.Millisecond)

	if _, ok := c.get("Tom"); ok {
		t.Fatalf("expected cache miss after TTL expiry")
	}
}

func TestCacheWithoutTTLDoesNotExpire(t *testing.T) {
	c := cache{cacheBytes: 1 << 10}
	c.add("Tom", ByteView{b: []byte("630")})

	time.Sleep(20 * time.Millisecond)

	if value, ok := c.get("Tom"); !ok || value.String() != "630" {
		t.Fatalf("expected cache hit without TTL, got ok=%v value=%q", ok, value.String())
	}
}

// TestCacheTTLJitter 测试TTL抖动：多次add同一个key，过期时间不完全相同
func TestCacheTTLJitter(t *testing.T) {
	c := cache{cacheBytes: 1 << 20}
	c.setTTL(1 * time.Second)

	// 添加多个key，它们的过期时间应该不完全相同（有jitter）
	for i := 0; i < 100; i++ {
		key := string(rune('A' + i))
		c.add(key, ByteView{b: []byte("val")})
	}

	// 检查：获取每个key，应该都能命中
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		t.Fatal("lru should be initialized")
	}

	// 收集所有过期时间
	distinctExpireAts := 0
	prevExpire := time.Time{}
	for i := 0; i < 100; i++ {
		key := string(rune('A' + i))
		if v, ok := c.lru.Get(key); ok {
			entry := v.(cachedValue)
			if entry.expireAt != prevExpire {
				distinctExpireAts++
				prevExpire = entry.expireAt
			}
		}
	}

	// 有jitter的情况下，100个key的过期时间不应该都相同
	// 至少应该有多个不同的过期时间
	if distinctExpireAts < 10 {
		t.Fatalf("expected at least 10 distinct expire times due to jitter, got %d", distinctExpireAts)
	}
}

// TestCacheAddWithOverrideTTL 测试自定义TTL（空值缓存短TTL）
func TestCacheAddWithOverrideTTL(t *testing.T) {
	c := cache{cacheBytes: 1 << 10}
	c.setTTL(10 * time.Second) // 默认长TTL

	// 用短TTL添加空值
	c.addWithTTL("missing_key", ByteView{b: []byte{}}, 50*time.Millisecond)

	if _, ok := c.get("missing_key"); !ok {
		t.Fatal("expected cache hit for empty value with short TTL")
	}

	// 等待短TTL过期
	time.Sleep(100 * time.Millisecond)

	if _, ok := c.get("missing_key"); ok {
		t.Fatal("expected cache miss after short TTL expiry")
	}
}

// TestCacheEvictExpired 测试主动过期扫描
func TestCacheEvictExpired(t *testing.T) {
	c := cache{cacheBytes: 1 << 20}
	c.setTTL(20 * time.Millisecond)

	c.add("key1", ByteView{b: []byte("val1")})
	c.add("key2", ByteView{b: []byte("val2")})

	// 确认存在
	if _, ok := c.get("key1"); !ok {
		t.Fatal("key1 should exist")
	}

	// 等待过期
	time.Sleep(30 * time.Millisecond)

	// 手动触发主动过期扫描
	c.evictExpired()

	// 确认已被清除
	c.mu.Lock()
	len1 := c.lru.Len()
	c.mu.Unlock()
	if len1 != 0 {
		t.Fatalf("expected 0 entries after eviction, got %d", len1)
	}
}

// TestCacheEvictionLoop 测试后台过期扫描goroutine
func TestCacheEvictionLoop(t *testing.T) {
	c := cache{cacheBytes: 1 << 20}
	c.setTTL(30 * time.Millisecond)
	c.startEviction(10 * time.Millisecond)
	defer c.stopEvictionLoop()

	c.add("temp", ByteView{b: []byte("will expire")})

	// 短时间内应该还在
	if _, ok := c.get("temp"); !ok {
		t.Fatal("temp should exist before expiry")
	}

	// 等待过期+扫描
	time.Sleep(60 * time.Millisecond)

	// 主动扫描应该已清除
	c.mu.Lock()
	l := c.lru.Len()
	c.mu.Unlock()
	if l != 0 {
		t.Fatalf("expected 0 after eviction scan, got %d", l)
	}
}
