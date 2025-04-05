package discovery

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ServiceDiscovery 用于向etcd注册服务和维持心跳
type ServiceDiscovery struct {
	cli        *clientv3.Client // etcd客户端
	leaseID    clientv3.LeaseID // 租约ID
	leaseTTL   int64            // 租约TTL（秒）
	key        string           // 服务注册的键
	value      string           // 服务注册的值（通常是地址）
	stopChan   chan struct{}    // 用于停止心跳的通道
	mu         sync.Mutex       // 保护对leaseID的访问
	registered bool             // 标记是否已成功注册
}

// NewServiceDiscovery 创建一个新的ServiceDiscovery实例
func NewServiceDiscovery(endpoints []string, serviceName, nodeAddr string, leaseTTL int64) (*ServiceDiscovery, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("连接etcd失败: %w", err)
	}

	sd := &ServiceDiscovery{
		cli:      cli,
		leaseTTL: leaseTTL,
		key:      fmt.Sprintf("/%s/%s", serviceName, nodeAddr), // 使用 /serviceName/nodeAddr 作为key
		value:    nodeAddr,
		stopChan: make(chan struct{}),
	}

	return sd, nil
}

// Register 注册服务并启动心跳续约
func (sd *ServiceDiscovery) Register() error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if sd.registered {
		return fmt.Errorf("服务 %s 已注册", sd.key)
	}

	// 1. 创建租约
	leaseResp, err := sd.cli.Grant(context.Background(), sd.leaseTTL)
	if err != nil {
		return fmt.Errorf("创建etcd租约失败: %w", err)
	}
	sd.leaseID = leaseResp.ID
	log.Printf("成功获取etcd租约，LeaseID: %x, TTL: %ds", sd.leaseID, sd.leaseTTL)

	// 2. 将服务信息与租约绑定并写入etcd
	_, err = sd.cli.Put(context.Background(), sd.key, sd.value, clientv3.WithLease(sd.leaseID))
	if err != nil {
		// 如果put失败，尝试撤销租约
		_, revokeErr := sd.cli.Revoke(context.Background(), sd.leaseID)
		if revokeErr != nil {
			log.Printf("警告：注册失败后撤销租约 %x 也失败: %v", sd.leaseID, revokeErr)
		}
		return fmt.Errorf("写入服务信息到etcd失败: %w", err)
	}

	// 3. 启动心跳续约
	keepAliveChan, err := sd.cli.KeepAlive(context.Background(), sd.leaseID)
	if err != nil {
		// 如果启动keepalive失败，尝试撤销租约和删除key
		log.Printf("启动etcd KeepAlive失败: %v。尝试清理...", err)
		sd.cleanupRegistration()
		return fmt.Errorf("启动etcd KeepAlive失败: %w", err)
	}

	go sd.keepAlive(keepAliveChan)
	sd.registered = true
	log.Printf("服务 %s (value: %s) 已成功注册到etcd，LeaseID: %x", sd.key, sd.value, sd.leaseID)
	return nil
}

// keepAlive 处理续约响应
func (sd *ServiceDiscovery) keepAlive(keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse) {
	log.Printf("心跳续约 goroutine 启动，监控 LeaseID: %x", sd.leaseID)
	for {
		select {
		case kaResp, ok := <-keepAliveChan:
			if !ok {
				log.Printf("KeepAlive通道关闭，LeaseID: %x 可能已过期或被撤销", sd.leaseID)
				// 可以在这里触发重新注册逻辑
				sd.mu.Lock()
				sd.registered = false // 标记为未注册
				sd.mu.Unlock()
				return // 结束goroutine
			}
			// 打印续约确认信息（可选，避免日志过多）
			// log.Printf("租约 %x 续约成功, TTL: %d", kaResp.ID, kaResp.TTL)
			_ = kaResp // 避免未使用变量错误
		case <-sd.stopChan:
			log.Printf("收到停止信号，停止对 LeaseID: %x 的心跳续约", sd.leaseID)
			return // 结束goroutine
		}
	}
}

// Unregister 注销服务（撤销租约）
func (sd *ServiceDiscovery) Unregister() error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if !sd.registered {
		log.Println("服务未注册或已注销，无需操作")
		return nil // 或者返回错误，取决于业务逻辑
	}

	// 停止心跳 goroutine
	close(sd.stopChan)

	// 撤销租约，etcd会自动删除关联的key
	_, err := sd.cli.Revoke(context.Background(), sd.leaseID)
	if err != nil {
		log.Printf("撤销etcd租约 %x 失败: %v", sd.leaseID, err)
		// 即使撤销失败，也标记为未注册，避免重复尝试
		sd.registered = false
		return fmt.Errorf("撤销etcd租约失败: %w", err)
	}

	sd.registered = false
	sd.leaseID = 0                    // 重置LeaseID
	sd.stopChan = make(chan struct{}) // 创建新的stopChan供下次注册使用
	log.Printf("服务 %s (原 LeaseID: %x) 已成功注销", sd.key, sd.leaseID)
	return nil
}

