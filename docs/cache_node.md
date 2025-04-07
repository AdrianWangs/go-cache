# Cache Node (`cmd/cachenode`)

Cache Node 是分布式缓存系统中实际存储和管理缓存数据的单元。每个 Cache Node 负责集群中的一部分数据，并处理来自 API Server 或其他节点的请求。

## 主要职责

1.  **服务注册**: 启动时向 `etcd` 注册自身地址，并维持心跳以表明其存活状态。
2.  **缓存管理**: 使用 LRU 策略 (`pkg/lru`) 管理本地缓存的添加、获取和淘汰。
3.  **缓存分组**: 支持将缓存数据划分到不同的组 (`internal/cache.Group`)，每个组可以有独立的缓存大小限制和数据源获取逻辑 (`Getter`)。
4.  **处理缓存请求**: 监听指定端口，接收并处理来自 API Server 的缓存读请求 (通过 `internal/server.HTTPPool`)。
5.  **数据加载**: 当本地缓存未命中时，调用与该缓存组关联的 `Getter` 函数从后端数据源加载数据。
6.  **节点列表更新**: 定期通过 HTTP 请求 API Server 的 `/peers` 接口，获取当前集群中所有活跃节点的列表，并更新内部的 `HTTPPool` 配置。_(注意：虽然 Cache Node 获取了所有节点列表，但在当前代码实现中，它似乎并不主动将请求转发给其他节点，而是依赖 API Server 进行路由。`HTTPPool` 中的 `PeerPicker` 逻辑主要由 API Server 使用，Cache Node 本身在处理请求时倾向于直接本地加载。)_
7.  **并发控制**: 使用 `singleflight` 机制防止缓存击穿，确保对于同一个 key 的并发加载请求只执行一次实际的数据源查询。

## 核心组件

- **`main` 函数 (`cmd/cachenode/main.go`)**: 程序的入口点。
  - 解析命令行参数 (etcd 地址, 服务名, 节点地址, 缓存大小等)。
  - 创建 `ServiceDiscovery` 实例并调用 `Register()` 向 `etcd` 注册服务。
  - 定义 `db` (示例数据源) 和 `GetterFunc` (本地数据获取逻辑)。
  - 创建 `cache.Group` 实例。
  - 创建 `server.HTTPPool` 实例，用于处理节点间通信，**明确指定使用 Protobuf 协议**。
  - 调用 `group.RegisterPeers(pool)` (尽管在本节点内部，`PickPeer` 可能不常用)。
  - 启动 HTTP 服务 (`http.ListenAndServe`) 监听节点通信端口，使用 `HTTPPool` 作为处理器。
  - 启动一个后台 goroutine (`updatePeers`)，定期向 API Server 请求 `/peers` 接口，获取节点列表，并调用 `pool.Set()` 更新 `HTTPPool` 中的节点信息。
  - 监听系统信号以实现优雅关闭 (调用 `sd.Unregister()` 注销服务)。
- **`Cache` (`internal/cache/cache.go`)**: 线程安全的 LRU 缓存实现，封装了 `pkg/lru`。
- **`Group` (`internal/cache/group.go`)**: 缓存命名空间。
  - 管理一个 `Cache` 实例 (`mainCache`)。
  - 包含一个 `Getter` 接口，用于缓存未命中时加载数据。
  - 包含一个 `singleflight.Group` (`loader`) 防止缓存击穿。
  - `Get` 方法是核心逻辑：先查本地缓存，未命中则调用 `load`。
  - `load` 方法使用 `singleflight.Do` 执行加载逻辑。
  - `getLocally`: 实际调用 `Getter` 从数据源获取数据，并将结果存入 `mainCache`。
  - `getFromPeerWithProto`: (理论上) 用于从其他节点获取数据，但在当前架构下主要由 API Server 调用其 `NodeGetter` 实现。
- **`HTTPPool` (`internal/server/http.go`)**: 作为 HTTP 服务端，处理来自 API Server 的请求。
  - 实现了 `http.Handler` 接口 (`ServeHTTP`)。
  - 根据配置的协议 (`protocol`) 调用 `handleHTTP` 或 `handleProtobuf`。
  - `handleProtobuf`: 处理 Protobuf 请求，反序列化 `pb.Request`，调用 `cache.GetGroup(req.Group).Get(req.Key)` 获取数据，序列化 `pb.Response` 并返回。
  - `Set`: 由 `updatePeers` 调用，用于更新其内部维护的一致性哈希环 (`peers`) 和到其他节点的客户端 (`httpGetters`)。_(主要供 `PeerPicker` 接口使用，在本节点作为服务端时，主要关注 `ServeHTTP` 逻辑)_
  - 实现了 `peers.PeerPicker` 接口 (`PickPeer`)，但如前所述，Cache Node 自身在处理请求时似乎不常用此方法。
- **`HTTPGetter` (`internal/server/http_getter.go`)**: 作为 HTTP 客户端，实现了 `peers.PeerGetter` 接口，用于从其他节点获取数据 (理论上)。与 `api/handlers/client_handlers.go` 中的 `HTTPGetter` 类似，但位于 `internal/server` 包下。

## 启动流程

见 `docs/architecture.md` 中的启动流程描述。

## 请求处理流程 (处理来自 API Server 的 Protobuf 请求)

1.  `HTTPPool` 的 `ServeHTTP` 方法接收到 HTTP POST 请求。
2.  由于协议配置为 `protobuf`，调用 `handleProtobuf` 方法。
3.  `handleProtobuf` 读取请求 Body，并使用 `proto.Unmarshal` 将其反序列化为 `pb.Request` 结构体。
4.  根据 `req.Group` 获取对应的 `cache.Group` 实例。
5.  调用 `group.Get(req.Key)` 方法获取数据：
    - 检查本地 `mainCache`。
    - **命中**: 返回 `ByteView`。
    - **未命中**: 调用 `load` -> `singleflight.Do` -> `getLocally` -> `getter.Get` 从数据源加载，加载成功后存入 `mainCache` 并返回 `ByteView`。
6.  如果 `group.Get` 返回错误，根据错误类型设置 HTTP 响应状态码 (404, 400, 500) 并返回错误信息。
7.  如果成功获取 `ByteView`，创建一个 `pb.Response` 结构体，将 `ByteView` 的数据存入 `resp.Value`。
8.  使用 `proto.Marshal` 将 `pb.Response` 序列化。
9.  设置 HTTP 响应头 `Content-Type` 为 `application/protobuf`。
10. 将序列化后的 Protobuf 数据写入 HTTP 响应体，状态码为 200 OK。
