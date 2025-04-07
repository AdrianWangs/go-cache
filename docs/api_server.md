# API Server (`cmd/apiserver`)

API Server 是整个分布式缓存系统的用户入口和集群协调者。它不实际存储缓存数据，而是负责接收客户端请求、通过服务发现了解缓存节点状态、利用一致性哈希将请求路由到正确的缓存节点，并将结果返回给客户端。

## 主要职责

1.  **接收用户请求**: 监听指定端口 (默认为 8080)，处理来自客户端的 HTTP 请求。
2.  **服务发现集成**: 连接 `etcd` 并使用 `ServiceWatcher` 实时监控 `cachenode` 的注册信息。当节点加入或离开时，动态更新内部的节点列表。
3.  **一致性哈希路由**: 维护一个一致性哈希环 (`consistenthash.Map`)。当收到缓存请求时，根据请求的 `key` 计算哈希值，并在环上找到对应的 `cachenode` 地址。
4.  **请求转发**: 将用户的缓存请求（使用 Protobuf 格式）转发给通过一致性哈希选中的目标 `cachenode`。
5.  **节点信息服务**: 提供 `/peers` HTTP 接口，供 `cachenode` 查询当前所有活跃节点的地址列表。
6.  **监控与健康检查**: 提供 `/api/metrics` (示例) 和 `/health` 接口。

## 核心组件 (`api` 包)

- **`ApiServer` (`api/api.go`)**: API Server 的主结构体，包含了配置、服务发现实例 (`ServiceWatcher`)、HTTP 服务器、路由器 (`pkg/router.Router`) 以及各种请求处理器。
- **`ApiServerConfig` (`api/api.go`)**: API Server 的配置结构。
- **`CacheHandler` (`api/handlers/cache_handlers.go`)**: 处理缓存相关的 API 请求 (如 `/api/cache`)。
  - 内部维护一致性哈希环 (`ring`) 和节点地址到 `NodeGetter` 的映射 (`nodeGetters`)。
  - `UpdatePeers`: 当 `ServiceWatcher` 检测到节点变化时被调用，用于重建哈希环和更新 `nodeGetters`。
  - `pickNode`: 根据 `key` 在哈希环上选择目标节点。
  - `GetCacheHandler`: 处理具体的 GET 请求，执行选择节点、转发请求的操作。
- **`NodeHandler` (`api/handlers/node_handlers.go`)**: 处理节点相关的 API 请求。
  - `GetNodesHandler`: 实现 `/peers` 和 `/api/nodes` 接口，返回当前已知的活跃节点列表。
  - `UpdateNodeAddresses`: 由 `ServiceWatcher` 回调，更新内部节点列表，并触发 `CacheHandler` 的 `UpdatePeers`。
  - `HealthCheckHandler`: 实现 `/health` 接口。
- **`MetricsHandler` (`api/handlers/metrics_handlers.go`)**: (示例) 处理监控指标相关的请求。
- **`HTTPGetter`/`ProtoGetter` (`api/handlers/client_handlers.go`)**: 实现了 `NodeGetter` 接口，负责与 `cachenode` 进行通信。它将 API Server 的请求封装成 Protobuf 格式，通过 HTTP POST 发送给目标 `cachenode`，并处理响应。
- **路由注册 (`api/routes/routes.go`)**: 定义了 API Server 提供的所有 HTTP 路由及其对应的处理器。

## 启动流程

1.  解析命令行参数或配置文件，获取 `etcd` 地址、服务名、监听端口等信息。
2.  创建 `ApiServerConfig`。
3.  调用 `NewApiServer` 创建 `ApiServer` 实例：
    - 创建 `ServiceWatcher` 连接 `etcd` 并指定要监视的服务名。
    - 创建 `CacheHandler`, `NodeHandler`, `MetricsHandler` 等处理器。
    - 设置 `NodeHandler` 的回调函数，使其在接收到 `ServiceWatcher` 的节点更新时，能够触发 `CacheHandler` 的 `UpdatePeers` 方法。
    - 创建 `pkg/router.Router` 实例并添加中间件（如日志、恢复、指标）。
    - 创建 `http.Server` 实例。
4.  调用 `apiServer.Start()` 方法：
    - 注册所有 API 路由 (`routes.RegisterRoutes`)。
    - 启动一个后台 goroutine 运行 `ServiceWatcher` 的 `Watch` 方法：
      - `Watch` 方法首先进行一次初始节点同步。
      - 然后开始监听 `etcd` 的变化事件。
      - 当收到节点更新列表 (`updatesChan`) 时，调用 `NodeHandler.UpdateNodeAddresses`，进而触发 `CacheHandler.UpdatePeers` 来更新哈希环。
    - 启动 HTTP 服务器 (`httpServer.ListenAndServe()`)，开始监听并处理请求。

## 请求处理流程 (以 GET `/api/cache?group={group}&key={key}` 为例)

1.  HTTP 请求到达，`pkg/router` 根据路径匹配到 `CacheHandler.GetCacheHandler`。
2.  `GetCacheHandler` 解析出 `groupName` 和 `key`。
3.  调用 `pickNode(key)` 方法：
    - 在一致性哈希环 (`ring`) 上根据 `key` 找到对应的节点地址 `nodeAddr`。
    - 从 `nodeGetters` 映射中获取对应的 `NodeGetter` 实例（通常是 `HTTPGetter` 或 `ProtoGetter`）。
4.  如果找不到合适的节点或 `NodeGetter`，返回错误。
5.  创建 Protobuf 请求 (`pb.Request`)。
6.  调用 `nodeGetter.GetByProto(req, resp)`：
    - `HTTPGetter`（或 `ProtoGetter`）将 `pb.Request` 序列化。
    - 构造 HTTP POST 请求，目标 URL 为 `http://{nodeAddr}{basePath}`，Body 为序列化后的 Protobuf 数据，`Content-Type` 为 `application/protobuf`。
    - 发送 HTTP 请求到目标 `cachenode`。
    - 接收目标 `cachenode` 的 HTTP 响应。
    - 检查响应状态码，处理可能的错误（如 404 Not Found）。
    - 读取响应 Body 中的 Protobuf 数据，并反序列化到 `pb.Response`。
7.  如果 `GetByProto` 返回错误，`GetCacheHandler` 根据错误类型（或错误消息内容）向客户端返回相应的 HTTP 错误（404, 400, 500 等）。
8.  如果成功，`GetCacheHandler` 将 `pb.Response.Value` 作为响应体写入 HTTP 响应，返回给客户端。