// cleanupRegistration 用于在注册过程中发生错误时清理资源
func (sd *ServiceDiscovery) cleanupRegistration() {
	if sd.leaseID != 0 {
		_, err := sd.cli.Revoke(context.Background(), sd.leaseID)
		if err != nil {
			log.Printf("清理：撤销租约 %x 失败: %v", sd.leaseID, err)
		}
		sd.leaseID = 0
	}
	// 尝试删除key，以防万一Revoke未完全生效或之前有残留
	_, err := sd.cli.Delete(context.Background(), sd.key)
	if err != nil {
		log.Printf("清理：删除etcd key %s 失败: %v", sd.key, err)
	}
}

// Close 关闭etcd客户端连接
func (sd *ServiceDiscovery) Close() error {
	if sd.cli != nil {
		return sd.cli.Close()
	}
	return nil
}

// --- Service Watcher --- //

// ServiceWatcher 用于监视etcd中特定服务下的节点变化
type ServiceWatcher struct {
	cli         *clientv3.Client
	serviceName string
	watchPrefix string
}

// NewServiceWatcher 创建一个新的ServiceWatcher实例
func NewServiceWatcher(endpoints []string, serviceName string) (*ServiceWatcher, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("连接etcd失败: %w", err)
	}

	sw := &ServiceWatcher{
		cli:         cli,
		serviceName: serviceName,
		watchPrefix: fmt.Sprintf("/%s/", serviceName), // 监视 /serviceName/ 前缀
	}
	return sw, nil
}

// Watch 启动对服务节点的监视
// 返回一个通道用于接收更新后的节点列表，以及一个错误通道
func (sw *ServiceWatcher) Watch(ctx context.Context) (<-chan []string, <-chan error) {
	updatesChan := make(chan []string)
	errChan := make(chan error, 1) // 带缓冲的错误通道，避免阻塞

	go func() {
		defer close(updatesChan)
		defer close(errChan)

		// 1. 先获取一次当前所有节点
		if err := sw.syncPeers(ctx, updatesChan); err != nil {
			errChan <- fmt.Errorf("首次同步节点列表失败: %w", err)
			return // 首次同步失败，直接退出goroutine
		}

		// 2. 创建Watch通道，监视指定前缀
		// 使用传入的ctx，以便外部可以取消watch
		wch := sw.cli.Watch(ctx, sw.watchPrefix, clientv3.WithPrefix())

		log.Printf("开始监视etcd前缀 '%s' 的变化...", sw.watchPrefix)

		for {
			select {
			case wresp, ok := <-wch:
				if !ok {
					log.Println("Etcd Watch通道已关闭 (可能是上下文取消或连接问题)")
					// 如果通道关闭，通常意味着上下文被取消或连接断开
					// 可以在这里尝试发送一个错误信号，或者依赖上下文取消来处理
					// errChan <- fmt.Errorf("etcd watch channel closed")
					return // 结束goroutine
				}
				if wresp.Err() != nil {
					log.Printf("Etcd Watch收到错误: %v", wresp.Err())
					errChan <- fmt.Errorf("etcd watch error: %w", wresp.Err())
					// 考虑是否需要在这里return或尝试重连
					continue // 继续等待下一个事件或错误
				}

				// 检查是否有事件发生 (PUT或DELETE)
				hasChanges := false
				for _, ev := range wresp.Events {
					// 只需要知道有变化即可，无需区分具体类型
					// log.Printf("Watch Event: Type: %s Key:%s Value:%s\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
					_ = ev // 避免未使用变量错误
					hasChanges = true
				}

				// 如果检测到变化，重新获取完整的节点列表并发送
				if hasChanges {
					log.Println("检测到etcd变化，重新同步节点列表...")
					if err := sw.syncPeers(ctx, updatesChan); err != nil {
						log.Printf("同步节点列表失败: %v", err)
						errChan <- fmt.Errorf("同步节点列表失败: %w", err)
						// 考虑是否需要在这里return或尝试重连
					}
				}

			case <-ctx.Done():
				log.Printf("Watch监视被取消 (context done)，停止监视前缀 '%s'", sw.watchPrefix)
				return // 结束goroutine
			}
		}
	}()

	return updatesChan, errChan
}

// syncPeers 获取当前所有节点并发送到updatesChan
func (sw *ServiceWatcher) syncPeers(ctx context.Context, updatesChan chan<- []string) error {
	resp, err := sw.cli.Get(ctx, sw.watchPrefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("从etcd获取服务列表失败: %w", err)
	}

	peers := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		peers = append(peers, string(kv.Value)) // 使用Value作为节点地址
	}

	// 发送更新后的列表到通道
	select {
	case updatesChan <- peers:
		log.Printf("已同步节点列表: %v", peers)
	case <-ctx.Done():
		return ctx.Err() // 上下文被取消
	}
	return nil
}

// Close 关闭etcd客户端连接
func (sw *ServiceWatcher) Close() error {
	if sw.cli != nil {
		return sw.cli.Close()
	}
	return nil
}
