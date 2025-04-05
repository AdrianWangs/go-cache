package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AdrianWangs/go-cache/internal/discovery" // 确保路径正确
)

var (
	etcdEndpoints = flag.String("etcd-endpoints", "localhost:2379", "etcd集群地址，多个用逗号分隔")
	serviceName   = flag.String("service-name", "go-cache-nodes", "要监视的服务名称")
	apiPort       = flag.Int("api-port", 8080, "API服务监听端口")
)

// peerList 存储当前活跃的peer节点地址
type peerList struct {
	mu    sync.RWMutex
	peers map[string]struct{} // 使用map方便快速查找和删除
}

func newPeerList() *peerList {
	return &peerList{
		peers: make(map[string]struct{}),
	}
}

// Update 更新peer列表
func (pl *peerList) Update(peers []string) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	newPeers := make(map[string]struct{}, len(peers))
	for _, p := range peers {
		newPeers[p] = struct{}{}
	}
	pl.peers = newPeers
	log.Printf("Peer列表已更新: %v", pl.getPeersLocked())
}

// GetPeers 返回当前peer列表的副本
func (pl *peerList) GetPeers() []string {
	pl.mu.RLock()
	defer pl.mu.RUnlock()
	return pl.getPeersLocked()
}

// getPeersLocked 返回peer列表（无锁版本，内部使用）
func (pl *peerList) getPeersLocked() []string {
	list := make([]string, 0, len(pl.peers))
	for p := range pl.peers {
		list = append(list, p)
	}
	return list
}

func main() {
	flag.Parse()

	endpoints := strings.Split(*etcdEndpoints, ",")
	if len(endpoints) == 0 || endpoints[0] == "" {
		log.Fatal("etcd-endpoints 不能为空")
	}

	log.Printf("API服务节点启动中...")
	log.Printf("Etcd Endpoints: %v", endpoints)
	log.Printf("监视的服务名称: %s", *serviceName)
	log.Printf("API监听端口: %d", *apiPort)

	peers := newPeerList()

	// 创建ServiceWatcher实例
	sw, err := discovery.NewServiceWatcher(endpoints, *serviceName)
	if err != nil {
		log.Fatalf("创建Service Watcher失败: %v", err)
	}

	// 启动goroutine来监视服务变化
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 确保在main函数退出时取消上下文

	go func() {
		log.Println("开始监视etcd中的服务变化...")
		updates, errChan := sw.Watch(ctx)
		for {
			select {
			case currentPeers, ok := <-updates:
				if !ok {
					log.Println("Etcd watch通道已关闭")
					return
				}
				log.Printf("检测到服务变化，当前节点列表: %v", currentPeers)
				peers.Update(currentPeers)
			case err := <-errChan:
				log.Printf("Etcd watch出错: %v。尝试重新连接...", err)
				// 简单的重试逻辑，实际应用中可能需要更复杂的退避策略
				time.Sleep(5 * time.Second)
				// 重新启动watch（注意：这里需要重新创建watcher或在其内部实现重连）
				// 为了简化，这里我们仅打印日志，实际项目中需要处理重连
				log.Println("错误处理：需要实现watch重连逻辑")
				// 或者选择退出
				// cancel()
				// return
			case <-ctx.Done():
				log.Println("Etcd watch被取消")
				return
			}
		}
	}()

	// 设置HTTP API路由
	http.HandleFunc("/peers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		currentPeers := peers.GetPeers()
		w.Header().Set("Content-Type", "application/json")
		// 简单地将列表以逗号分隔字符串返回，实际应用可能返回JSON数组
		fmt.Fprintf(w, `{"peers": ["%s"]}`, strings.Join(currentPeers, `", "`))
	})

	// 启动HTTP服务器
	serverAddr := fmt.Sprintf(":%d", *apiPort)
	httpServer := &http.Server{Addr: serverAddr}

	go func() {
		log.Printf("API服务器正在监听 %s", serverAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("无法启动HTTP服务器: %v", err)
		}
	}()

	// 优雅关机处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("收到停止信号，API服务开始关闭...")

	// 停止etcd监视
	cancel() // 发送取消信号给watch goroutine

	// 关闭HTTP服务器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP服务器关闭失败: %v", err)
	}

	// 关闭etcd watcher连接
	if err := sw.Close(); err != nil {
		log.Printf("关闭etcd watcher连接失败: %v", err)
	}

	log.Println("API服务已关闭")
}
