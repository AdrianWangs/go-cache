package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/AdrianWangs/go-cache/pkg/logger"
)

// LoggingMiddleware 创建一个记录请求日志的中间件
func LoggingMiddleware() MiddlewareFunc {
	return func(next Handler) Handler {
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// 包装ResponseWriter以捕获状态码
			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // 默认状态码
			}

			// 处理请求
			next.ServeHTTP(wrapper, r)

			// 计算请求处理时间
			duration := time.Since(start)

			// 记录请求信息
			logger.Infof("%s %s %d %s",
				r.Method,
				r.URL.Path,
				wrapper.statusCode,
				duration,
			)
		})
	}
}

// RecoveryMiddleware 创建一个恢复中间件，防止程序崩溃
func RecoveryMiddleware() MiddlewareFunc {
	return func(next Handler) Handler {
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// 记录错误
					logger.Errorf("处理请求 %s 时发生错误: %v", r.URL.Path, err)

					// 返回500错误
					http.Error(w,
						fmt.Sprintf("Internal Server Error: %v", err),
						http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// MethodMiddleware 创建一个检查HTTP方法的中间件
func MethodMiddleware(method string) MiddlewareFunc {
	return func(next Handler) Handler {
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriterWrapper 包装http.ResponseWriter以捕获状态码
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 重写WriteHeader方法以捕获状态码
func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
