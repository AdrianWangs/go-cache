// Package api 提供API服务器实现
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/AdrianWangs/go-cache/api/handlers"
	"github.com/AdrianWangs/go-cache/api/routes"
	"github.com/AdrianWangs/go-cache/internal/discovery"
	"github.com/AdrianWangs/go-cache/pkg/logger"
	"github.com/AdrianWangs/go-cache/pkg/router"
)

// ApiServerConfig API服务器配置
type ApiServerConfig struct {
	EtcdEndpoints []string              // Etcd服务地址
	ServiceName   string                // 缓存节点服务名称
	ApiPort       int                   // API服务器端口
	Replicas      int                   // 虚拟节点倍数
	BasePath      string                // 内部通信路径
	Protocol      handlers.ProtocolType // 通信协议类型
}

// ApiServer API服务器
type ApiServer struct {
	config         *ApiServerConfig          // 配置
	serviceWatcher *discovery.ServiceWatcher // 服务发现
	httpServer     *http.Server              // HTTP服务器
	router         *router.Router            // 路由器
	cacheHandler   *handlers.CacheHandler    // 缓存处理器
	nodeHandler    *handlers.NodeHandler     // 节点处理器
	metricsHandler *handlers.MetricsHandler  // 指标处理器
	cancelWatch    context.CancelFunc        // 用于取消服务发现
}

// NewApiServer 创建新的API服务器
func NewApiServer(config *ApiServerConfig) (*ApiServer, error) {
	if config == nil {
		return nil, fmt.Errorf("API服务器配置不能为空")
	}

	// 创建服务发现
	serviceWatcher, err := discovery.NewServiceWatcher(config.EtcdEndpoints, config.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("创建服务发现失败: %v", err)
	}

	// 设置默认协议
	if config.Protocol == "" {
		config.Protocol = handlers.ProtocolHTTP
	}

	// 创建处理器
	cacheHandler := handlers.NewCacheHandler(config.BasePath, config.Replicas, handlers.CacheHandlerOptions{
		Protocol: config.Protocol,
	})
	nodeHandler := handlers.NewNodeHandler()
	metricsHandler := handlers.NewMetricsHandler()

	// 设置节点变更回调
	nodeHandler.SetServiceChangeHook(func(nodes []string) {
		// 当节点列表变化时更新缓存处理器中的节点列表
		if config.Protocol == handlers.ProtocolGRPC {
			// 使用gRPC getter
			cacheHandler.UpdatePeers(nodes, func(addr string) handlers.NodeGetter {
				return handlers.NewGRPCGetter(addr)
			})
		} else {
			// 使用HTTP getter
			cacheHandler.UpdatePeers(nodes, func(baseURL string) handlers.NodeGetter {
				return handlers.NewHTTPGetter(baseURL)
			})
		}
	})

	// 创建路由器
	r := router.New()

	// 添加中间件 (示例日志和指标中间件)
	r.Use(router.LoggingMiddleware())
	r.Use(router.RecoveryMiddleware())
	r.Use(func(h router.Handler) router.Handler {
		return router.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			metricsHandler.IncrementRequestCount() // 记录请求次数
			h.ServeHTTP(w, req)
		})
	})

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.ApiPort),
		Handler: r,
	}

	return &ApiServer{
		config:         config,
		serviceWatcher: serviceWatcher,
		httpServer:     server,
		router:         r,
		cacheHandler:   cacheHandler,
		nodeHandler:    nodeHandler,
		metricsHandler: metricsHandler,
	}, nil
}

// Start 启动API服务器
func (s *ApiServer) Start() error {
	// 注册路由
	routes.RegisterRoutes(s.router, s.cacheHandler, s.nodeHandler, s.metricsHandler)

	// 创建用于服务发现的上下文
	watchCtx, cancelWatch := context.WithCancel(context.Background())
	s.cancelWatch = cancelWatch // 保存取消函数，用于Stop时调用

	// 启动服务发现
	go func() {
		logger.Info("启动服务发现...")
		updatesChan, errChan := s.serviceWatcher.Watch(watchCtx)
		for {
			select {
			case services, ok := <-updatesChan:
				if !ok {
					logger.Warn("服务发现更新通道已关闭")
					return
				}
				logger.Infof("发现服务变化，当前有 %d 个节点: %v", len(services), services)
				s.nodeHandler.UpdateNodeAddresses(services)
			case err, ok := <-errChan:
				if !ok {
					logger.Warn("服务发现错误通道已关闭")
					return
				}
				logger.Errorf("服务发现遇到错误: %v", err)
				// 这里可以添加重试逻辑或退出
			case <-watchCtx.Done():
				logger.Info("服务发现已停止 (context canceled)")
				return
			}
		}
	}()

	// 启动HTTP服务器
	logger.Infof("API服务器启动在 http://localhost:%d", s.config.ApiPort)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Errorf("HTTP服务器启动失败: %v", err)
		return fmt.Errorf("HTTP服务器启动失败: %w", err)
	}
	return nil
}

// Stop 停止API服务器
func (s *ApiServer) Stop() error {
	logger.Info("正在停止API服务器...")

	// 停止服务发现
	if s.cancelWatch != nil {
		s.cancelWatch()
		logger.Info("已发送停止信号给服务发现")
	}

	// 创建一个有超时的上下文用于HTTP服务器关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅地关闭HTTP服务器
	if err := s.httpServer.Shutdown(ctx); err != nil {
		logger.Errorf("HTTP服务器关闭失败: %v", err)
		// 即使关闭失败，也要继续关闭其他资源
	}

	// 关闭服务发现客户端连接 (如果需要，可以放在最后)
	if s.serviceWatcher != nil {
		if err := s.serviceWatcher.Close(); err != nil {
			logger.Errorf("服务发现客户端关闭失败: %v", err)
		}
	}

	logger.Info("API服务器已停止")
	return nil
}
