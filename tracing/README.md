# Tracing 使用说明

`tracing` 为基于 Go kit 的服务提供 OpenTelemetry 链路追踪辅助能力。它的默认策略是“追踪失败不影响业务链路”：未启用 tracing 或初始化失败时，会返回 noop tracer 和可安全调用的 shutdown 函数。

## 配置

`New` 会通过 `fino/config` 读取 `tracing` 配置段：

```yaml
tracing:
  enable: false
  endpoint: localhost:4318
  sampleRatio: 1
  environment: dev
  serviceVersion: ''
  serviceInstanceID: ''
```

字段说明：

- `enable`：为 `true` 时启用 OTLP 导出；为 `false` 时使用 noop tracer。
- `endpoint`：OTLP HTTP endpoint，可写成 `host:port` 或完整 URL。空值会回退到 `localhost:4318`。
- `sampleRatio`：root trace 采样比例。`1` 表示全采样，`0.01` 表示约 1% 采样，`<= 0` 表示主动丢弃新 root trace。旧配置文件缺失该字段或值为 `0` 时，默认按 `1` 处理，保持兼容。
- `environment`：部署环境，会导出为 `deployment.environment`。
- `serviceVersion`：服务版本，会导出为 `service.version`。
- `serviceInstanceID`：实例 ID、Pod 名或主机 ID，会导出为 `service.instance.id`。

## 初始化

普通服务优先使用 `New`：

```go
tracer, shutdown, err := tracing.New("example.user.UserService")
if err != nil {
	logs.Warnw("failed to set tracer", "error", err)
}
defer func() {
	_ = shutdown(context.Background())
}()
```

测试或嵌入式场景需要显式传配置时，使用 `NewWith`：

```go
tracer, shutdown, err := tracing.NewWith(ctx, "example.user.UserService", &tracing.Config{
	Enable:            true,
	Endpoint:          "otel-collector:4318",
	SampleRatio:       0.05,
	Environment:       "prod",
	ServiceVersion:    "v1.2.3",
	ServiceInstanceID: "user-server-7d8f",
})
```

`NewTracer` 和 `NewTraceProvider` 是更底层的辅助函数，适合只需要指定 OTLP endpoint 的调用方。应用代码通常优先使用 `New` 或 `NewWith`。

## 传输层服务端追踪

HTTP 和 gRPC 服务应在 access log、路由和 endpoint 之前创建 transport-level server span：

```go
handler = tracing.HTTPServerMiddleware(tracer)(handler)

grpc.NewServer(grpc.ChainUnaryInterceptor(
	tracing.GRPCUnaryServerTracingInterceptor(tracer),
	accesslog.UnaryServerInterceptor(accesslog.LoadConfig()),
))
```

两个 middleware 都会先提取上游 trace context，再创建本地 server span，因此 access log 可以直接从请求 context 记录 `trace_id` 和 `span_id`。`GRPCUnaryServerInterceptor()` 保留为仅提取 metadata 的兼容 API；使用 server span 时不要同时串联这两个 gRPC interceptor。

transport 层应只创建一个 server span。endpoint 层如果还需要记录业务操作，应创建 child span，并使用 internal span kind，避免同一个请求出现两个 server span。

## 搭建 otel-collector:4318

`otel-collector:4318` 是服务访问 OpenTelemetry Collector 的 OTLP HTTP 接收地址。Collector 配置由 `receivers`、`processors`、`exporters` 和 `service.pipelines` 组成；只在 `receivers` 里声明组件不会生效，必须把它放进对应 pipeline。

最小可用配置如下，适合本地验证。它接收 OTLP HTTP `4318` 和 OTLP gRPC `4317`，经过 `memory_limiter`、`batch` 后用 `debug` exporter 打印 trace：

```yaml
# otel-collector.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 256
    spike_limit_mib: 64
  batch:
    timeout: 1s
    send_batch_size: 512

exporters:
  debug:
    verbosity: basic

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [debug]
```

本地 Docker Compose：

```yaml
services:
  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    command: ["--config=/etc/otelcol/config.yaml"]
    volumes:
      - ./otel-collector.yaml:/etc/otelcol/config.yaml:ro
    ports:
      - "4318:4318" # OTLP HTTP，gokit/tracing 默认使用这个端口
      - "4317:4317" # OTLP gRPC，给其他 SDK 或工具使用
```

服务侧配置：

```yaml
tracing:
  enable: true
  endpoint: otel-collector:4318
  sampleRatio: 1
  environment: dev
  serviceVersion: v0.1.0
  serviceInstanceID: local-1
```

如果应用和 Collector 不在同一个 Docker network，应用不能解析 `otel-collector`。这时本机进程使用 `localhost:4318`；容器内应用则把应用容器和 `otel-collector` 放到同一个 compose network。

