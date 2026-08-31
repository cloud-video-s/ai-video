# Adjust 用户事件上报

## 事件映射

| 用户操作 | Adjust event token |
| --- | --- |
| 支付完成 | `fdhs2r` |
| 创建积分商品订单 | `6u9gko` |
| 当前设备第一次注册 | `b159ty` |
| 首次绑定三方账号 | `qd5uvh` |
| 订阅商品支付完成 | `9yot8l` |

业务操作提交成功后，服务会生成包含用户 ID、事件 token、操作发生时的渠道 ID 及规则所需快照的事件。操作发生时未绑定渠道（`channel_id=0`）的事件不进入主队列，直接写入 `video_adjust_pending_event`。已有渠道的事件通过 Asynq 写入 Redis，由消费者查询 `video_user_attribution`：未归因或归因记录的 `channel_id` 仍为 `0` 时继续保存在待归因表；只有归因记录写入非零 `channel_id` 后才重新投递，并按对应渠道规则发送 Adjust S2S 事件。

## 服务配置

```yaml
adjust:
  # 入站：是否接收 Adjust 归因 callback。
  enabled: true
  # 出站：是否注册本功能的 Redis 队列消费和 Adjust S2S 上报。
  event_enabled: true
  # 本服务自行定义的 callback 共享鉴权值，不是 Adjust S2S Security token。
  callback_token: "replace-with-a-random-secret"
  max_body_bytes: 65536
  tracker_channels: {}
  # 仅在 Adjust 控制台开启 S2S Security 后配置，由控制台生成。
  event_auth_token: ""
  event_base_url: https://s2s.adjust.com
  event_environment: production # production | sandbox
  # 归因记录没有 app_token 时的兜底；key 为内部 app_code。
  event_app_tokens: {}
```

`callback_token` 是本服务在 callback URL/Header 中使用的应用自定义共享值，Adjust 官方没有规定最小长度。`event_auth_token` 则是 Adjust 控制台生成的 S2S Security token，上报时通过 `Authorization: Bearer ...` 发送，两者不能混用。

事件队列复用 `internal/pkg/task` 的 Asynq Redis 实现：

- 任务类型为 `adjust:event`，投递到 `default` 队列；`task.queues` 必须包含 `default`。
- 订阅延迟通过 Redis 定时任务实现，无需额外插件或 dead-letter 队列。
- 每个事件最多执行 6 次（首次执行加 5 次重试）；耗尽重试后由 Asynq 归档，可通过 Asynq 管理工具检查或重新入队。
- 稳定的事件 ID 同时作为 Redis 任务 ID；重复投递视为成功，成功任务保留 24 小时以降低重复上报风险。

队列采用至少一次投递语义。Redis 短暂断连由统一 Worker 自动重连，进程异常退出时进行中的任务会被重新投递。

## 渠道规则

只有状态启用且 `ad_platform` 等于 `Adjust` 的渠道会发送事件。`video_channel.callback_config` 使用以下结构：

```json
{
  "rules": [
    {
      "trigger_event": "payment",
      "callback_events": ["payment"],
      "order_count_threshold": 0,
      "payment_minimum_amount": 1,
      "subscription_delay_minutes": 0,
      "amount_deduction_enabled": false,
      "amount_deduction_percent": 0
    }
  ]
}
```

支持的事件名为 `activation`、`login`、`order_created`、`payment`、`subscription`。规则含义：

- `trigger_event`：实际发生的业务操作。
- `callback_events`：该操作需要映射上报的一个或多个 Adjust 事件。
- `order_count_threshold`：上报 `order_created` 时，用户操作快照中的订单次数必须与该值相等。
- `payment_minimum_amount`：上报 `payment` 的最低实付金额；无实付金额时使用应付金额。
- `subscription_delay_minutes`：上报 `subscription` 前的延迟分钟数。
- `amount_deduction_percent`：启用后，支付/订阅收入按该百分比扣减后上报。

S2S 请求的 app token 优先使用归因设备记录中的 `app_token`，缺失时依次按归因记录的 `app_code` 从 `event_app_tokens` 查找。两处都没有配置会进入重试，最终转入失败队列。

## 部署前检查

1. 审核并单独执行 `scripts/schema/20260820_003_create_adjust_pending_event.sql`。代码不会自动建表；仓库中创建脚本不代表已经执行，也不授权执行。
2. 上线切换前先处理旧 RabbitMQ 队列中的存量消息；新版本不再连接 RabbitMQ，也不会自动迁移旧队列消息。
3. 确认 Redis 连接信息正确，并确保 `task.queues` 包含 `default`。
4. 确认渠道已启用、`ad_platform=Adjust` 且已保存对应 callback 规则。
5. 确认归因 callback 能保存 `app_token`，或为每个内部 `app_code` 配置 `event_app_tokens`。
6. 若 Adjust 控制台启用了 S2S Security，配置具有 Events scope 的 `event_auth_token`；未启用时保持为空。

Adjust 官方参考：

- [Server-to-server events](https://dev.adjust.com/en/api/s2s-api/events/)
- [Server-to-server security](https://dev.adjust.com/en/api/s2s-api/security/)
- [Set up server callbacks](https://help.adjust.com/en/article/server-callbacks)
