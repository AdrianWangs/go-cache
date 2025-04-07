package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/AdrianWangs/go-cache/pkg/logger"
	pb "github.com/AdrianWangs/go-cache/proto/cache_server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCGetter 实现从gRPC缓存节点获取数据的NodeGetter接口
type GRPCGetter struct {
	addr    string              // 服务器地址 (格式: host:port)
	timeout time.Duration       // 请求超时
	conn    *grpc.ClientConn    // gRPC连接
	client  pb.GroupCacheClient // gRPC客户端
}

// NewGRPCGetter 创建一个新的gRPC缓存数据获取器
func NewGRPCGetter(addr string) *GRPCGetter {
	return &GRPCGetter{
		addr:    addr,
		timeout: 3 * time.Second, // 默认超时时间
	}
}

// ensureConnection 确保gRPC连接已建立
func (g *GRPCGetter) ensureConnection() error {
	if g.client != nil {
		return nil // 已经有连接
	}

	// 创建新连接
	conn, err := grpc.Dial(g.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(2*time.Second),
	)
	if err != nil {
		return fmt.Errorf("无法连接到gRPC服务器 %s: %v", g.addr, err)
	}

	g.conn = conn
	g.client = pb.NewGroupCacheClient(conn)
	logger.Debugf("已连接到gRPC服务器: %s", g.addr)
	return nil
}

// Close 关闭gRPC连接
func (g *GRPCGetter) Close() error {
	if g.conn != nil {
		err := g.conn.Close()
		g.conn = nil
		g.client = nil
		return err
	}
	return nil
}

// Get 从gRPC缓存节点获取数据
func (g *GRPCGetter) Get(group string, key string) ([]byte, error) {
	// 确保连接已建立
	if err := g.ensureConnection(); err != nil {
		return nil, err
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	// 发送gRPC请求
	req := &pb.Request{
		Group: group,
		Key:   key,
	}

	resp, err := g.client.Get(ctx, req)
	if err != nil {
		// 如果是连接问题，尝试重连
		logger.Warnf("gRPC调用失败: %v，将尝试重连", err)
		g.Close() // 关闭旧连接

		if reconnErr := g.ensureConnection(); reconnErr != nil {
			logger.Errorf("重连失败: %v", reconnErr)
			return nil, err // 返回原始错误
		}

		// 重试一次
		resp, err = g.client.Get(ctx, req)
		if err != nil {
			return nil, err
		}
	}

	return resp.Value, nil
}

// GetByProto 通过protobuf从gRPC缓存节点获取数据
func (g *GRPCGetter) GetByProto(req *pb.Request, resp *pb.Response) error {
	// 确保连接已建立
	if err := g.ensureConnection(); err != nil {
		return err
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	// 发送gRPC请求
	result, err := g.client.Get(ctx, req)
	if err != nil {
		// 如果是连接问题，尝试重连
		logger.Warnf("gRPC调用失败: %v，将尝试重连", err)
		g.Close() // 关闭旧连接

		if reconnErr := g.ensureConnection(); reconnErr != nil {
			logger.Errorf("重连失败: %v", reconnErr)
			return err // 返回原始错误
		}

		// 重试一次
		result, err = g.client.Get(ctx, req)
		if err != nil {
			return err
		}
	}

	// 复制结果到响应
	resp.Value = result.Value
	return nil
}

// SetTimeout 设置请求超时时间
func (g *GRPCGetter) SetTimeout(timeout time.Duration) {
	g.timeout = timeout
}

// Delete 从gRPC缓存节点删除指定的缓存项
func (g *GRPCGetter) Delete(group string, key string) error {
	// 确保连接已建立
	if err := g.ensureConnection(); err != nil {
		return err
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	// 创建请求
	req := &pb.DeleteRequest{
		Group: group,
		Key:   key,
	}

	// 发送gRPC请求
	_, err := g.client.Delete(ctx, req)
	if err != nil {
		// 如果是连接问题，尝试重连
		logger.Warnf("gRPC Delete调用失败: %v，将尝试重连", err)
		g.Close() // 关闭旧连接

		if reconnErr := g.ensureConnection(); reconnErr != nil {
			logger.Errorf("重连失败: %v", reconnErr)
			return err // 返回原始错误
		}

		// 重试一次
		_, err = g.client.Delete(ctx, req)
		if err != nil {
			return err
		}
	}

	return nil
}
