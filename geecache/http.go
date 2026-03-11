package geecache

import (
	"GopherStore/geecache/consistenthash"
	pb "GopherStore/geecache/geecachepb"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"

)

// HTTPPool结构体作为承载节点间http通信的核心数据结构
// 这里只实现服务端
const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

// 实现了PeerPicker接口的HTTPPool结构体，负责选择节点和创建HTTP客户端
type HTTPPool struct {
	self     string //自己的地址，包括主机名/IP和端口号
	basePath string //节点间通讯地址的前缀

	mu          sync.Mutex //保护下面的变量
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter //远程节点与对应httpGetter的映射表
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s]%s", p.self, fmt.Sprintf(format, v...))
}

// 这一步就相当于从本地拿数据
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path:" + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	//期望格式：/<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)

	if len(parts) != 2 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]
	group := GetGroup(groupName)

	if group == nil {
		http.Error(w, "No such group: "+groupName, http.StatusNotFound)
		return
	}
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := &pb.Response{
		Value: view.ByteSlice(),
	}
	body, err := proto.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream") //表示类型是不关心具体格式的二进制数据
	w.Write(body)
}

func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	//实例化了一致性哈希算法，并添加了传入的节点，并为每一个节点创建了一个
	//HTTP客户端httpGetter
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))

	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer方法根据key选择相应节点，并返回该节点的HTTP客户端（httpGetter)
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	//包装了一致性哈希算法的 Get() 方法，根据具体的 key，选择节点，返回节点对应的 HTTP 客户端。
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		//排除了没找的和找到自己的情况
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

// 创建具体的HTTP客户端类httpGetter，实现PeerGetter接口
type httpGetter struct {
	baseURL string //要访问的远程节点的地址
}

// 向远程节点请求缓存数据
func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	//拼接完整URL
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)

	res, err := http.Get(u) //获取返回值，并转化为[]byte类型
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d", res.StatusCode)
	}

	bytes, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}
	return nil
}

var _ PeerGetter = (*httpGetter)(nil)
