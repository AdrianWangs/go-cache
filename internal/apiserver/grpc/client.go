package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/AdrianWangs/go-cache/pkg/logger"
	pb "github.com/AdrianWangs/go-cache/proto/cache_server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CacheClient gRPC缓存客户端
type CacheClient struct {
	addr    string
	conn    *grpc.ClientConn
	client  pb.GroupCacheClient
	timeout time.Duration
}

// NewCacheClient 创建一个新的gRPC缓存客户端
func NewCacheClient(addr string) *CacheClient {
	return &CacheClient{
		addr:    addr,
		timeout: 3 * time.Second, // 默认超时时间
	}
}

// Connect 连接到gRPC服务器
func (c *CacheClient) Connect() error {
	// 如果已经有连接了，先关闭
	if c.conn != nil {
		c.conn.Close()
	}

	// 创建新连接
	conn, err := grpc.Dial(c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(2*time.Second),
	)
	if err != nil {
		return fmt.Errorf("无法连接到gRPC服务器 %s: %v", c.addr, err)
	}

	c.conn = conn
	c.client = pb.NewGroupCacheClient(conn)
	logger.Debugf("已连接到gRPC服务器: %s", c.addr)
	return nil
}

// Close 关闭连接
func (c *CacheClient) Close() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.client = nil
		return err
	}
	return nil
}

// Get 通过gRPC获取缓存值
func (c *CacheClient) Get(group, key string) ([]byte, error) {
	if c.client == nil {
		if err := c.Connect(); err != nil {
			return nil, err
		}
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// 发送gRPC请求
	req := &pb.Request{
		Group: group,
		Key:   key,
	}

	resp, err := c.client.Get(ctx, req)
	if err != nil {
		// 如果是连接问题，可以尝试重连
		logger.Warnf("gRPC调用失败: %v，将尝试重连", err)
		if reconnErr := c.Connect(); reconnErr != nil {
			logger.Errorf("重连失败: %v", reconnErr)
			return nil, err // 返回原始错误
		}

		// 重试一次
		resp, err = c.client.Get(ctx, req)
		if err != nil {
			return nil, err
		}
	}

	return resp.Value, nil
}

// SetTimeout 设置客户端请求超时
func (c *CacheClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}
