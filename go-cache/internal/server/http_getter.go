package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/AdrianWangs/go-cache/internal/peers"
	"github.com/AdrianWangs/go-cache/pkg/logger"
	pb "github.com/AdrianWangs/go-cache/proto/cache_server"
	"google.golang.org/protobuf/proto"
)

// httpGetter 用于从HTTP服务器获取数据
type httpGetter struct {
	baseURL string
}

// Get 从HTTP服务器获取数据
//
// 传入参数:
//   - group: 组名
//   - key: 键名
//
// 返回值:
//   - 数据: []byte
//   - 错误: error
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	logger.Debugf("[httpGetter] Get %s/%s from %s", group, key, h.baseURL)

	// 最终要访问的HTTP服务器的完整URL
	url := fmt.Sprintf("%v%v/%v", h.baseURL, group, key)

	resp, err := http.Get(url)
	if err != nil {
		logger.Errorf("HTTP Get error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// 如果HTTP状态码不是200，则返回错误
	if resp.StatusCode != http.StatusOK {
		logger.Warnf("Server returned non-OK status: %v", resp.Status)
		return nil, fmt.Errorf("server returned: %v", resp.Status)
	}

	// 读取响应体
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("Reading response body error: %v", err)
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	logger.Debugf("Successfully got %d bytes from %s", len(bytes), url)
	return bytes, nil
}

// GetByProto 从HTTP服务器获取数据,使用protobuf协议
//
// 传入参数:
//   - in: 请求
//   - out: 响应
//
// 返回值:
//   - 错误: error
func (h *httpGetter) GetByProto(in *pb.Request, out *pb.Response) error {
	url := fmt.Sprintf("%v", h.baseURL)

	req := &pb.Request{
		Group: in.Group,
		Key:   in.Key,
	}

	requestBytes, err := proto.Marshal(req)
	if err != nil {
		logger.Errorf("Failed to marshal request: %v", err)
		return err
	}

	resp, err := http.Post(url, "application/protobuf", bytes.NewBuffer(requestBytes))
	if err != nil {
		logger.Errorf("HTTP Get error: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warnf("Server returned non-OK status: %v", resp.Status)
		return fmt.Errorf("server returned: %v", resp.Status)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("Reading response body error: %v", err)
		return fmt.Errorf("reading response body: %v", err)
	}

	if err := proto.Unmarshal(bytes, out); err != nil {
		logger.Errorf("Failed to unmarshal response: %v", err)
		return fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return nil
}

// 确保httpGetter实现了peers.PeerGetter接口
var _ peers.PeerGetter = (*httpGetter)(nil)
