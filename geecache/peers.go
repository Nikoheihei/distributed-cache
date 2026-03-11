package geecache

import pb "GopherStore/geecache/geecachepb"

// PeerPicker接口的PickPeer方法根据传入的key选择相应节点PeerGetter
type PeerPicker interface { //HTTPPool实现
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter接口的Get方法用于从远程节点获取缓存值
type PeerGetter interface { //httpGetter实现
	Get(in *pb.Request, out *pb.Response) error
}
