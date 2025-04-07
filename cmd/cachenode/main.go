package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/AdrianWangs/go-cache/internal/cache"
	"github.com/AdrianWangs/go-cache/internal/cachenode/grpc"
	httpserver "github.com/AdrianWangs/go-cache/internal/cachenode/http"
	"github.com/AdrianWangs/go-cache/internal/discovery"
	"github.com/AdrianWangs/go-cache/internal/server"
	"github.com/AdrianWangs/go-cache/pkg/logger"
)

var (
	etcdEndpoints = flag.String("etcd-endpoints", "localhost:2379", "etcd集群地址，多个用逗号分隔")
	serviceName   = flag.String("service-name", "go-cache-nodes", "服务名称")
	nodeHost      = flag.String("node-host", "", "本节点主机名或IP地址（留空则自动检测）")
	nodePort      = flag.Int("node-port", 9090, "本节点gRPC监听端口")
	httpPort      = flag.Int("http-port", 9091, "本节点HTTP监听端口")
	apiAddr       = flag.String("api-addr", "localhost:8080", "API服务器地址")
	cacheSize     = flag.Int64("cache-size", 1024*1024*64, "缓存大小 (bytes)")
	groupName     = flag.String("group-name", "scores", "缓存组名称")
	leaseTTL      = flag.Int64("lease-ttl", 10, "etcd租约TTL（秒）")
	ttl           = flag.Int64("ttl", 0, "缓存过期时间（秒）")
)

// 模拟数据源
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// getLocalIP 获取本地非环回IP地址
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("无法找到本地非环回IP地址")
}

func main() {
	flag.Parse()

	endpoints := strings.Split(*etcdEndpoints, ",")
	if len(endpoints) == 0 || endpoints[0] == "" {
		logger.Fatal("etcd-endpoints 不能为空")
	}

	host := *nodeHost
	if host == "" {
		var err error
		host, err = getLocalIP()
		if err != nil {
			logger.Fatalf("自动获取本地IP失败: %v。请使用 -node-host 指定。", err)
		}
	}

	// gRPC地址
	grpcAddr := fmt.Sprintf("%s:%d", host, *nodePort)
	// HTTP地址
	httpAddr := fmt.Sprintf("%s:%d", host, *httpPort)

	logger.Info("缓存节点启动中...")
	logger.Infof("Etcd Endpoints: %v", endpoints)
	logger.Infof("服务名称: %s", *serviceName)
	logger.Infof("节点gRPC地址: %s", grpcAddr)
	logger.Infof("节点HTTP地址: %s", httpAddr)
	logger.Infof("租约 TTL: %ds", *leaseTTL)
	logger.Infof("API 服务器地址: %s", *apiAddr)
	logger.Infof("缓存组名称: %s", *groupName)
	logger.Infof("缓存大小: %d bytes", *cacheSize)

	// 创建ServiceDiscovery实例
	sd, err := discovery.NewServiceDiscovery(endpoints, *serviceName, grpcAddr, *leaseTTL)
	if err != nil {
		logger.Fatalf("创建Service Discovery失败: %v", err)
	}

	// 注册服务并启动心跳
	if err := sd.Register(); err != nil {
		logger.Fatalf("注册服务失败: %v", err)
	}
	defer func() {
		logger.Info("开始注销服务...")
		if err := sd.Unregister(); err != nil {
			logger.Errorf("注销服务失败: %v", err)
		} else {
			logger.Info("服务注销成功")
		}
		// 确保关闭连接
		if err := sd.Close(); err != nil {
			logger.Errorf("关闭etcd连接失败: %v", err)
		}
	}()

	logger.Infof("缓存节点 %s 已成功注册到etcd", grpcAddr)

	// --- 创建缓存逻辑 ---
	// 1. 创建缓存组
	getter := cache.GetterFunc(func(key string) ([]byte, error) {
		logger.Debugf("[本地数据源] 尝试获取 key: %s", key)
		if v, ok := db[key]; ok {
			logger.Debugf("[本地数据源] 找到 key: %s, value: %s", key, v)
			return []byte(v), nil
		}
		logger.Debugf("[本地数据源] 未找到 key: %s", key)
		return nil, fmt.Errorf("本地未找到 key: %s", key)
	})
	group := cache.NewGroup(*groupName, *cacheSize, getter, time.Duration(*ttl))

	// 2. 创建 HTTP Pool，显式设置 Protobuf 协议
	pool := server.NewHTTPPool(httpAddr,
		server.WithProtocol(server.ProtocolProtobuf), // 明确指定 Protobuf 协议
	)

	// 3. 注册 PeerPicker
	group.RegisterPeers(pool)

	// 4. 创建和启动 gRPC 服务器
	grpcServer := grpc.NewCacheServer(grpcAddr)
	if err := grpcServer.Start(); err != nil {
		logger.Fatalf("启动gRPC服务器失败: %v", err)
	}
	defer grpcServer.Stop()

	// 5. 创建和启动 HTTP 服务器 (提供API接口)
	httpServer := httpserver.NewServer(httpAddr)
	if err := httpServer.Start(); err != nil {
		logger.Fatalf("启动HTTP服务器失败: %v", err)
	}
	defer httpServer.Stop()

	// 6. 定期从 API Server 更新 Peer 列表
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 确保在退出时停止更新goroutine

	go func(ctx context.Context) {
		ticker := time.NewTicker(5 * time.Second) // 每5秒更新一次
		defer ticker.Stop()
		updatePeers(pool, *apiAddr) // 初始更新一次
		for {
			select {
			case <-ticker.C:
				updatePeers(pool, *apiAddr)
			case <-ctx.Done():
				logger.Info("停止更新 peer 列表")
				return
			}
		}
	}(ctx)

	logger.Infof("缓存节点已启动，提供 gRPC 服务于 %s 和 HTTP 服务于 %s", grpcAddr, httpAddr)

	// 优雅关机处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // 阻塞直到接收到停止信号

	logger.Info("收到停止信号，缓存节点开始关闭...")
	cancel() // 停止 peer 更新 goroutine
	// 在defer中处理了注销和关闭逻辑
	time.Sleep(1 * time.Second) // 等待注销完成
	logger.Info("缓存节点已关闭")
}

