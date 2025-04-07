// Package router 提供统一的HTTP路由管理
package router

import (
	"net/http"
	"strings"

	"github.com/AdrianWangs/go-cache/pkg/logger"
)

// Handler 是一个包装了http.HandlerFunc的接口
// 支持中间件和其他路由功能
type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// HandlerFunc 是一个处理HTTP请求的函数类型
type HandlerFunc func(w http.ResponseWriter, r *http.Request)

// ServeHTTP 实现http.Handler接口
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// Router 是一个HTTP路由器，负责路由注册和分发请求
type Router struct {
	mux         *http.ServeMux
	routes      map[string]Handler
	middlewares []MiddlewareFunc
}

// MiddlewareFunc 是一个中间件函数类型
type MiddlewareFunc func(Handler) Handler

// New 创建一个新的路由器
func New() *Router {
	return &Router{
		mux:         http.NewServeMux(),
		routes:      make(map[string]Handler),
		middlewares: make([]MiddlewareFunc, 0),
	}
}

// Use 添加一个全局中间件
func (r *Router) Use(middleware MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middleware)
}

// Register 注册路由和处理器
func (r *Router) Register(pattern string, handler Handler) {
	// 应用中间件
	finalHandler := handler
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		finalHandler = r.middlewares[i](finalHandler)
	}

	// 注册到标准ServeMux
	r.mux.Handle(pattern, finalHandler)
	r.routes[pattern] = handler

	logger.Infof("已注册路由: %s", pattern)
}

// RegisterFunc 注册一个处理函数
func (r *Router) RegisterFunc(pattern string, handlerFunc HandlerFunc) {
	r.Register(pattern, handlerFunc)
}

// Group 创建一个子路由组
func (r *Router) Group(prefix string) *RouterGroup {
	return &RouterGroup{
		prefix: prefix,
		router: r,
	}
}

// ServeHTTP 实现http.Handler接口，将请求转发给ServeMux
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// RouterGroup 表示一个路由组，所有注册的路由都将添加相同的前缀
type RouterGroup struct {
	prefix      string
	router      *Router
	middlewares []MiddlewareFunc
}

// Use 为当前路由组添加中间件
func (g *RouterGroup) Use(middleware MiddlewareFunc) {
	g.middlewares = append(g.middlewares, middleware)
}

// Group 创建一个嵌套路由组
func (g *RouterGroup) Group(relPrefix string) *RouterGroup {
	return &RouterGroup{
		prefix: joinPaths(g.prefix, relPrefix),
		router: g.router,
	}
}

// Register 在路由组注册路由
func (g *RouterGroup) Register(pattern string, handler Handler) {
	fullPattern := joinPaths(g.prefix, pattern)

	// 应用组中间件
	finalHandler := handler
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		finalHandler = g.middlewares[i](finalHandler)
	}

	g.router.Register(fullPattern, finalHandler)
}

// RegisterFunc 在路由组注册处理函数
func (g *RouterGroup) RegisterFunc(pattern string, handlerFunc HandlerFunc) {
	g.Register(pattern, handlerFunc)
}

// 工具函数，连接两个路径
func joinPaths(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}

	aSlash := strings.HasSuffix(a, "/")
	bSlash := strings.HasPrefix(b, "/")

	switch {
	case aSlash && bSlash:
		return a + b[1:]
	case !aSlash && !bSlash:
		return a + "/" + b
	default:
		return a + b
	}
}
