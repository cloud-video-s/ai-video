# Redis 消息队列

项目通过 `internal/pkg/task` 封装 Asynq，使用 Redis 持久化消息。封装提供即时、延迟、定时、去重和指定优先级队列投递，以及原始字节和强类型 JSON 两种消费方式。

## 定义与注册消费者

任务类型应保持稳定且全局唯一。JSON 消费者可以只处理业务结构，不需要自行反序列化：

```go
package notification

import (
    "context"
    "fmt"

    "ai-video/internal/pkg/task"
)

const TypeWelcome = "notification:welcome"

type WelcomeMessage struct {
    UserID uint64 `json:"user_id"`
}

func Register(manager *task.Manager) {
    task.HandleJSON(manager.Worker, TypeWelcome, func(ctx context.Context, message WelcomeMessage) error {
        if err := sendWelcome(ctx, message.UserID); err != nil {
            return fmt.Errorf("send welcome notification: %w", err)
        }
        return nil
    })
}
```

所有消费者必须在 `manager.Start()` 之前注册。处理成功返回 `nil`；返回错误或发生 panic 时，消息会进入 Asynq 重试流程。未显式设置 `asynq.MaxRetry` 时使用 Asynq 默认重试次数，耗尽后消息归档，不会静默丢弃。

周期任务只需要提供稳定的任务类型、执行方法和间隔。Manager 会统一完成 Asynq Scheduler 初始化、调度表达式转换、默认队列和跨实例投递去重：

```go
err := manager.RegisterPeriodic(task.PeriodicTasks{
    TypeRefreshCache: {
        Every: 10 * time.Minute,
        Run:   refreshCache,
    },
})
```

## 投递消息

优先使用带 `context.Context` 的方法，让请求取消或超时能终止本次 Redis 操作：

```go
import (
    "time"

    "github.com/hibiken/asynq"
)

_, err := manager.Client.EnqueueToQueueContext(
    ctx,
    TypeWelcome,
    WelcomeMessage{UserID: userID},
    "default",
    asynq.MaxRetry(5),
    asynq.Timeout(30*time.Second),
)
```

可用投递方法：

- `EnqueueContext`：立即投递到默认队列。
- `EnqueueDelayContext`：延迟指定时长后执行。
- `EnqueueAtContext`：在指定时间执行。
- `EnqueueUniqueContext`：在 TTL 窗口内去重。
- `EnqueueToQueueContext`：投递到指定队列。

不需要请求上下文时，也可以使用对应的不带 `Context` 后缀的方法。

队列采用至少一次投递语义。同一消息在超时、进程退出或网络异常时可能再次执行，因此消费者必须使用业务唯一键、状态检查或幂等写入避免重复副作用。

## 队列与自动恢复

`task.queues` 按从高到低的顺序配置权重，例如 `critical`、`default`、`low`。`task.concurrency` 控制单进程并发消费数。

Worker 由监督循环管理：启动报错或 panic 后会销毁异常实例并重新创建，等待时间从 `worker_restart_delay_seconds` 开始指数增长，最高不超过 `worker_restart_max_delay_seconds`。运行期间 Redis 短暂断连由 Asynq 自身持续重连；单个业务处理器 panic 会被恢复为任务错误并触发消息重试，不会带崩消费进程。

Adjust 用户事件上报也复用该 Worker，任务类型为 `adjust:event`，使用 `default` 队列。请勿从 `task.queues` 中移除 `default`，否则事件只会积压而不会被消费。

Adjust tracker 同步任务类型为 `adjust:trackers:sync`，每 15 分钟投递到 `default` 队列。每次任务先通过 `GetTrackers` 分页拉取全部一级数据，再只选择一个一级类目，使用其 token 调用 `GetTrackerChildren`，按深度优先递归同步完整子树；子级列表同样处理 cursor 分页。未同步过的一级类目优先，之后按本地 `updated_at` 从旧到新轮转，避免进程重启后总是重复首个类目。记录按 token 幂等写入 `video_adjust_media_ads`，子级 `pid` 指向父级的本地数据库 ID。启用前需配置 `adjust.tracker_sync_enabled`、`adjust.campaign_api_token` 和 `adjust.campaign_app_token`。

```yaml
task:
  concurrency: 10
  worker_restart_delay_seconds: 1
  worker_restart_max_delay_seconds: 30
  queues:
    - critical
    - default
    - low
```

应用退出时调用 `manager.Close()` 会停止消费、等待正在处理的任务完成，并关闭消息投递客户端。
