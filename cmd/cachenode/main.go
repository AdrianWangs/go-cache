package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/AdrianWangs/go-cache/internal/discovery" // 确保路径正确
)

var (
	etcdEndpoints = flag.String("etcd-endpoints", "localhost:2379", "etcd集群地址，多个用逗号分隔")
	serviceName   = flag.String("service-name", "go-cache-nodes", "服务名称")
	nodeHost      = flag.String("node-host", "", "本节点主机名或IP地址（留空则自动检测）")
	nodePort      = flag.Int("node-port", 9090, "本节点监听端口")
	leaseTTL      = flag.Int64("lease-ttl", 10, "etcd租约TTL（秒）")
)

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
		log.Fatal("etcd-endpoints 不能为空")
	}

	host := *nodeHost
	if host == "" {
		var err error
		host, err = getLocalIP()
		if err != nil {
			log.Fatalf("自动获取本地IP失败: %v。请使用 -node-host 指定。", err)
		}
	}

	nodeAddr := fmt.Sprintf("%s:%d", host, *nodePort)

	log.Printf("缓存节点启动中...")
	log.Printf("Etcd Endpoints: %v", endpoints)
	log.Printf("服务名称: %s", *serviceName)
	log.Printf("节点地址 (注册到etcd): %s", nodeAddr)
	log.Printf("租约 TTL: %ds", *leaseTTL)

	// 创建ServiceDiscovery实例
	sd, err := discovery.NewServiceDiscovery(endpoints, *serviceName, nodeAddr, *leaseTTL)
	if err != nil {
		log.Fatalf("创建Service Discovery失败: %v", err)
	}

	// 注册服务并启动心跳
	if err := sd.Register(); err != nil {
		log.Fatalf("注册服务失败: %v", err)
	}
	defer func() {
		log.Println("开始注销服务...")
		if err := sd.Unregister(); err != nil {
			log.Printf("注销服务失败: %v", err)
		} else {
			log.Println("服务注销成功")
		}
		// 确保关闭连接
		if err := sd.Close(); err != nil {
			log.Printf("关闭etcd连接失败: %v", err)
		}
	}()

	log.Printf("缓存节点 %s 已成功注册到etcd，正在运行...", nodeAddr)
	// 在这里可以启动该节点的实际缓存服务逻辑，例如监听端口等
	// 为了演示，这里仅阻塞等待退出信号

	// 优雅关机处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // 阻塞直到接收到停止信号

	log.Println("收到停止信号，缓存节点开始关闭...")
	// 在defer中处理了注销和关闭逻辑
	time.Sleep(1 * time.Second) // 等待注销完成
	log.Println("缓存节点已关闭")
}
