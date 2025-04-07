// Package handlers 实现API处理器
package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/AdrianWangs/go-cache/pkg/logger"
)

// MetricsHandler 系统指标处理器
type MetricsHandler struct {
	mu           sync.RWMutex
	startTime    time.Time // 服务启动时间
	requestCount int64     // 总请求次数
	hitCount     int64     // 缓存命中次数
	missCount    int64     // 缓存未命中次数
}

// MetricsResponse 系统指标响应
type MetricsResponse struct {
	Uptime       string  `json:"uptime"`       // 运行时间
	NumGoroutine int     `json:"numGoroutine"` // goroutine数量
	RequestCount int64   `json:"requestCount"` // 总请求次数
	HitCount     int64   `json:"hitCount"`     // 缓存命中次数
	MissCount    int64   `json:"missCount"`    // 缓存未命中次数
	HitRate      float64 `json:"hitRate"`      // 缓存命中率
}

// NewMetricsHandler 创建新的指标处理器
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{
		startTime: time.Now(),
	}
}

// IncrementRequestCount 增加请求计数
func (h *MetricsHandler) IncrementRequestCount() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.requestCount++
}

// IncrementHitCount 增加命中计数
func (h *MetricsHandler) IncrementHitCount() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hitCount++
}

// IncrementMissCount 增加未命中计数
func (h *MetricsHandler) IncrementMissCount() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.missCount++
}

// GetMetricsHandler 获取系统指标
func (h *MetricsHandler) GetMetricsHandler(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	requestCount := h.requestCount
	hitCount := h.hitCount
	missCount := h.missCount
	uptime := time.Since(h.startTime).String()
	h.mu.RUnlock()

	// 计算命中率
	var hitRate float64
	if requestCount > 0 {
		hitRate = float64(hitCount) / float64(requestCount) * 100
	}

	metrics := MetricsResponse{
		Uptime:       uptime,
		NumGoroutine: runtime.NumGoroutine(),
		RequestCount: requestCount,
		HitCount:     hitCount,
		MissCount:    missCount,
		HitRate:      hitRate,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		logger.Errorf("序列化指标响应失败: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	logger.Debugf("返回系统指标，请求次数: %d, 命中率: %.2f%%", requestCount, hitRate)
}
