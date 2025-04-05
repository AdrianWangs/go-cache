// Package api provides HTTP API for the cache service
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/AdrianWangs/go-cache/internal/cache"
	"github.com/AdrianWangs/go-cache/pkg/logger"
)

const (
	defaultAPITimeout = 5 * time.Second
)

// Server represents an API server for the cache
type Server struct {
	addr   string
	server *http.Server
	cancel context.CancelFunc
}

// NewServer creates a new API server
func NewServer(addr string) *Server {
	return &Server{
		addr: addr,
	}
}

// RegisterEndpoints registers HTTP endpoints for the cache
func (s *Server) RegisterEndpoints(mux *http.ServeMux, cacheGroups ...*cache.Group) {
	// Register health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Register cache endpoint
	mux.HandleFunc("/api/cache", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		group := r.URL.Query().Get("group")
		if group == "" {
			http.Error(w, "group parameter is required", http.StatusBadRequest)
			return
		}

		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "key parameter is required", http.StatusBadRequest)
			return
		}

		cacheGroup := cache.GetGroup(group)
		if cacheGroup == nil {
			http.Error(w, fmt.Sprintf("group %s not found", group), http.StatusNotFound)
			return
		}

		data, err := cacheGroup.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data.ByteSlice())
	})

	// Register metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("metrics will be implemented here"))
	})
}

// Start starts the API server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	s.RegisterEndpoints(mux)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	logger.Infof("API server starting on %s", s.addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("API server error: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		logger.Info("Shutting down API server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			logger.Errorf("Error shutting down API server: %v", err)
		}
	}()

	return nil
}

// Stop stops the API server
func (s *Server) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}
