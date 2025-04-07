# 服务发现 (Etcd)

本项目采用 `etcd` 作为服务发现的核心组件，以实现缓存节点 (`cachenode`) 的动态注册与发现，取代了手动配置节点列表的方式。

服务发现主要涉及两个部分：节点的注册与心跳 (`ServiceDiscovery`) 和节点的监控与更新 (`ServiceWatcher`)。

## 服务注册与心跳 (`internal/discovery.ServiceDiscovery`)

当一个 `cachenode` 启动时，它会执行以下操作来将自己注册到 `etcd` 并维持其存在：

1.  **连接 Etcd**: 创建一个 `etcd` 客户端实例 (`clientv3.New`)，连接到配置的 `etcd` 集群地址。
2.  **创建租约 (Lease)**: 调用 `cli.Grant()` 向 `etcd` 请求一个租约，并指定一个 TTL (Time-To-Live)。租约有一个唯一的 `LeaseID`。
3.  **绑定键值对与租约**: 调用 `cli.Put()` 将节点的服务信息写入 `etcd`。写入的 `key` 通常格式为 `/服务名/节点地址` (e.g., `/go-cache-nodes/192.168.1.10:9090`)，`value` 为节点地址。在 `Put` 操作中，通过 `clientv3.WithLease(leaseID)` 将这个键值对与之前创建的租约关联起来。
4.  **启动心跳 (KeepAlive)**: 调用 `cli.KeepAlive()` 启动一个后台 goroutine，该 goroutine 会定期向 `etcd` 发送心跳信号，以确保持有关联键值对的租约不会过期。只要节点存活且心跳正常，`etcd` 中的注册信息就保持有效。
5.  **处理续约响应**: `keepAlive` goroutine 会监听来自 `etcd` 的续约响应通道 (`keepAliveChan`)。如果通道关闭（可能由于网络问题或租约已被撤销），则认为注册失效，并可能触发重新注册逻辑。

### 服务注销

当 `cachenode` 优雅关闭时（例如收到 `SIGINT` 或 `SIGTERM` 信号），会执行以下操作：

1.  **停止心跳**: 通过关闭 `stopChan` 来通知 `keepAlive` goroutine 停止续约。
2.  **撤销租约**: 调用 `cli.Revoke()` 明确告知 `etcd` 撤销之前授予的租约。`etcd` 在收到撤销请求后，会自动删除与该租约关联的所有键值对。

如果节点异常崩溃，心跳会停止，租约将在 TTL 到期后自动失效，`etcd` 同样会自动删除关联的键值对。

## 服务监控与更新 (`internal/discovery.ServiceWatcher`)

`apiserver` 使用 `ServiceWatcher` 来动态感知 `cachenode` 集群的变化：

1.  **连接 Etcd**: 同样需要创建一个 `etcd` 客户端实例。
2.  **首次同步**: 在开始监视之前，`ServiceWatcher` 会先调用 `cli.Get()` 并指定服务前缀 (`/服务名/`)，获取当前 `etcd` 中所有已注册的节点列表，并将这个初始列表发送给 `apiserver`。
3.  **启动监视 (Watch)**: 调用 `cli.Watch()` 并指定服务前缀 (`clientv3.WithPrefix()`)，创建一个 `Watch` 通道 (`wch`)。`etcd` 会将该前缀下任何键值对的变化事件（`PUT` 或 `DELETE`）推送到这个通道。
4.  **处理事件**: `ServiceWatcher` 的后台 goroutine 会持续监听 `Watch` 通道：
    - 当收到事件时（无论多少个事件，只要有变化），表明节点列表可能已发生变化。
    - 为了获取最新的、一致的节点列表，`ServiceWatcher` 会**重新**调用 `cli.Get()` 来获取当前该服务前缀下的**所有**键值对。
    - 将获取到的最新节点地址列表通过 `updatesChan` 发送给 `apiserver`。
    - 如果 `Watch` 通道关闭或收到错误，会记录日志并通过 `errChan` 发送错误信号。

`apiserver` 接收到 `updatesChan` 发来的新节点列表后，会更新其内部维护的节点信息，并重建一致性哈希环，以确保后续请求能够正确路由。

## 优点

- **自动化**: 节点加入和离开集群无需手动修改配置。
- **高可用**: `apiserver` 能够快速感知节点故障并停止向其路由请求。
- **弹性伸缩**: 可以方便地增加或减少缓存节点数量。
