package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/AdrianWangs/go-cache/internal/cache"
	"github.com/AdrianWangs/go-cache/internal/interfaces"
	"github.com/AdrianWangs/go-cache/internal/server"
	"github.com/AdrianWangs/go-cache/pkg/logger"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *cache.Group {
	return cache.NewGroup("scores", 2<<10, interfaces.GetterFunc(
		func(key string) ([]byte, error) {
			logger.Debugf("[SlowDB] search key %s", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		},
	))
}

// 启动缓存服务器: 创建HTTPPool, 添加节点, 注册到gocache中
func startCacheServer(addr string, addrs []string, gocache *cache.Group) {
	peers := server.NewHTTPPool(addr)
	peers.Set(addrs...)
	gocache.RegisterPeers(peers)
	logger.Infof("go cache is running at %s", addr)
	http.ListenAndServe(addr[7:], peers)
}

// 启动API服务器: 创建HTTPPool, 添加节点, 注册到gocache中
func startAPIServer(apiAddr string, gocache *cache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			key := r.URL.Query().Get("key")
			view, err := gocache.Get(key)
			if err != nil {
				logger.Errorf("API server error: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		},
	))
	logger.Infof("frontend server is running at %s", apiAddr)
	logger.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	// 初始化日志
	logger.InitLogger("info")

	var port int
	var api bool

	flag.IntVar(&port, "port", 8001, "GoCache server port")
	flag.BoolVar(&api, "api", false, "Start API server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	// 创建group
	group := createGroup()

	// 启动API服务器
	if api {
		go startAPIServer(apiAddr, group)
	}

	// 启动缓存服务器
	startCacheServer(addrMap[port], addrs, group)
	select {}
}
