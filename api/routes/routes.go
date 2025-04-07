// Package routes 实现API路由管理
package routes

import (
	"net/http"

	"github.com/AdrianWangs/go-cache/api/handlers"
	"github.com/AdrianWangs/go-cache/pkg/logger"
	"github.com/AdrianWangs/go-cache/pkg/router"
)

// RegisterRoutes 注册所有API路由
func RegisterRoutes(r *router.Router, cacheHandler *handlers.CacheHandler,
	nodeHandler *handlers.NodeHandler, metricsHandler *handlers.MetricsHandler) {

	logger.Info("正在注册API路由...")

	// 注册健康检查路由
	r.RegisterFunc("/health", nodeHandler.HealthCheckHandler)

	// 兼容性路由 - 旧的 /peers 接口
	r.RegisterFunc("/peers", nodeHandler.GetNodesHandler)

	// 注册API路由组
	apiGroup := r.Group("/api")

	// 缓存路由组
	cacheRoutes := apiGroup.Group("/cache")
	// 同时支持GET和DELETE方法
	cacheRoutes.RegisterFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == "" {
			cacheHandler.GetCacheHandler(w, r)
		} else if r.Method == http.MethodDelete {
			cacheHandler.DeleteCacheHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 节点路由组
	nodeRoutes := apiGroup.Group("/nodes")
	nodeRoutes.RegisterFunc("", nodeHandler.GetNodesHandler)

	// 监控指标路由组
	metricsRoutes := apiGroup.Group("/metrics")
	metricsRoutes.RegisterFunc("", metricsHandler.GetMetricsHandler)

	logger.Info("API路由注册完成")
}
