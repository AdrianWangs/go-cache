// Package handlers 实现API处理器
package handlers

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/AdrianWangs/go-cache/pkg/logger"
)

// NodeHandler 节点服务管理处理器
type NodeHandler struct {
	mu                sync.RWMutex
	nodeAddresses     []string       // 缓存节点地址列表
	serviceChangeHook func([]string) // 节点变更通知回调函数
}

// NodeResponse 节点信息响应
type NodeResponse struct {
	Count int      `json:"count"` // 节点数量
	Nodes []string `json:"nodes"` // 节点地址列表
}

// 旧版本响应格式，用于兼容
type LegacyPeersResponse struct {
	Peers []string `json:"peers"` // 节点地址列表
}

// NewNodeHandler 创建新的节点处理器
func NewNodeHandler() *NodeHandler {
	return &NodeHandler{
		nodeAddresses: make([]string, 0),
	}
}

// SetServiceChangeHook 设置节点变更通知回调
func (h *NodeHandler) SetServiceChangeHook(hook func([]string)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.serviceChangeHook = hook
}

// UpdateNodeAddresses 更新节点地址列表
func (h *NodeHandler) UpdateNodeAddresses(addresses []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 判断节点列表是否发生实质性变化
	if !isStringSliceEqual(h.nodeAddresses, addresses) {
		logger.Infof("节点列表更新，从 %d 个节点变为 %d 个节点", len(h.nodeAddresses), len(addresses))
		h.nodeAddresses = addresses

		// 触发回调通知
		if h.serviceChangeHook != nil {
			h.serviceChangeHook(h.getNodeAddresses())
		}
	}
}

// 获取节点地址列表的副本
func (h *NodeHandler) getNodeAddresses() []string {
	result := make([]string, len(h.nodeAddresses))
	copy(result, h.nodeAddresses)
	return result
}

// GetNodesHandler 获取当前节点列表
func (h *NodeHandler) GetNodesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.mu.RLock()
	nodes := h.getNodeAddresses()
	h.mu.RUnlock()

	// 检查路径，如果是/peers则使用旧的格式
	if r.URL.Path == "/peers" {
		response := LegacyPeersResponse{
			Peers: nodes,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Errorf("序列化旧格式节点列表响应失败: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		logger.Debugf("返回旧格式节点列表，共 %d 个节点", len(nodes))
		return
	}

	// 默认使用新的格式
	response := NodeResponse{
		Count: len(nodes),
		Nodes: nodes,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("序列化节点列表响应失败: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	logger.Debugf("返回节点列表，共 %d 个节点", len(nodes))
}

// HealthCheckHandler 健康检查处理器
func (h *NodeHandler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// 比较两个字符串切片是否相等
func isStringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// 创建映射表提高比较效率
	exist := make(map[string]bool)
	for _, v := range a {
		exist[v] = true
	}

	for _, v := range b {
		if !exist[v] {
			return false
		}
	}

	return true
}
