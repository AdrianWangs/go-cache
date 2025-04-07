# 通信协议 (Protobuf over HTTP)

为了提高分布式缓存系统内部通信的效率和性能，本项目在 API Server 与 Cache Node 之间，以及 Cache Node 理论上的相互通信（尽管当前架构下节点间直接通信较少）采用了基于 Protobuf (Protocol Buffers) 的通信协议，并通过 HTTP POST 请求进行传输。

## 为什么选择 Protobuf？

- **高效性**: Protobuf 是一种二进制序列化格式，相比于基于文本的格式（如 JSON 或 XML），序列化和反序列化的速度更快，传输的数据量更小。
- **强类型与结构化**: `.proto` 文件定义了清晰的数据结构 (`message`)，有助于保证通信双方数据格式的一致性，减少解析错误。
- **语言无关**: Protobuf 支持多种编程语言，便于未来可能的异构系统集成。
- **向后兼容性**: Protobuf 的设计考虑了协议的演进，可以在不破坏现有服务的情况下添加新字段。

## 实现方式

1.  **定义 Proto 文件 (`proto/cache_server/cache.proto`)**: 定义了通信所需的数据结构，主要是 `Request` 和 `Response` 消息类型。

    ```protobuf
    syntax = "proto3";
    package cache_server;
    option go_package = "./;cache_server";

    // 请求消息
    message Request {
      string group = 1; // 缓存组名称
      string key = 2;   // 缓存键
    }

    // 响应消息
    message Response {
      bytes value = 1; // 缓存值
    }
    ```

2.  **代码生成**: 使用 `protoc` 工具和 `protoc-gen-go` 插件根据 `.proto` 文件生成 Go 语言代码 (`proto/cache_server/cache.pb.go`)。这会生成对应的 Go 结构体以及序列化/反序列化方法。
3.  **客户端 (API Server 的 `NodeGetter`)**: 在 `api/handlers/client_handlers.go` 中的 `HTTPGetter` 或 `ProtoGetter`：
    - 当需要向 Cache Node 发送请求时，创建一个 `pb.Request` 结构体实例并填充 `group` 和 `key`。
    - 调用 `proto.Marshal()` 将该结构体序列化为二进制字节流。
    - 创建一个 HTTP POST 请求，将序列化后的字节流作为请求体 (request body)。
    - 设置请求头 `Content-Type` 为 `application/protobuf`。
    - 将请求发送到目标 Cache Node 的 `HTTPPool` 监听地址 (`http://{nodeAddr}{basePath}`)。
    - 接收到响应后，读取响应体中的二进制数据。
    - 调用 `proto.Unmarshal()` 将响应数据反序列化为 `pb.Response` 结构体。
4.  **服务端 (Cache Node 的 `HTTPPool`)**: 在 `internal/server/http.go` 中的 `handleProtobuf` 方法：
    - 接收到 HTTP POST 请求。
    - 检查 `Content-Type` 是否为 `application/protobuf` (虽然当前代码似乎未显式检查，但约定如此)。
    - 读取请求体中的二进制数据。
    - 调用 `proto.Unmarshal()` 将请求数据反序列化为 `pb.Request` 结构体。
    - 根据 `pb.Request` 中的信息处理缓存逻辑。
    - 处理完成后，创建一个 `pb.Response` 结构体实例并填充 `value`。
    - 调用 `proto.Marshal()` 将 `pb.Response` 序列化为二进制字节流。
    - 设置响应头 `Content-Type` 为 `application/protobuf`。
    - 将序列化后的字节流写入响应体。

## 协议配置

- `internal/server.HTTPPool` 结构体中有一个 `protocol` 字段 (类型为 `server.Protocol`)，用于指定通信协议。
- `NewHTTPPool` 函数默认将协议设置为 `ProtocolProtobuf`。
- 可以通过 `server.WithProtocol()` 选项在创建 `HTTPPool` 时指定协议。
- `ServeHTTP` 方法会根据 `protocol` 字段的值选择调用 `handleHTTP` (传统 HTTP GET) 还是 `handleProtobuf` (Protobuf over HTTP POST)。

通过这种方式，系统内部的关键通信路径利用了 Protobuf 的高效性，有助于降低延迟和网络负载。
