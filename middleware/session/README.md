# `middleware/session`

这个包只保留 session 相关 middleware，核心模型和校验能力来自 `github.com/fino-io/gokit/session`。

## Transport middleware

- `HTTPMiddleware` 从 `Authorization: Bearer ...`、`X-Session-Token` 或 `session_token` cookie 读取 token。
- `UnaryServerInterceptor` / `StreamServerInterceptor` 从 gRPC metadata 的 `authorization` 或 `x-session-token` 读取 token。
- 读取 token 后调用 `session.Validator`，再用 `session.WithToken` / `session.WithSession` 写回 context。
- 用 `WithBearerHeader`、`WithTokenHeader`、`WithTokenCookie` 可以适配服务自己的接入约定；传空字符串表示禁用该来源。

## Endpoint middleware

- `ValidateMiddleware` 从 `session.TokenFromContext` 读取 token，调用 `session.Validator`，再写回校验后的 session。
- `AuthenticateMiddleware` 在校验 session 后调用 `UserResolver`，并把解析出的业务 user 写入 middleware 自己的 user context。

业务模型、签发、校验、store、token codec、`WithSession` 和 `WithToken` 都在 `github.com/fino-io/gokit/session`。

## 接入示例

```go
// net/http
handler = sessionmw.HTTPMiddleware(manager)(handler)

// gRPC
server := grpc.NewServer(
	grpc.UnaryInterceptor(sessionmw.UnaryServerInterceptor(manager)),
	grpc.StreamInterceptor(sessionmw.StreamServerInterceptor(manager)),
)

// Go kit endpoint
endpoint = sessionmw.AuthenticateMiddleware(manager, resolver)(endpoint)
```

自定义接入字段：

```go
handler = sessionmw.HTTPMiddleware(
	manager,
	sessionmw.WithBearerHeader("X-Auth"),
	sessionmw.WithTokenHeader("X-Auth-Token"),
	sessionmw.WithTokenCookie("sid"),
)(handler)
```
