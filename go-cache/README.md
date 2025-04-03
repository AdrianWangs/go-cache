# Go-Cache

一个简单的分布式缓存系统，使用 Go 语言实现。

## 功能特性

- 单机缓存和基于 HTTP 的分布式缓存
- 最近最少使用(LRU)缓存策略
- 使用一致性哈希选择节点
- 使用互斥锁防止缓存击穿
- 使用 logrus 进行统一的日志处理

## 项目结构

```
.
├── cmd                    // 命令行应用
│   └── gocache           // 主程序入口
├── internal               // 内部包
│   ├── cache             // 缓存相关实现
│   ├── consistenthash    // 一致性哈希算法
│   ├── interfaces        // 接口定义
│   ├── peers             // 节点通信接口
│   └── server            // HTTP服务实现
└── pkg                    // 公共包
    └── logger            // 日志工具
```

## 如何使用

1. 启动缓存服务器:

```bash
# 启动第一个缓存服务器
go run cmd/gocache/main.go -port=8001

# 在新的终端启动第二个缓存服务器
go run cmd/gocache/main.go -port=8002

# 在新的终端启动第三个缓存服务器
go run cmd/gocache/main.go -port=8003
```

2. 启动 API 服务器:

```bash
go run cmd/gocache/main.go -port=8001 -api=true
```

3. 测试缓存:

```bash
curl "http://localhost:9999/api?key=Tom"
```

## 贡献

欢迎提出问题和贡献代码。
