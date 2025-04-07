// Package handlers 实现各种API处理器
package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/AdrianWangs/go-cache/internal/cache"
	"github.com/AdrianWangs/go-cache/internal/consistenthash"
	"github.com/AdrianWangs/go-cache/pkg/logger"
	pb "github.com/AdrianWangs/go-cache/proto/cache_server"
)

// CacheHandler 缓存处理器，处理缓存相关的请求
type CacheHandler struct {
	mu          sync.RWMutex
	basePath    string                // 缓存节点内部通信路径
	ring        *consistenthash.Map   // 一致性哈希环
	replicas    int                   // 虚拟节点倍数
	nodeGetters map[string]NodeGetter // 节点地址到 NodeGetter 的映射
}

// NodeGetter 统一了获取缓存节点数据的接口
type NodeGetter interface {
	// Get 返回指定组和键的值
	Get(group string, key string) ([]byte, error)
	// GetByProto 使用 protobuf 获取指定请求的值
	GetByProto(req *pb.Request, resp *pb.Response) error
}

// NewCacheHandler 创建新的缓存处理器
func NewCacheHandler(basePath string, replicas int) *CacheHandler {
	return &CacheHandler{
		basePath:    basePath,
		replicas:    replicas,
		ring:        consistenthash.New(replicas, nil),
		nodeGetters: make(map[string]NodeGetter),
	}
}

// UpdatePeers 更新节点列表和一致性哈希环
func (h *CacheHandler) UpdatePeers(peers []string, getterFactory func(baseURL string) NodeGetter) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 重建一致性哈希环
	h.ring = consistenthash.New(h.replicas, nil)
	h.ring.Add(peers...)

	// 更新 node getters
	newGetters := make(map[string]NodeGetter)
	for _, peer := range peers {
		if getter, ok := h.nodeGetters[peer]; ok {
			// 复用现有的 getter
			newGetters[peer] = getter
		} else {
			// 为新节点创建 getter
			baseURL := fmt.Sprintf("http://%s%s", peer, h.basePath)
			newGetters[peer] = getterFactory(baseURL)
			logger.Infof("为节点 %s 创建新的 getter (URL: %s)", peer, baseURL)
		}
	}
	h.nodeGetters = newGetters
}

// GetNodeGetters 获取所有节点getter
func (h *CacheHandler) GetNodeGetters() map[string]NodeGetter {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 创建一个副本
	result := make(map[string]NodeGetter, len(h.nodeGetters))
	for k, v := range h.nodeGetters {
		result[k] = v
	}
	return result
}