// --- 更新 Peer 列表的函数 ---
func updatePeers(pool *server.HTTPPool, apiAddr string) {
	// 构建API地址
	peerURL := fmt.Sprintf("http://%s/peers", apiAddr)
	resp, err := http.Get(peerURL)
	if err != nil {
		logger.Errorf("从 API Server (%s) 获取 peers 失败: %v", peerURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("从 API Server (%s) 获取 peers 失败，状态码: %d", peerURL, resp.StatusCode)
		return
	}

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取 API Server (%s) 响应失败: %v", peerURL, err)
		return
	}

	// 尝试使用JSON解析
	var result struct {
		Peers []string `json:"peers"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		logger.Warnf("解析JSON响应失败: %v，尝试使用旧的解析方式", err)

		// 兼容旧的解析逻辑
		bodyString := string(bodyBytes)
		bodyString = strings.TrimPrefix(bodyString, `{"peers": ["`)
		bodyString = strings.TrimSuffix(bodyString, `"]}`)
		if bodyString == "" { // 处理空列表的情况
			pool.Set() // 设置为空列表
			logger.Info("从 API Server 获取到空的 peer 列表")
			return
		}
		result.Peers = strings.Split(bodyString, `", "`)
	}

	// 如果获取到了peers列表
	if len(result.Peers) > 0 {
		// 更新 Pool 的 peers
		pool.Set(result.Peers...) // 使用解构赋值传入 slice
		logger.Infof("从 API Server (%s) 更新 peer 列表: %v", peerURL, result.Peers)
	} else {
		// 空列表情况
		pool.Set() // 设置为空列表
		logger.Info("从 API Server 获取到空的 peer 列表")
	}
}
