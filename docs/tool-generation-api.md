# 按工具创建生成任务

## 接口

`POST /api/generation/tool-tasks`

需要 Bearer JWT，请求体为 `application/json`。这是独立于通用生成接口的工具任务协议，客户端不能指定模型、任务类型、提示词或模型参数。

服务端会根据 `tool_id` 读取在线工具，并自动确定：

- `tool_type=1` 创建图片任务，`tool_type=2` 创建视频任务；
- 使用工具关联的启用模型及工具提示词；
- 使用该模型的服务端默认参数，由现有模型适配器转换为上游请求；
- 任务持久化 `tool_config_id`，并将 `source_type` 设为 `3`（工具）。

## 请求示例

```json
{
  "tool_id": 16,
  "config_type": 2,
  "image": "/uploads/images/2026/08/26/character.png",
  "video": "/uploads/videos/2026/08/26/motion.mp4",
  "client_request_id": "tool-task-20260826-0001"
}
```

## 字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `tool_id` | integer | 是 | 在线且未删除的工具 ID |
| `config_type` | integer | 是 | 必须与工具配置一致：1=无，2=参考图，3=年龄，4=比例 |
| `val` | string | 否 | 工具配置选择值 |
| `image` | string | 条件必填 | 单张图片地址；支持 `/uploads/...` 半链接和 HTTP(S) 全链接 |
| `video` | string | 条件必填 | 单个视频地址；支持 `/uploads/...` 半链接和 HTTP(S) 全链接 |
| `client_request_id` | string | 否 | 最长 64 字符的幂等标识；仅支持字母、数字、点、下划线和中划线 |

`image` 和 `video` 至少传一个。图片类工具必须仅传 `image`；视频类工具可以传单图、单视频或两者，最终组合仍需满足工具所关联模型的配置要求。

本接口不接收二进制文件。客户端应先通过 `POST /api/uploads/oss/signature` 将图片或视频直传 OSS，然后将返回的 `file_url` 传入本接口。

## 响应

成功响应与通用生成任务快照一致，其中：

- `tool_id` 为本次使用的工具 ID；
- `source_type` 固定为 `3`；
- 任务先持久化为待提交状态，再由现有后台 worker 调用工具所配置的模型。

工具不存在或已下线时返回 HTTP 404；媒体组合、工具模型或提示词配置无效时返回 HTTP 400。
