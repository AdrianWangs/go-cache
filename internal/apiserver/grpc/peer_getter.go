package grpc

import (
	"github.com/AdrianWangs/go-cache/internal/peers"
	pb "github.com/AdrianWangs/go-cache/proto/cache_server"
)

// PeerGetter 是使用gRPC实现的PeerGetter接口
type PeerGetter struct {
	client *CacheClient
}

// NewPeerGetter 创建一个新的gRPC PeerGetter
func NewPeerGetter(addr string) *PeerGetter {
	return &PeerGetter{
		client: NewCacheClient(addr),
	}
}

// Get 通过gRPC获取缓存数据
func (p *PeerGetter) Get(group string, key string) ([]byte, error) {
	return p.client.Get(group, key)
}

// GetByProto 实现通过protobuf的获取方法
func (p *PeerGetter) GetByProto(req *pb.Request, resp *pb.Response) error {
	// 使用gRPC客户端获取数据
	value, err := p.client.Get(req.Group, req.Key)
	if err != nil {
		return err
	}

	// 设置响应
	resp.Value = value
	return nil
}

// Close 关闭gRPC连接
func (p *PeerGetter) Close() error {
	return p.client.Close()
}

// 确保PeerGetter实现了peers.PeerGetter接口
var _ peers.PeerGetter = (*PeerGetter)(nil)
