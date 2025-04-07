// Package handlers 实现API处理器
package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/AdrianWangs/go-cache/internal/cache"
	"github.com/AdrianWangs/go-cache/pkg/logger"
	pb "github.com/AdrianWangs/go-cache/proto/cache_server"
	"google.golang.org/protobuf/proto"
)

// HTTPClient 封装了HTTP请求功能
type HTTPClient interface {
	Get(url string) (*http.Response, error)
	Post(url, contentType string, body io.Reader) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
}

// 默认使用标准HTTP客户端
var defaultHTTPClient HTTPClient = &http.Client{}

// HTTPGetter 使用HTTP协议实现的NodeGetter
type HTTPGetter struct {
	baseURL    string     // 基础URL
	httpClient HTTPClient // HTTP客户端
}

// NewHTTPGetter 创建新的HTTP客户端
func NewHTTPGetter(baseURL string) *HTTPGetter {
	return &HTTPGetter{
		baseURL:    baseURL,
		httpClient: defaultHTTPClient,
	}
}

// Get 通过HTTP获取缓存值
func (h *HTTPGetter) Get(group, key string) ([]byte, error) {
	// 构建请求URL
	u := fmt.Sprintf("%v/%v/%v", h.baseURL, url.QueryEscape(group), url.QueryEscape(key))

	logger.Debugf("发送HTTP GET请求: %s", u)

	// 发送HTTP请求
	res, err := h.httpClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// 检查响应状态
	if res.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("key not found: %s", key)
	} else if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回错误: %v", res.Status)
	}

	// 读取响应内容
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	return bytes, nil
}

// GetByProto 通过Protobuf获取缓存值
func (h *HTTPGetter) GetByProto(req *pb.Request, resp *pb.Response) error {
	// 序列化请求
	body, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}

	// 构建完整的URL (baseURL包含basePath)
	logger.Debugf("发送Protobuf POST请求: %s (group=%s, key=%s)",
		h.baseURL, req.GetGroup(), req.GetKey())

	// 创建HTTP请求
	httpReq, err := http.NewRequest(http.MethodPost, h.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置正确的Content-Type
	httpReq.Header.Set("Content-Type", "application/protobuf")

	// 发送HTTP POST请求
	res, err := h.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer res.Body.Close()

	// 检查响应状态
	if res.StatusCode == http.StatusNotFound {
		// 返回统一的"键不存在"错误
		return cache.ErrNotFound
	} else if res.StatusCode != http.StatusOK {
		// 读取错误响应内容，以便提供更详细的错误信息
		errBody, _ := io.ReadAll(res.Body)
		errMsg := string(errBody)

		// 根据错误消息判断错误类型
		if strings.Contains(errMsg, "key not found") ||
			strings.Contains(errMsg, "not found") ||
			strings.Contains(errMsg, "not exist") ||
			strings.Contains(errMsg, "本地未找到") ||
			strings.Contains(errMsg, "未找到") {
			return cache.ErrNotFound
		} else if strings.Contains(errMsg, "key is empty") ||
			strings.Contains(errMsg, "键为空") {
			return cache.ErrEmptyKey
		} else if strings.Contains(errMsg, "no such group") ||
			strings.Contains(errMsg, "group not found") ||
			strings.Contains(errMsg, "组不存在") {
			return cache.ErrNoSuchGroup
		}

		return fmt.Errorf("服务器返回错误: %v, 详情: %s", res.Status, errMsg)
	}

	// 读取响应体
	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	// 反序列化响应
	if err = proto.Unmarshal(respBody, resp); err != nil {
		return fmt.Errorf("反序列化响应失败: %v", err)
	}

	return nil
}

// ProtoGetter 专用于Protobuf通信的客户端
type ProtoGetter struct {
	baseURL    string     // 基础URL
	httpClient HTTPClient // HTTP客户端
}

// NewProtoGetter 创建新的Protobuf客户端
func NewProtoGetter(baseURL string) *ProtoGetter {
	return &ProtoGetter{
		baseURL:    baseURL,
		httpClient: defaultHTTPClient,
	}
}

// Get 通过HTTP获取缓存值
func (p *ProtoGetter) Get(group, key string) ([]byte, error) {
	// 构建Protobuf请求
	req := &pb.Request{
		Group: group,
		Key:   key,
	}

	// 发送Protobuf请求
	resp := &pb.Response{}
	if err := p.GetByProto(req, resp); err != nil {
		return nil, err
	}

	return resp.Value, nil
}

// GetByProto 通过Protobuf获取缓存值
func (p *ProtoGetter) GetByProto(req *pb.Request, resp *pb.Response) error {
	// 序列化请求
	body, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}

	// 使用baseURL作为请求地址
	logger.Debugf("发送Protobuf请求: %s (group=%s, key=%s)",
		p.baseURL, req.GetGroup(), req.GetKey())

	// 创建HTTP请求
	httpReq, err := http.NewRequest(http.MethodPost, p.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置正确的Content-Type
	httpReq.Header.Set("Content-Type", "application/protobuf")

	// 发送HTTP POST请求
	res, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer res.Body.Close()

	// 检查响应状态
	if res.StatusCode == http.StatusNotFound {
		// 返回统一的"键不存在"错误
		return cache.ErrNotFound
	} else if res.StatusCode != http.StatusOK {
		// 读取错误响应内容，以便提供更详细的错误信息
		errBody, _ := io.ReadAll(res.Body)
		errMsg := string(errBody)

		// 根据错误消息判断错误类型
		if strings.Contains(errMsg, "key not found") ||
			strings.Contains(errMsg, "not found") ||
			strings.Contains(errMsg, "not exist") ||
			strings.Contains(errMsg, "本地未找到") ||
			strings.Contains(errMsg, "未找到") {
			return cache.ErrNotFound
		} else if strings.Contains(errMsg, "key is empty") ||
			strings.Contains(errMsg, "键为空") {
			return cache.ErrEmptyKey
		} else if strings.Contains(errMsg, "no such group") ||
			strings.Contains(errMsg, "group not found") ||
			strings.Contains(errMsg, "组不存在") {
			return cache.ErrNoSuchGroup
		}

		return fmt.Errorf("服务器返回错误: %v, 详情: %s", res.Status, errMsg)
	}

	// 读取响应体
	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	// 反序列化响应
	if err = proto.Unmarshal(respBody, resp); err != nil {
		return fmt.Errorf("反序列化响应失败: %v", err)
	}

	return nil
}