## Sidecar 使用方式

Sidecar 模式是“每个业务 Pod 内同时运行业务容器和 Collector 容器”。业务容器把 trace 发给同 Pod 内的 Collector，所以 endpoint 写 `localhost:4318`。Collector 再把 trace 转发到集群内的 gateway Collector 或 tracing 后端。

示例 ConfigMap：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: user-service-otel-sidecar
data:
  otel-collector.yaml: |
    receivers:
      otlp:
        protocols:
          http:
            endpoint: 127.0.0.1:4318

    processors:
      memory_limiter:
        check_interval: 1s
        limit_mib: 128
        spike_limit_mib: 32
      batch:
        timeout: 1s
        send_batch_size: 256

    exporters:
      otlphttp/gateway:
        endpoint: http://otel-gateway.observability.svc.cluster.local:4318

    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [memory_limiter, batch]
          exporters: [otlphttp/gateway]
```

示例 Deployment 片段：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
spec:
  template:
    spec:
      containers:
        - name: app
          image: example/user-service:v1.2.3
          env:
            - name: SERVICE_VERSION
              value: v1.2.3
          ports:
            - containerPort: 30101

        - name: otel-collector
          image: otel/opentelemetry-collector-contrib:latest
          args:
            - --config=/etc/otelcol/config.yaml
          ports:
            - name: otlp-http
              containerPort: 4318
          volumeMounts:
            - name: otel-config
              mountPath: /etc/otelcol/config.yaml
              subPath: otel-collector.yaml
              readOnly: true

      volumes:
        - name: otel-config
          configMap:
            name: user-service-otel-sidecar
```

业务服务配置：

```yaml
tracing:
  enable: true
  endpoint: localhost:4318
  sampleRatio: 0.05
  environment: prod
  serviceVersion: v1.2.3
  serviceInstanceID: ${HOSTNAME}
```

Sidecar 的优点是业务进程只依赖本地 Collector，网络抖动和后端限流不会直接打到业务进程；不同服务也可以有不同采样、脱敏和路由策略。代价是每个 Pod 多一个容器，资源成本和配置发布复杂度更高。

生产环境更常见的折中方式是 `app -> local sidecar/daemonset -> gateway collector -> tracing backend`。sidecar/daemonset 负责本地接收、batch、限流和短暂缓冲；gateway 负责统一鉴权、导出、跨集群路由和后端协议适配。

## 上下文传播

HTTP 入站请求应在 endpoint 执行前提取 W3C TraceContext 和 Baggage：

```go
serverOptions := []httptransport.ServerOption{
	httptransport.ServerBefore(tracing.HTTPToContext),
}
```

HTTP 出站请求可使用 `InjectHTTPHeader` 注入传播头：

```go
tracing.InjectHTTPHeader(ctx, req.Header)
```

gRPC 入站请求应在 endpoint 执行前从 metadata 提取 trace 上下文：

```go
serverOptions := []grpctransport.ServerOption{
	grpctransport.ServerBefore(tracing.GRPCToContext),
}
```

当前 propagator 使用 W3C TraceContext 和 Baggage。如果运行环境仍依赖 B3 或 Jaeger header，需要先补充兼容 propagator，再期望跨技术栈 trace 连续。

## Endpoint Span

服务端和客户端 endpoint 可分别使用 `TraceServer` 和 `TraceClient` 包装：

```go
endpoint := tracing.TraceServer(tracer, "GetUser")(endpoint)
clientEndpoint := tracing.TraceClient(tracer, "GetUser")(clientEndpoint)
```

`TraceServer` 创建 server span，`TraceClient` 创建 client span。两者都会记录返回的 error 和 `endpoint.Failer` 暴露的业务错误。若业务错误只希望作为属性记录、不希望把 span 标记为失败，可使用 `WithIgnoreBusinessError`。

operation name 应保持低基数，例如 RPC 方法名。不要把用户 ID、资源 ID、原始 path、query string 或请求体字段放进 span name 或高频属性。

## 生产建议

- span 优先发送到本地 sidecar、daemonset 或集群内 OpenTelemetry Collector，不建议业务服务直接连接远端 tracing backend。
- 高流量服务不要长期全采样。可从 `sampleRatio: 0.01` 或 `0.05` 开始，事故排查或灰度验证时再临时调高。
- `environment`、`serviceVersion`、`serviceInstanceID` 建议由部署系统注入，方便按环境、版本、实例过滤 trace。
- tracing 应按 best-effort 处理。导出失败、collector 不可用或 shutdown flush 失败应记录日志并监控，但不能导致业务请求失败。
- 不要在 span 属性中写入敏感信息或高基数字段。优先记录稳定的方法名、状态分类、重试次数和依赖名称。
