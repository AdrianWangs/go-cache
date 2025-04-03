package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/AdrianWangs/go-cache/go_cache"
	"github.com/AdrianWangs/go-cache/interfaces"
	"github.com/AdrianWangs/go-cache/server"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *go_cache.Group {
	return go_cache.NewGroup("scores", 2<<10, interfaces.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		},
	))
}

// 启动缓存服务器: 创建HTTPPool, 添加节点, 注册到gocache中
func startCacheServer(addr string, addrs []string, gocache *go_cache.Group) {
	peers := server.NewHTTPPool(addr)
	peers.Set(addrs...)
	gocache.RegisterPeers(peers)
	log.Println("go cache is running at", addr)
	http.ListenAndServe(addr[7:], peers)
}

// 启动API服务器: 创建HTTPPool, 添加节点, 注册到gocache中
func startAPIServer(apiAddr string, gocache *go_cache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			key := r.URL.Query().Get("key")
			view, err := gocache.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		},
	))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}
func main() {

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
