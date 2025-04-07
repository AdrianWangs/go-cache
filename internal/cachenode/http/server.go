package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/AdrianWangs/go-cache/internal/cache"
	"github.com/AdrianWangs/go-cache/pkg/logger"
)

// Server HTTP缓存服务器
type Server struct {
	addr       string         // 服务器地址
	httpServer *http.Server   // HTTP服务器
	mux        *http.ServeMux // HTTP路由
}

// NewServer 创建一个新的HTTP缓存服务器
func NewServer(addr string) *Server {
	mux := http.NewServeMux()

	server := &Server{
		addr: addr,
		httpServer: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		mux: mux,
	}

	// 注册默认路由处理程序
	server.registerHandlers()

	return server
}

// registerHandlers 注册HTTP路由处理程序
func (s *Server) registerHandlers() {
	// API路由: /api/cache/{group}/{key}
	s.mux.HandleFunc("/api/cache/", s.cacheHandler)

	// 状态检查路由
	s.mux.HandleFunc("/status", s.statusHandler)

	// 健康检查路由
	s.mux.HandleFunc("/health", s.healthHandler)
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	logger.Infof("HTTP缓存服务器正在监听: %s", s.addr)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("HTTP服务器运行错误: %v", err)
		}
	}()
	return nil
}

// Stop 停止HTTP服务器
func (s *Server) Stop() error {
	logger.Info("HTTP缓存服务器正在关闭")
	return s.httpServer.Close()
}

// cacheHandler 处理缓存请求
func (s *Server) cacheHandler(w http.ResponseWriter, r *http.Request) {
	// 解析路径: /api/cache/{group}/{key}
	parts := strings.SplitN(r.URL.Path[len("/api/cache/"):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "Bad Request: expected /api/cache/{group}/{key}", http.StatusBadRequest)
		return
	}

	groupName, key := parts[0], parts[1]

	// 获取对应的缓存组
	group := cache.GetGroup(groupName)
	if group == nil {
		http.Error(w, fmt.Sprintf("Group not found: %s", groupName), http.StatusNotFound)
		return
	}

	// 根据HTTP方法处理不同的请求
	switch r.Method {
	case http.MethodGet, "": // 默认为GET
		// 从缓存获取值
		view, err := group.Get(key)
		if err != nil {
			status := http.StatusInternalServerError
			if err == cache.ErrNotFound {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}

		// 设置响应头
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(view.ByteSlice())

	case http.MethodDelete:
		// 从缓存删除值
		err := group.Delete(key)
		if err != nil {
			status := http.StatusInternalServerError
			if err == cache.ErrEmptyKey {
				status = http.StatusBadRequest
			}
			http.Error(w, err.Error(), status)
			return
		}

		// 返回成功响应
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Deleted successfully"))

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// statusHandler 处理状态请求
func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	// 获取所有缓存组信息
	groups := cache.GetGroups()

	// 构建响应
	fmt.Fprintln(w, "Cache Status:")
	for name, group := range groups {
		stats := group.Stats()
		fmt.Fprintf(w, "Group: %s\n", name)
		fmt.Fprintf(w, "  - Hits: %d\n", stats.Hits)
		fmt.Fprintf(w, "  - Gets: %d\n", stats.Gets)
		if stats.Gets > 0 {
			fmt.Fprintf(w, "  - Hit Rate: %.2f%%\n", float64(stats.Hits)/float64(stats.Gets)*100)
		}
	}
}

// healthHandler 处理健康检查请求
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}
