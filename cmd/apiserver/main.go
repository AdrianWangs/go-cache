package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/AdrianWangs/go-cache/api"
	"github.com/AdrianWangs/go-cache/api/handlers"
	"github.com/AdrianWangs/go-cache/pkg/logger"
)

var (
	etcdEndpoints = flag.String("etcd-endpoints", "localhost:2379", "etcd集群地址，多个用逗号分隔")
	serviceName   = flag.String("service-name", "go-cache-nodes", "要监视的服务名称")
	apiPort       = flag.Int("api-port", 8080, "API服务监听端口")
	replicas      = flag.Int("replicas", 3, "一致性哈希虚拟节点倍数")
	basePath      = flag.String("base-path", "/_gocache/", "缓存节点内部通信路径")
	protocol      = flag.String("protocol", "grpc", "通信协议 (http 或 grpc)")
)

func main() {
	flag.Parse()

	endpoints := strings.Split(*etcdEndpoints, ",")
	if len(endpoints) == 0 || endpoints[0] == "" {
		logger.Fatal("etcd-endpoints 不能为空")
	}

	// 检查协议类型
	var protocolType handlers.ProtocolType
	switch strings.ToLower(*protocol) {
	case "http":
		protocolType = handlers.ProtocolHTTP
	case "grpc":
		protocolType = handlers.ProtocolGRPC
	default:
		logger.Fatalf("不支持的协议类型: %s，只能是 http 或 grpc", *protocol)
	}

	logger.Info("API服务节点启动中...")
	logger.Infof("Etcd Endpoints: %v", endpoints)
	logger.Infof("监视的服务名称: %s", *serviceName)
	logger.Infof("API监听端口: %d", *apiPort)
	logger.Infof("一致性哈希虚拟节点倍数: %d", *replicas)
	logger.Infof("缓存节点内部通信路径: %s", *basePath)
	logger.Infof("使用通信协议: %s", protocolType)

	// 创建 ApiServer 配置
	cfg := &api.ApiServerConfig{
		EtcdEndpoints: endpoints,
		ServiceName:   *serviceName,
		ApiPort:       *apiPort,
		Replicas:      *replicas,
		BasePath:      *basePath,
		Protocol:      protocolType,
	}

	// 创建并启动 ApiServer
	apiServer, err := api.NewApiServer(cfg)
	if err != nil {
		logger.Fatalf("创建 API 服务失败: %v", err)
	}

	if err := apiServer.Start(); err != nil {
		logger.Fatalf("启动 API 服务失败: %v", err)
	}

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("收到停止信号，API服务开始关闭...")
	apiServer.Stop()
	logger.Info("API服务已关闭")
}
