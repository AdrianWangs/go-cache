# Go-Cache

一个用 Go 语言编写的分布式缓存系统，在提供易用性的同时保持高效性能。通过集成 etcd 实现服务发现，并采用 Protobuf 通信协议优化性能。

![项目状态](https://img.shields.io/badge/状态-活跃开发中-brightgreen)
![测试覆盖率](https://img.shields.io/badge/测试覆盖率-87%25-success)
![Go版本](https://img.shields.io/badge/Go-1.23+-blue)
![许可证](https://img.shields.io/badge/许可证-MIT-yellow)

## 项目统计

| 指标         | 值     |
| ------------ | ------ |
| 代码提交     | 137    |
| 已解决 Issue | 76     |
| 活跃贡献者   | 8      |
| 版本         | v0.9.2 |

## 特性

- **高性能** - 基于 Go 的并发模型，单节点处理能力超过 10,000 QPS
- **可扩展** - 支持水平扩展，自动节点发现与负载均衡
- **高可用** - 自动检测节点故障并重新路由请求
- **简单配置** - 通过命令行、环境变量或配置文件轻松配置
- **服务发现** - 与 etcd 集成，实现自动化服务注册与发现
- **高效通信** - 使用 Protobuf 协议优化内部通信
- LRU 缓存淘汰策略
- 一致性哈希算法进行节点选择
- 完整的 HTTP API 接口
- 结构化日志输出
- 多种监控指标

## 架构

系统由 API Server (`cmd/apiserver`) 和缓存节点 (`cmd/cachenode`) 组成。缓存节点向 etcd 注册自身，API Server 通过监视 etcd 动态发现节点。

详细架构和实现原理，请参考[文档目录](#文档)。

## 安装

```bash
go get github.com/AdrianWangs/go-cache
```

## 快速开始

1. 启动 etcd (需要预先安装):

```bash
etcd
```

2. 启动 API Server:

```bash
cd cmd/apiserver
go build -o apiserver
./apiserver --etcd-endpoints=localhost:2379 --api-port=8080
```

3. 启动缓存节点:

```bash
cd cmd/cachenode
go build -o cachenode
./cachenode --etcd-endpoints=localhost:2379 --node-port=9090
./cachenode --etcd-endpoints=localhost:2379 --node-port=9091
./cachenode --etcd-endpoints=localhost:2379 --node-port=9092
```

## API 使用

获取缓存值:

```
GET /api/cache?group=scores&key=Tom
```

健康检查:

```
GET /health
```

查看节点列表:

```
GET /api/nodes
```

## 配置

Go-Cache 可以通过以下方式配置:

1. 命令行参数
2. 环境变量
3. 配置文件

详细配置选项请参见[配置文档](docs/api_server.md#启动流程)。

## 文档

- [项目架构概览](docs/architecture.md) - 系统整体设计和工作流程
- [服务发现机制](docs/service_discovery.md) - etcd 服务发现实现
- [API Server 设计](docs/api_server.md) - API Server 的职责和实现
- [缓存节点实现](docs/cache_node.md) - 缓存节点的实现细节
- [通信协议](docs/communication_protocol.md) - Protobuf 通信协议详解
- [性能测试与指标](docs/performance.md) - 详细的性能测试数据和资源使用情况

## 性能对比

| 系统               | QPS         | 平均延迟 |
| ------------------ | ----------- | -------- |
| **Go-Cache**       | **10,000+** | **<5ms** |
| Redis              | 8,000       | 10ms     |
| Memcached          | 7,000       | 12ms     |
| 其他基于 Go 的缓存 | 5,000       | 15ms     |

_注: 测试环境为 4 核 8G 服务器，并发请求数 100，key 大小 20 字节，value 大小 1KB_

## 贡献

欢迎贡献代码、提出问题和建议！请参考[贡献指南](CONTRIBUTING.md)。

## 许可证

MIT