// GetCacheHandler 处理 /cache/{group}/{key} 或 /api/cache/{group}/{key} 请求
func (h *CacheHandler) GetCacheHandler(w http.ResponseWriter, r *http.Request) {
	// 解析 URL 路径
	parts := h.parseCachePath(r.URL.Path)
	if parts == nil {
		http.Error(w, "Bad Request: expected /cache/{group}/{key} or /api/cache/{group}/{key}", http.StatusBadRequest)
		return
	}

	groupName, key := parts[0], parts[1]
	logger.Debugf("收到缓存请求: group=%s, key=%s", groupName, key)

	// 根据 key 选择节点
	nodeAddr, getter := h.pickNode(key)
	if getter == nil {
		http.Error(w, "No suitable cache node available", http.StatusServiceUnavailable)
		logger.Warnf("无法为 key '%s' 找到合适的缓存节点", key)
		return
	}

	logger.Debugf("选择节点 %s 处理 key=%s (group=%s)", nodeAddr, key, groupName)

	// 创建 protobuf 请求
	req := &pb.Request{
		Group: groupName,
		Key:   key,
	}
	res := &pb.Response{}

	// 发送请求到选中的节点
	err := getter.GetByProto(req, res)
	if err != nil {
		// 使用错误类型比较
		errMsg := err.Error()

		// 我们有两种情况：
		// 1. 错误可能是我们自己的CacheError
		// 2. 错误可能是从远程节点返回的错误消息

		// 先尝试使用错误类型系统判断
		if errors.Is(err, cache.ErrNotFound) || cache.IsKeyNotFoundError(err) {
			// 键不存在错误
			http.Error(w, fmt.Sprintf("Key not found: %s", key), http.StatusNotFound)
			logger.Warnf("键不存在: %s (group=%s)", key, groupName)
		} else if errors.Is(err, cache.ErrEmptyKey) || cache.IsKeyEmptyError(err) {
			// 键为空错误
			http.Error(w, "Key is empty", http.StatusBadRequest)
			logger.Warnf("键为空错误: %s", errMsg)
		} else if errors.Is(err, cache.ErrNoSuchGroup) || cache.IsGroupNotFoundError(err) {
			// 组不存在错误
			http.Error(w, fmt.Sprintf("Group not found: %s", groupName), http.StatusNotFound)
			logger.Warnf("组不存在: %s", groupName)
		} else if strings.Contains(errMsg, "key not found") ||
			strings.Contains(errMsg, "not found") ||
			strings.Contains(errMsg, "not exist") ||
			strings.Contains(errMsg, "本地未找到") ||
			strings.Contains(errMsg, "未找到") {
			// 通过错误消息判断是键不存在（兼容来自远程节点的错误消息）
			http.Error(w, fmt.Sprintf("Key not found: %s", key), http.StatusNotFound)
			logger.Warnf("键不存在: %s (group=%s)", key, groupName)
		} else if strings.Contains(errMsg, "key is empty") ||
			strings.Contains(errMsg, "键为空") {
			// 通过错误消息判断是键为空
			http.Error(w, "Key is empty", http.StatusBadRequest)
			logger.Warnf("键为空错误: %s", errMsg)
		} else if strings.Contains(errMsg, "no such group") ||
			strings.Contains(errMsg, "group not found") ||
			strings.Contains(errMsg, "组不存在") {
			// 通过错误消息判断是组不存在
			http.Error(w, fmt.Sprintf("Group not found: %s", groupName), http.StatusNotFound)
			logger.Warnf("组不存在: %s", groupName)
		} else {
			// 其他类型的错误仍然返回500
			http.Error(w, fmt.Sprintf("Failed to get data: %v", err), http.StatusInternalServerError)
			logger.Errorf("从节点 %s 获取数据失败: %v", nodeAddr, err)
		}
		return
	}

	// 返回响应
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(res.Value)
	logger.Debugf("成功从节点 %s 获取数据, 长度: %d bytes", nodeAddr, len(res.Value))
}

// 解析缓存路径 /cache/{group}/{key} 或 /api/cache/{group}/{key}
func (h *CacheHandler) parseCachePath(path string) []string {
	parts := strings.Split(path, "/")

	// 移除空字符串元素（由分割"/"产生）
	var nonEmptyParts []string
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}

	// 检查路径格式
	if len(nonEmptyParts) < 3 {
		return nil
	}

	// 处理 /api/cache/{group}/{key} 格式
	if len(nonEmptyParts) >= 4 && nonEmptyParts[0] == "api" && nonEmptyParts[1] == "cache" {
		return []string{nonEmptyParts[2], nonEmptyParts[3]} // [group, key]
	}

	// 处理 /cache/{group}/{key} 格式
	if len(nonEmptyParts) >= 3 && nonEmptyParts[0] == "cache" {
		return []string{nonEmptyParts[1], nonEmptyParts[2]} // [group, key]
	}

	return nil
}

// 根据 key 选择节点和对应的 getter
func (h *CacheHandler) pickNode(key string) (string, NodeGetter) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.nodeGetters) == 0 {
		return "", nil
	}

	node := h.ring.Get(key)
	if node == "" {
		return "", nil
	}

	if getter, ok := h.nodeGetters[node]; ok {
		return node, getter
	}

	return "", nil
}
