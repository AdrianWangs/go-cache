package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/AdrianWangs/go-cache/internal/cache"
	"github.com/AdrianWangs/go-cache/pkg/logger"
	pb "github.com/AdrianWangs/go-cache/proto/cache_server"
	"google.golang.org/grpc"
)

// CacheServer 实现缓存节点的gRPC服务
type CacheServer struct {
	pb.UnimplementedGroupCacheServer
	server *grpc.Server
	addr   string
}

// NewCacheServer 创建一个新的gRPC缓存服务器
func NewCacheServer(addr string) *CacheServer {
	return &CacheServer{
		addr: addr,
	}
}

// Start 启动gRPC服务器
func (s *CacheServer) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("无法监听地址 %s: %v", s.addr, err)
	}

	s.server = grpc.NewServer()
	pb.RegisterGroupCacheServer(s.server, s)

	logger.Infof("gRPC缓存服务器正在监听：%s", s.addr)
	go func() {
		if err := s.server.Serve(lis); err != nil {
			logger.Errorf("gRPC服务器运行错误: %v", err)
		}
	}()

	return nil
}

// Stop 停止gRPC服务器
func (s *CacheServer) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
		logger.Info("gRPC缓存服务器已停止")
	}
}

// Get 实现gRPC的Get方法，从缓存中获取值
func (s *CacheServer) Get(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	group := cache.GetGroup(req.Group)
	if group == nil {
		return nil, fmt.Errorf("未找到组: %s", req.Group)
	}

	// 从缓存获取值
	val, err := group.Get(req.Key)
	if err != nil {
		return nil, err
	}

	return &pb.Response{
		Value: val.ByteSlice(),
	}, nil
}

// Delete 实现gRPC的Delete方法，从缓存中删除值
func (s *CacheServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	group := cache.GetGroup(req.Group)
	if group == nil {
		return nil, fmt.Errorf("未找到组: %s", req.Group)
	}

	// 从缓存删除值
	err := group.Delete(req.Key)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteResponse{
		Success: true,
	}, nil
}
