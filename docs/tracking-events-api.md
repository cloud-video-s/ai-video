# 客户端数据埋点上报

## 接口

- 方法：`POST /api/tracking/events`
- 鉴权：与其他客户端接口一致，需要 Bearer Token 和客户端公共请求头。
- 语义：一次请求只上报一个事件；每次成功请求都会新增一条记录，不去重、不覆盖。

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `tracking_type` | string | 是 | 埋点事件类型，只支持下表中的九个值，且大小写敏感 |
| `extension_type` | string | 条件必填 | `Payment_Create`、`Payment_Suc`、`Case_create` 必填；其他类型可省略 |
| `model_id` | integer | 否 | 模板 ID；不适用时可省略或传 `0` |

请求示例：

```json
{
  "tracking_type": "Payment_Create",
  "extension_type": "OB_back",
  "model_id": 123
}
```

成功响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 12345,
    "tracking_type": "Payment_Create",
    "extension_type": "OB_back",
    "model_id": 123,
    "created_at": "2026-08-20T10:11:12+08:00"
  }
}
```

上例只展示客户端需要依赖的稳定业务字段。当前响应记录还包含服务端保存的应用、安装包、版本、渠道、国家、设备型号和系统类型快照；客户端不应使用这些响应快照覆盖本地环境信息。

## 埋点类型

| `tracking_type` | 说明 | 是否必须提供 `extension_type` |
| --- | --- | --- |
| `OB_Payment_show` | 进入 OB 付费页 | 否 |
| `OB_Payment_back_show` | 进入 OB 付费返回拦截页 | 否 |
| `Home_Show` | 进入主页 | 否 |
| `Launc_Payment_Show` | 老用户进入主页展示付费页 | 否 |
| `Launc_Payment_back_Show` | 老用户进入主页展示返回拦截付费页 | 否 |
| `Payment_Show` | 进入付费页 | 否 |
| `Payment_Create` | 创建订单 | 是 |
| `Payment_Suc` | 付费成功 | 是 |
| `Case_create` | 创建任务 | 是 |

名称大小写敏感，并原样保留需求表中的现有拼写。

## 扩展来源与模板

`extension_type` 是单个来源标识字符串，不是 JSON 对象。当前服务只校验后三种埋点必须提供该字段，不限制具体枚举；客户端应使用双方约定的稳定值，避免同一来源产生不同拼写。

当前约定：

- `Payment_Create`、`Payment_Suc`：可用 `OB`、`OB_back`、`Launch`、`Launch_back`、`User_center`、`Ai_image`、`Ai_video` 等值标识来源。
- `Case_create`：可用 `Ai_image`、`Ai_video` 等值标识任务来源。
- 需要关联模板时，通过独立的 `model_id` 数值字段传递，不要把 ID 拼进 `extension_type`。

## 存储与部署

表名为 `video_tracking_event`。当前 Go 代码读写 `tracking_type`、`extension_type` 和 `model_id`，并保存应用、包、版本、渠道、国家、设备型号和系统类型快照。

现有 `scripts/schema/20260820_001_create_video_tracking_event.sql` 仍定义旧字段 `data_type` 和 `extended_fields`，与当前代码不一致。部署前必须另行创建并审核新的版本化 SQL 变更，使真实数据库结构与当前代码一致；本文档更新没有执行或授权任何数据库变更。
