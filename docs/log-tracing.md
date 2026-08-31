# 日志链路跟踪

服务使用轻量级 W3C Trace Context 做日志关联，不依赖单独的链路采集后端。

## HTTP 入口

请求进入服务时按以下优先级确定链路 ID：

1. 继承格式合法的 `traceparent`；
2. 继承 32 位十六进制 `X-Trace-ID`；
3. 生成新的 32 位 `trace_id`。

每个请求都会生成新的 16 位 `span_id`。响应包含 `X-Trace-ID` 和 `traceparent`，客户端可把 `X-Trace-ID` 提供给技术支持以检索完整请求日志。

```bash
curl -i -H "X-Trace-ID: 4bf92f3577b34da6a3ce929d0e0e4736" \
  http://localhost:8080/docs/openapi.json
```

非法、全零或长度错误的客户端 ID 会被忽略并重新生成，避免把任意文本写入结构化日志。

## 日志与下游传播

- HTTP 请求日志自动包含 `trace_id`、`span_id`。
- 使用请求 `context.Context` 的 GORM SQL 日志自动包含相同字段。
- 项目中的下游 HTTP 请求会发送 W3C `traceparent` 和兼容的 `X-Trace-ID`，并为每次调用生成新的子 `span_id`。
- 后台生成任务和 Asynq 任务在每次执行时创建独立根链路。
- 管理端操作日志表中的 `trace_id` 与应用日志使用同一个值，可直接交叉检索。

新增带 `context.Context` 的业务日志时，应使用上下文日志器：

```go
config.Logger(ctx).Infow("payment completed", "order_no", orderNo)
```

新增下游 HTTP 请求时，应使用带传播能力的构造函数：

```go
req, err := tracing.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
```

启动、关闭和配置加载等不属于某个请求或任务的进程级日志不会包含链路字段。
