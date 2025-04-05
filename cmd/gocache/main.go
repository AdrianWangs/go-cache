// Package main is the entry point for the go-cache application
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/AdrianWangs/go-cache/api"
	"github.com/AdrianWangs/go-cache/config"
	"github.com/AdrianWangs/go-cache/internal/cache"
	"github.com/AdrianWangs/go-cache/internal/server"
	"github.com/AdrianWangs/go-cache/pkg/logger"
)

// Simple in-memory database for demo
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	// Parse command line flags
	var (
		configFile string
		port       int
		apiServer  bool
		logLevel   string
	)

	flag.StringVar(&configFile, "config", "", "Path to config file")
	flag.IntVar(&port, "port", 0, "Port to run cache server on (overrides config)")
	flag.BoolVar(&apiServer, "api", false, "Start API server")
	flag.StringVar(&logLevel, "log", "", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Load configuration
	var cfg *config.Config
	if configFile != "" {
		var err error
		cfg, err = config.LoadFromFile(configFile)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
	} else {
		cfg = config.LoadFromEnv()
	}

	// Override with command line flags if provided
	if port != 0 {
		cfg.CachePort = port
	}
	if logLevel != "" {
		cfg.LogLevel = logLevel
	}

	// Initialize logger
	logger.SetLevel(cfg.LogLevel)
	if cfg.LogFormat == "json" {
		logger.UseJSONFormat()
	}

	// Create a cache group
	group := createGroup(cfg.MaxCacheBytes)

	// Determine this node's address
	selfAddr := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.CachePort)

	// Start cache server
	cachePool := server.NewHTTPPool(selfAddr,
		server.WithBasePath(cfg.BasePath),
		server.WithProtocol(server.ProtocolProtobuf),
	)
	cachePool.Set(cfg.PeerAddresses...)
	group.RegisterPeers(cachePool)

	go func() {
		err := cachePool.Start(cfg.Host, cfg.CachePort)
		if err != nil {
			logger.Errorf("Failed to start cache server: %v", err)
			os.Exit(1)
		}
	}()

	logger.Infof("Cache server running at http://%s:%d", cfg.Host, cfg.CachePort)

	// Start API server if requested
	var apiSrv *api.Server
	if apiServer {
		apiAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.APIPort)
		apiSrv = api.NewServer(apiAddr)
		err := apiSrv.Start()
		if err != nil {
			logger.Errorf("Failed to start API server: %v", err)
			os.Exit(1)
		}
		logger.Infof("API server running at http://%s:%d", cfg.Host, cfg.APIPort)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	logger.Info("Shutting down...")
	if apiSrv != nil {
		apiSrv.Stop()
	}
	cachePool.Stop()
}

// createGroup creates a new cache group
func createGroup(cacheBytes int64) *cache.Group {
	return cache.NewGroup("scores", cacheBytes, cache.GetterFunc(
		func(key string) ([]byte, error) {
			logger.Debugf("[SlowDB] search key %s", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not found in database", key)
		},
	))
}
