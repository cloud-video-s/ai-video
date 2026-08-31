# 任务管理 API

本文档描述管理后台“运营管理 / 任务管理”查询客户端用户图片、视频生成任务及预览生成结果的只读接口。接口统一使用 `/admin` 前缀，需要管理员 JWT，并受 Casbin API 权限控制。

## 接口清单

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/user-generation-tasks` | 分页查询所有客户端用户的生成任务 |
| `GET` | `/admin/user-generation-tasks/:id` | 查询任务详情、请求响应及媒体地址 |

菜单路由为 `/operation/generation-tasks`，权限标识为 `user:generation-task:list`。菜单和 API 元数据由显式 `SeedAdminMetadata` 入口维护，应用启动不会自动写入数据库。

## 分页查询

`GET /admin/user-generation-tasks`

### Query 参数

| 参数 | 必填 | 说明 |
| --- | ---: | --- |
| `page` | 否 | 页码，默认 `1` |
| `page_size` | 否 | 每页数量，默认 `10`，最大 `100` |
| `user_id` | 否 | 客户端用户 ID |
| `model_id` | 否 | 生成模型 ID |
| `media_type` | 否 | `1` 图片，`2` 视频 |
| `status` | 否 | 任务状态，取值 `1` 至 `7` |
| `task_code` | 否 | 精确匹配任务编码 |
| `keyword` | 否 | 模糊匹配任务号、提示词、模型名称/编码、用户昵称、邮箱、账号或设备标识 |
| `date_from` | 否 | 创建日期下限，格式 `YYYY-MM-DD`，包含当天 |
| `date_to` | 否 | 创建日期上限，格式 `YYYY-MM-DD`，包含当天 |

状态值：

| 值 | `status_name` | 说明 |
| ---: | --- | --- |
| `1` | `submitting` | 待提交或提交中 |
| `2` | `submitted` | 已提交 |
| `3` | `pending` | 等待处理 |
| `4` | `running` | 生成中 |
| `5` | `downloading` | 下载并持久化结果中 |
| `6` | `success` | 成功 |
| `7` | `failure` | 失败 |

### 成功响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 101,
        "user_id": 7,
        "model_id": 11,
        "template_id": 0,
        "client_request_id": "client-request-001",
        "task_code": "task-20260728-001",
        "third_task_code": "provider-task-001",
        "status": 6,
        "status_name": "success",
        "task_type": 2,
        "progress": 100,
        "media_type": "video",
        "prompt": "海边日落的电影感镜头",
        "remote_urls": ["https://provider.example/result.mp4"],
        "local_urls": ["/storage/videos/result.mp4"],
        "preview_urls": ["/storage/videos/result.mp4"],
        "cover_image_url": "/storage/images/task-cover.jpg",
        "result_count": 1,
        "error_message": "",
        "usage_duration": 18,
        "score": 12,
        "submitted_at": "2026-07-28T10:00:01+08:00",
        "started_at": "2026-07-28T10:00:02+08:00",
        "finished_at": "2026-07-28T10:00:19+08:00",
        "last_polled_at": "2026-07-28T10:00:19+08:00",
        "created_at": "2026-07-28T10:00:00+08:00",
        "updated_at": "2026-07-28T10:00:19+08:00",
        "user": {
          "id": 7,
          "username": "Alice",
          "email": "alice@example.com",
          "login_account": "alice-login",
          "imei": "",
          "device_code": "device-7"
        },
        "model": {
          "id": 11,
          "name": "Video Model",
          "code": "video-model",
          "model_type": 2,
          "version": "v1"
        }
      }
    ],
    "total": 1,
    "page": 1,
    "size": 20
  }
}
```

列表响应不返回体积较大的 `request_payload` 和 `provider_response`，这两个字段仅在详情接口返回。

## 查询详情

`GET /admin/user-generation-tasks/:id`

详情结构与列表项一致，并增加：

- `request_payload`：可解析的 JSON 返回对象或数组；历史脏数据无法解析时返回原始字符串。
- `provider_response`：可解析的 JSON 返回对象或数组；无法解析时返回原始字符串。

任务不存在时返回业务错误码 `10002`。

## 媒体预览规则

- `remote_urls` 和 `local_urls` 始终返回字符串数组，兼容数据库中的 JSON 数组及逗号、分号、竖线或换行分隔的历史格式。
- `preview_urls` 优先使用 `local_urls`；没有本地结果时回退到 `remote_urls`。
- `media_type` 优先根据任务快照字段 `task_type` 返回 `image` 或 `video`；历史数据缺失时回退到关联模型及结果文件扩展名，仍无法判断时返回 `unknown`。
- 管理端图片使用图片查看器放大预览，视频使用浏览器原生播放器并保留“在新窗口打开”入口。
