package geecache

import (
	pb "GopherStore/geecache/geecachepb"
	"GopherStore/geecache/singleflight"
	"fmt"
	"log"
	"sync"
	"time"
)

// 当缓存未命中，且不从远程节点获取时，采用回调函数。这部分交给用户来实现，否则数据源种类太多+扩展性不好
type Getter interface {
	Get(key string) ([]byte, error)
}

// 函数类型，实现了Getter接口的Get方法
// 函数类型实现一个接口被称为接口型函数。方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数。
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// 空值缓存的短TTL，防止缓存穿透
const emptyCacheTTL = 30 * time.Second

// 定义group结构体，一个group表示一个缓存命名空间
type Group struct {
	name      string
	getter    Getter //缓存未命中时获取源数据的回调
	mainCache cache
	peers     PeerPicker          //这里就是HTTPPool
	loader    *singleflight.Group //确保每个key只被fetch一次
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("geecache.NewGroup: getter is nil")
	}
	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes}, //注意这里没有初始化lru
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	IncRequests()
	if v, ok := g.mainCache.get(key); ok {
		IncHits()
		return v, nil
	}
	IncMisses()
	return g.load(key)
}
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

func (g *Group) SetCacheTTL(ttl time.Duration) {
	g.mainCache.setTTL(ttl)
	// 设置TTL后启动主动过期扫描，扫描间隔为TTL的1/4
	if ttl > 0 {
		g.mainCache.startEviction(ttl / 4)
	}
}

func (g *Group) load(key string) (value ByteView, err error) {
	start := time.Now()
	defer func() {
		ObserveLoad(time.Since(start))
	}()
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getlocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getlocally(key string) (ByteView, error) {
	IncSourceRequests()
	bytes, err := g.getter.Get(key)
	if err != nil {
		// 缓存空值防穿透：DB也查不到时，缓存一个空值，设短TTL
		// 这样同一个key短时间内不会反复穿透到DB
		IncPenetrations()
		g.populateCacheWithTTL(key, ByteView{b: []byte{}}, emptyCacheTTL)
		return ByteView{}, err
	}

	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	IncPeerRequests()
	err := peer.Get(req, res)
	if err != nil {
		IncPeerErrors()
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// populateCacheWithTTL 使用自定义TTL写入缓存（用于空值缓存短TTL）
func (g *Group) populateCacheWithTTL(key string, value ByteView, ttl time.Duration) {
	g.mainCache.addWithTTL(key, value, ttl)
}
