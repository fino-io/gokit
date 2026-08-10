# gokit

`gokit` 是一套 Go 微服务基础组件库，提供服务运行、HTTP/gRPC 通信、服务发现、可观测性和常用中间件，方便业务服务直接复用。

## 包含什么

- **服务运行时**：进程生命周期、配置和服务启动辅助（`server`）。
- **传输层**：HTTP 响应封装、错误处理和客户端；gRPC 客户端、服务发现解析器及错误映射（`transport/http`、`transport/grpc`）。
- **服务发现与负载均衡**：直连、etcd，以及随机、轮询和重试策略（`sd`、`sd/lb`）。
- **可观测性**：Prometheus 指标、OpenTelemetry tracing、日志和访问日志（`metrics`、`tracing`、`kit`、`middleware/accesslog`）。
- **通用中间件**：参数校验、日志、限流、国际化、上下文缓存和 session 认证（`middleware`）。
- **通用工具**：带 HMAC 签名的服务端 session、分页游标、重试/退避、网络和日志工具（`session`、`pagination`、`retry`、`backoff`、`util`）。


按需引入子包即可：

```go
import "github.com/fino-io/gokit/transport/http"
```

具体用法见各目录下的 README，例如 [session](session/README.md)、[tracing](tracing/README.md) 和 [session middleware](middleware/session/README.md)。

## 开发

项目使用 Go Modules，要求 Go `1.25.12` 或兼容版本。

```bash
go test ./...
```
