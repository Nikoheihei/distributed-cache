package geecache

import (
	"GopherStore/geecache/consistenthash"
	"GopherStore/geecache/geecachepb"
	"GopherStore/geerpc"
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type CacheService struct {
}

// 包装了这段内部调用流程
// GetGroup(groupName) -> group.Get(key) -> ByteView -> []byte
func (s *CacheService) Get(args *geecachepb.Request, reply *geecachepb.Response) error {
	//1.找到对应缓存分组
	group := GetGroup(args.Group)
	if group == nil {
		return fmt.Errorf("no such group:%s", args.Group)
	}
	//2.调用本地缓存的Get方法
	view, err := group.Get(args.Key)
	if err != nil {
		return err
	}
	//3.把结果塞进reply返回
	reply.Value = view.ByteSlice()
	return nil
}

// 实际上是PeerGetter
type rpc struct {
	addr string
}

func (f *rpc) Get(in *geecachepb.Request, out *geecachepb.Response) error {
	//1.构造请求
	log.Printf("[GeeCache RPC] call peer=%s group=%s key=%s", f.addr, in.GetGroup(), in.GetKey())
	client, err := geerpc.Dial("tcp", f.addr)
	if err != nil {
		return err
	}
	defer client.Close()
	//2.发起远程调用
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return client.Call(ctx, "CacheService.Get", in, out)

}

// 实际上是PeerPicker
type RPCPool struct {
	self        string
	mu          sync.Mutex
	peers       *consistenthash.Map
	rpcFetchers map[string]*rpc
}

func (p *RPCPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	//问一致性哈希，这个key要去哪

	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		log.Printf("[GeeCache RPC] pick peer=%s for key=%s self=%s", peer, key, p.self)
		return p.rpcFetchers[peer], true
	}
	return nil, false
}

func (p *RPCPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.rpcFetchers = make(map[string]*rpc, len(peers))
	for _, peer := range peers {
		p.rpcFetchers[peer] = &rpc{addr: peer}
	}
}
func NewRPCPool(addr string) *RPCPool {
	return &RPCPool{
		self: addr,
	}
}
