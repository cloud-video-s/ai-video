# 客户端 API 接口文档

本文档对应当前项目实际注册的 `/api` 路由。OpenAPI 版本为 3.0.3，仅包含客户端接口，不包含 `/admin` 后台接口。

## 文档入口

- 在线文档：`http://localhost:8080/docs/ui`
- 在线 OpenAPI：`http://localhost:8080/docs/openapi.json`
- 静态 OpenAPI：[openapi.json](openapi.json)

重新生成静态文档：

```powershell
go run ./cmd/apidocgen -output docs/openapi.json
```

## 鉴权

除健康检查、游客登录接口 `/api/auth/login`、Apple 服务端通知回调和 Adjust 服务端回调外，其他客户端接口均需要在 Header 中携带：

```http
Authorization: Bearer <JWT>
```

JWT 可由 `POST /api/auth/login`、`POST /api/auth/apple_order_login` 或 `POST /api/auth/refresh` 返回。后两个接口调用时仍要求当前有效的 Bearer Token。`Authorization` 属于鉴权信息，不作为普通请求参数重复列在各接口参数表中。

`POST /api/auth/refresh` 使用当前未过期的 Bearer Token 换取新 Token。刷新成功后，原 Token 立即失效，客户端应立即替换本地 Token。

Adjust 回调不使用客户端 JWT，而是通过 `callback_token` Query/Form 参数或 `X-Adjust-Callback-Token` Header 使用独立密钥鉴权。

## API 公共请求参数

下列 Header 由受保护接口统一读取，各接口的专属参数中不再重复介绍：

| Header | 必填 | 说明 |
|---|---:|---|
| `Video_App_Code` | 是 | 应用代码，对应应用配置中的 `app_code` |
| `Video_App_Package_Code` | 是 | 应用包代码，对应安装包配置中的 `package_code` |
| `Video_App_Version` | 是 | 当前应用版本号 |
| `Video_Phone_Model` | 是 | 客户端设备型号 |
| `Video_Channel_Code` | 是 | 渠道代码，对应渠道配置中的 `channel_code` |
| `Video_Device_Country` | 否 | ISO 3166-1 alpha-2 国家或地区代码；用于内容投放和语言选择，未传时可根据客户端 IP 推断 |
| `Accept-Language` | 否 | 国家未配置语言时的回退语言，例如 `zh-CN`、`en-US` |

## 响应语言

语言不与安装包关联。服务按以下顺序确定响应语言：

1. `Video_Device_Country` 对应的启用国家配置语言。
2. 国家未配置语言或查询不到国家时，使用 `Accept-Language`。
3. 未传 `Accept-Language` 但存在国家代码时，按国家映射到当前支持的语言。
4. 仍无法确定时使用 `zh-CN`。

当前支持 `zh-CN`、`en-US`、`ja-JP`、`ko-KR`、`es-ES`。最终语言通过响应头 `Content-Language` 返回。

## 统一响应结构

普通 JSON 接口统一返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

## 当前接口清单

### 公共接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/health` | 健康检查 |
| POST | `/api/auth/login` | 游客登录或创建设备账号 |
| POST | `/api/payments/apple/notification` | 接收 App Store Server Notifications V2 回调；由 Apple 服务器调用，无需客户端鉴权与公共请求头 |
| GET | [`/api/attributions/adjust/callback`](adjust-attribution-callback.md) | 接收 Adjust 服务端 GET 回调；使用独立的 callback token 鉴权 |
| POST | [`/api/attributions/adjust/callback`](adjust-attribution-callback.md) | 接收 Adjust 服务端 POST 回调；使用独立的 callback token 鉴权 |

### 用户与认证

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/auth/apple_order_login` | 根据 Apple 原始交易 ID 数组 `order_code` 查询关联用户并签发登录 Token |
| POST | `/api/auth/refresh` | 刷新当前 API Token；无请求体，返回结构与登录接口相同 |
| POST | `/api/auth/logout` | 退出登录 |
| POST | `/api/third_binding` | 绑定或切换第三方身份 |
| GET | `/api/users/me` | 查询当前用户 |
| PUT | `/api/users/me/country` | 更新用户国家或地区 |
| GET | `/api/users/points` | 分页查询当前用户的积分变动明细 |
| POST | `/api/users/active_reporting` | 上报当前用户的活跃时长 |

#### 根据 Apple 订单登录

`POST /api/auth/apple_order_login`

该接口位于认证路由组，调用方必须携带当前有效的 Bearer Token 和 API 公共请求 Header。服务端将 `order_code` 数组中的值去除首尾空白后，与订单表的 `original_transaction_id` 匹配；仅查询 VIP 订阅订单，并按订单 ID 倒序选取最新一条。随后读取订单关联用户，按该用户当前的登录类型、设备标识和 Token 版本签发新的客户端 Token。若启用了单设备登录，新 Token 签发时会按现有登录策略递增关联用户的 Token 版本。

当订单属于另一个账号时：若目标账号已绑定第三方身份，接口返回脱敏邮箱并要求客户端引导用户使用对应账号登录；若目标账号未绑定第三方身份，客户端首次调用应传 `force_new=false` 获取切换确认，用户确认后再以 `force_new=true` 重试。

请求体：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `order_code` | string[] | 是 | Apple 原始交易 ID 数组，至少 1 项、最多 191 项；服务端按数组中的值匹配订单 |
| `force_new` | boolean | 否 | 是否确认切换到其他未绑定第三方身份的订单账号；默认 `false` |

请求示例：

```json
{
  "order_code": ["2000001209105682", "2000001209105683"],
  "force_new": false
}
```

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOi...",
    "login_type": 3,
    "expire_at": 1785736800,
    "token_version": 2
  }
}
```

错误响应：

| HTTP 状态 | `code` | 场景 |
|---:|---:|---|
| 400 | `10001` | 请求体无效，或 `order_code` 数组为空/超过 191 项 |
| 401 | `401` | 当前 Bearer Token 或公共请求 Header 无效 |
| 200 | `40001` | 订单属于另一个已绑定第三方身份的账号；`message` 包含脱敏邮箱 |
| 200 | `40002` | 订单属于另一个未绑定第三方身份的账号，需要用户确认切换 |
| 200 | `20004` | 订单关联用户已禁用 |
| 200 | `10002` | 未找到匹配的 VIP 订阅订单 |
| 200 | `20002` | 订单关联用户不存在 |
| 200 | `10000` | 查询或 Token 签发失败 |

#### 查询积分明细

`GET /api/users/points`

Query 参数：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `page` | integer | 否 | 页码，从 1 开始；默认 1 |
| `page_size` | integer | 否 | 每页数量；默认 10 |
| `points_type` | integer | 否 | 积分变动方向：`1` 收入，`2` 支出；默认 1 |
| `start_time` | integer | 否 | 筛选开始时间（Unix 秒）；仅与 `end_time` 同时提供时生效 |
| `end_time` | integer | 否 | 筛选结束时间（Unix 秒）；仅与 `start_time` 同时提供时生效 |

响应 `data` 为分页对象，包含 `page`、`pageSize`、`total`、`totalPages` 和 `list`。`list` 中每条积分明细包含变动方向、变动量、变动前后余额、说明及创建/更新时间。

成功响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "page": 1,
    "pageSize": 10,
    "total": 1,
    "totalPages": 1,
    "list": [
      {
        "id": 1,
        "user_id": 8,
        "direction": 1,
        "points_change": 100,
        "balance_before": 20,
        "balance_after": 120,
        "description": "购买 VIP 月卡赠送",
        "created_at": 1785816000,
        "updated_at": 1785816000
      }
    ]
  }
}
```

#### 上报用户活跃时长

`POST /api/users/active_reporting`

请求体：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `time_long` | integer | 是 | 本次上报的活跃时长，单位秒，必须大于 0 |

请求示例：

```json
{
  "time_long": 300
}
```

成功时返回：

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

### 客户端配置与内容

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/ob_delay` | 获取扁平数值格式的延迟配置对象 |
| GET | `/api/configs/list` | 获取客户端公开配置 |
| GET | `/api/banners/list` | 按展示位置、国家、应用、包、版本及会员状态查询 Banner；范围未绑定表示全部 |
| GET | `/api/tools/list` | 查询全部在线且未删除的工具；按排序值和 ID 升序返回 |
| GET | `/api/templates/categories` | 分页查询当前客户端可见且包含可用模板的分类；默认每页 5 个分类 |
| GET | `/api/templates/recommend` | 查询推荐模板 |
| GET | `/api/templates/list` | 分页查询分类及其模板 |
| GET | `/api/templates/template_list` | 分页查询分类模板 |
| GET | `/api/templates/template_info` | 查询模板详情 |
| POST | `/api/templates/:id/favorite` | 收藏模板 |
| DELETE | `/api/templates/:id/favorite` | 取消收藏模板 |
| POST | `/api/templates/complaint` | 提交模板投诉 |
| GET | `/api/vip/recommend` | 查询推荐 VIP 套餐 |
| GET | `/api/vip/list` | 按 `vip_types` 查询当前客户端可展示的 VIP 套餐列表 |

#### 模板查询接口统一返回结构

以下五个模板查询接口均已使用当前统一的模板返回对象：

- `GET /api/templates/categories`
- `GET /api/templates/list`
- `GET /api/templates/recommend`
- `GET /api/templates/template_list`
- `GET /api/templates/template_info`

模板对象字段：

| 字段               | 类型 | 必定返回 | 说明 |
|--------------------|---|---:|---|
| `id`               | integer | 是 | 模板 ID |
| `template_type_id` | integer | 是 | 模板分类 ID |
| `name`             | string | 是 | 模板名称 |
| `template_type`    | integer | 是 | 模板类型：`1` 图片模板，`2` 视频模板 |
| `cover_image_url`  | string | 是 | 模板封面图片地址 |
| `original_url`     | string | 是 | 模板原始媒体地址 |
| `thumbnail_url`    | string | 是 | 模板缩略媒体地址 |
| `prompt`           | string | 是 | 模板提示词 |
| `description`      | string | 是 | 模板说明 |
| `sort`             | integer | 是 | 排序值 |
| `usage_count`      | integer | 是 | 使用次数 |
| `favorite_count`   | integer | 是 | 收藏次数 |
| `view_count`       | integer | 是 | 浏览次数 |
| `is_favorite`      | integer | 是 | 当前用户是否已收藏：`0` 否，`1` 是 |
| `model_score`      | integer | 是 | 生成模型评分 |

当前结构不再返回旧字段 `video_template_type_id`、`cover_image`、`template_video`、`thumbnail_video`、`model_id`、`model_code`、`model_name` 或 `model_parameters`。

`GET /api/templates/categories`、`GET /api/templates/list` 和 `GET /api/templates/template_list` 的响应 `data` 使用统一分页对象：`page` 为当前页码，`pageSize` 为每页数量，`total` 为总记录数，`totalPages` 为总页数，`list` 为当前页数据。`GET /api/templates/recommend` 仍返回模板数组，`GET /api/templates/template_info` 仍返回单个模板对象。

分类对象字段：

| 字段 | 类型 | 必定返回 | 说明 |
|---|---|---:|---|
| `id` | integer | 是 | 分类 ID |
| `category_name` | string | 是 | 分类名称 |
| `description` | string | 是 | 分类说明 |
| `sort` | integer | 是 | 分类排序值 |
| `templates` | array | 是 | 使用上述统一结构的模板数组 |

#### 分页查询模板分类

`GET /api/templates/categories`

该接口需要 Bearer JWT 和公共请求 Header。服务端固定查询 `homeCategory` 展示位置，并根据 `Video_Device_Country`、`Video_App_Code`、`Video_App_Package_Code` 和 `Video_App_Version` 匹配分类投放范围。某个维度没有关联记录时表示支持全部；存在关联记录时必须命中启用且未删除的目标配置。

分类必须处于启用状态，并且至少关联一个启用且未删除的模板。没有关联模板、仅关联禁用模板或仅关联已删除模板的分类会在分页前被排除，因此不会占用每页分类名额。

Query 参数：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `page` | integer | 否 | 分类页码，从 1 开始；未传时按第 1 页处理 |
| `page_size` | integer | 否 | 每页分类数量；默认 5 |

分页与排序规则：

- `page_size` 未传或为 0 时默认每页返回 5 个分类。
- 每个分类固定最多返回 10 个模板。
- 分类按 `sort DESC, id DESC` 排序。
- 模板按 `sort DESC, usage_count DESC, like_count DESC, view_count DESC, id DESC` 排序。
- 响应 `data` 是分页对象，分类数组位于 `data.list`；当页没有数据时 `list` 为空数组。

成功响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "page": 1,
    "pageSize": 5,
    "total": 1,
    "totalPages": 1,
    "list": [
      {
        "id": 1,
        "category_name": "热门模板",
        "description": "当前客户端可用的热门模板",
        "sort": 100,
        "templates": [
          {
            "id": 101,
            "template_type_id": 1,
            "name": "动漫视频",
            "template_type": 2,
            "cover_image_url": "https://cdn.example.com/templates/101-cover.jpg",
            "original_url": "https://cdn.example.com/templates/101.mp4",
            "thumbnail_url": "https://cdn.example.com/templates/101-thumbnail.mp4",
            "prompt": "生成动漫风格视频",
            "description": "动漫风格模板",
            "sort": 100,
            "usage_count": 120,
            "favorite_count": 18,
            "view_count": 360,
            "is_favorite": 1,
            "model_score": 95
          }
        ]
      }
    ]
  }
}
```

#### 查询分类及其模板

`GET /api/templates/list`

Query 参数：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `page` | integer | 否 | 分类页码，从 1 开始；默认 1 |
| `page_size` | integer | 否 | 每页分类数量；默认 5 |
| `position_key` | string | 否 | 展示位置标识；未传时仅匹配未限定展示位置的分类 |

响应 `data` 为分页对象，分类数组位于 `data.list`；每个分类的 `templates` 均使用当前统一模板对象结构。

#### 查询推荐模板

`GET /api/templates/recommend`

Query 参数：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `position_key` | string | 是 | 推荐模板展示位置标识，最长 64 个字符 |

响应 `data` 为统一模板对象数组，不再返回 `display_config_id`、`position_key` 或 `display_sort` 等展示配置字段。

#### 分页查询分类模板

`GET /api/templates/template_list`

Query 参数：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `page` | integer | 否 | 页码，从 1 开始；默认 1 |
| `page_size` | integer | 否 | 每页模板数量；默认 10 |
| `template_type_id` | integer | 否 | 模板分类 ID |
| `position_key` | string | 否 | 展示位置标识，最长 64 个字符 |

响应 `data` 为分页对象，模板数组位于 `data.list`，并返回 `page`、`pageSize`、`total` 和 `totalPages`。

#### 查询模板详情

`GET /api/templates/template_info`

Query 参数：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `template_id` | integer | 是 | 模板 ID，必须大于 0 |

响应 `data` 为单个统一模板对象。服务端会根据当前登录用户设置 `is_favorite`。

### 内容生成

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/generation/models` | 按模型类型查询启用模型及其参数 |
| POST | `/api/generation/tasks` | 校验并创建待异步处理的生成任务，返回任务订单号 |
| POST | [`/api/generation/template-tasks`](template-generation-api.md) | 按模板创建图片或视频生成任务 |
| POST | [`/api/generation/tool-tasks`](tool-generation-api.md) | 按工具配置创建独立的图片或视频生成任务 |
| GET | `/api/generation/tasks` | 分页查询当前用户生成任务 |
| GET | `/api/generation/tasks/:id` | 查询生成任务详情 |
| DELETE | `/api/generation/tasks/:id` | 删除生成任务 |

视频任务提交成功并写入 `third_task_code` 后，后台 worker 每 3 秒按该字段请求 ModelVerse `GET /v1/tasks/status?task_id=<third_task_code>`。`Pending`、`Running`、`Success`、`Failure` 分别更新为本地等待、生成中、结果保存中和失败状态；`Success` 返回的 `output.urls` 会继续进入配置驱动的本地或 OSS 保存流程。协议参考 [UCloud Kling V3 Omni 任务状态](https://astraflow.ucloud.cn/reference/modelverse-api-protocol/video/kling-o3/get-kling-o3-task-status)。

创建普通任务或模板任务时，客户端媒体参数 `input.images[]`、`input.video`、`input.first_frame`、`input.end_frame` 同时支持以下两种格式：

- 半链接，例如 `/uploads/images/reference.png`、`/uploads/videos/reference.mp4`；
- HTTP(S) 全链接，例如 `https://cdn.example.com/uploads/images/reference.png`。

半链接会在调用生成服务前使用当前文件代理/CDN 域名补全；已经是全链接的地址不会重复拼接域名。参数值必须是纯 URL 字符串，不能使用 Markdown 链接格式。

#### 查询启用模型及参数

`GET /api/generation/models`

该接口需要 Bearer JWT 和公共请求 Header。服务端根据 `model_type` 查询，仅返回平台与模型均处于启用状态且未删除的模型。每个模型返回全部 `parameter_type=2` 的请求参数，以及 `is_display=1` 的 `parameter_type=1` 选项参数；参数按 `parameter_type ASC, sort_order ASC, id ASC` 排序。

Query 参数：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `model_type` | integer | 是 | 模型类型，必须大于 0；当前约定 `1` 为生成图片、`2` 为生成视频 |

成功响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "name": "Kling v3 视频生成",
      "model_code": "kling-v3",
      "score": 10,
      "icon": "",
      "description": "描述",
      "parameter": [
        {
          "param_key": "aspect_ratio",
          "default_value": "16:9",
          "allowed_values": ["16:9", "9:16", "1:1"],
          "allowed_value_options": [
            {"value": "16:9", "alias": "横屏"},
            {"value": "9:16", "alias": "竖屏"},
            {"value": "1:1", "alias": "方形"}
          ],
          "description": "生成视频的宽高比",
          "parameter_type": 1,
          "constraints": "{}",
          "alias": "画面比例",
          "display_type": "select",
          "is_display": 1
        },
        {
          "param_key": "prompt",
          "default_value": null,
          "allowed_values": [],
          "allowed_value_options": [],
          "description": "生成提示词",
          "parameter_type": 2,
          "constraints": "{\"max_length\": 2500}",
          "alias": "提示词",
          "display_type": "textarea",
          "is_display": 1
        }
      ]
    }
  ]
}
```

参数对象字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `param_key` | string | 提交生成请求时使用的参数键 |
| `default_value` | any | 参数默认值；没有默认值时为 `null` |
| `allowed_values` | any[] | 可提交的参数值数组 |
| `allowed_value_options` | object[] | 选择值配置；每项用 `{value, alias}` 将模型值和展示别名直接绑定，没有可选值时为 `[]` |
| `description` | string | 参数说明 |
| `parameter_type` | integer | `1` 选项参数，`2` 请求参数 |
| `constraints` | string | JSON 字符串格式的约束；客户端需要时再解析为 JSON 对象，`{}` 表示没有额外约束 |
| `alias` | string | 客户端展示名称 |
| `display_type` | string | 建议使用的客户端控件类型 |
| `is_display` | integer | 是否展示：`1` 是，`0` 否；`parameter_type=2` 的请求参数仍可能返回 |

为保持旧客户端兼容，`allowed_values` 继续返回。新客户端应使用 `allowed_value_options` 获取成对的模型值与展示别名；管理端旧请求仅传 `allowed_values` 仍可保存，此时别名默认使用值本身。若管理端同时提交两个字段，服务端会校验每个 `allowed_values[i]` 与 `allowed_value_options[i].value` 一致。

模型无参数时，`parameter` 返回空数组 `[]`，不会返回 `null`。

#### 查询生成任务列表

`GET /api/generation/tasks`

Query 参数：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `page` | integer | 否 | 页码，从 1 开始 |
| `page_size` | integer | 否 | 每页数量，必须大于 0；注意响应字段使用驼峰命名 `pageSize` |
| `task_type` | integer | 否 | 任务类型筛选：`1` 生成图片，`2` 生成视频，`3` 不限制类型 |
| `status` | integer | 否 | 任务状态筛选；当前请求参数支持 `1`、`2`、`3`，未传时不限制状态 |

成功响应中的 `data` 是分页对象，`list` 中的每一项是完整任务快照：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "page": 1,
    "pageSize": 10,
    "total": 2,
    "totalPages": 1,
    "list": [
      {
        "id": 13,
        "task_code": "eafe15f0-780f-4a7f-9c62-7e99484be521",
        "task_type": 2,
        "status": 7,
        "progress": 100,
        "input": {
          "end_frame": "https://cdn.example.com/uploads/images/end-frame.png",
          "first_frame": "https://cdn.example.com/uploads/images/first-frame.png",
          "prompt": "生成一个可爱风格的动漫视频",
          "video": "https://cdn.example.com/uploads/videos/reference.mp4"
        },
        "parameters": {
          "aspect_ratio": "1:1",
          "duration": 10,
          "external_task_id": "eafe15f0-780f-4a7f-9c62-7e99484be521",
          "mode": "std"
        },
        "local_urls": [],
        "error_message": "provider task id not found: field=data.task_id, provider_error=Tail image is not supported with video input",
        "usage_duration": 10,
        "submitted_at": "2026-07-30T14:44:41.602+08:00",
        "started_at": "2026-07-30T14:44:43.804+08:00",
        "finished_at": "2026-07-30T14:48:33.154+08:00",
        "created_at": "2026-07-30T14:44:28.235+08:00",
        "updated_at": "2026-07-30T14:48:33.155+08:00"
      },
      {
        "id": 11,
        "task_code": "d9dd18d6-6cd4-4df3-8dff-f7c3622990b7",
        "task_type": 2,
        "status": 6,
        "progress": 100,
        "input": {
          "end_frame": "",
          "first_frame": "",
          "images": [
            "https://cdn.example.com/uploads/images/reference-1.png",
            "https://cdn.example.com/uploads/images/reference-2.png"
          ],
          "prompt": "生成一个可爱风格的动漫视频",
          "video": ""
        },
        "parameters": {
          "aspect_ratio": "1:1",
          "duration": 10,
          "external_task_id": "d9dd18d6-6cd4-4df3-8dff-f7c3622990b7",
          "mode": "std"
        },
        "local_urls": [
          "https://cdn.example.com/uploads/generated/1/task-11-1.mp4"
        ],
        "usage_duration": 10,
        "submitted_at": "2026-07-30T09:26:25.938+08:00",
        "finished_at": "2026-07-30T11:24:22.563+08:00",
        "created_at": "2026-07-30T09:21:32.387+08:00",
        "updated_at": "2026-07-30T11:24:39.676+08:00"
      }
    ]
  }
}
```

#### 查询生成任务详情

`GET /api/generation/tasks/:id`

路径参数 `id` 是任务记录 ID。成功响应的 `data` 直接返回单个任务对象，字段与列表中的 `data.list[]` 完全一致，不包含分页字段：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 11,
    "task_code": "d9dd18d6-6cd4-4df3-8dff-f7c3622990b7",
    "task_type": 2,
    "status": 6,
    "progress": 100,
    "input": {
      "images": [
        "https://cdn.example.com/uploads/images/reference-1.png",
        "https://cdn.example.com/uploads/images/reference-2.png"
      ],
      "prompt": "生成一个可爱风格的动漫视频"
    },
    "parameters": {
      "aspect_ratio": "1:1",
      "duration": 10,
      "external_task_id": "d9dd18d6-6cd4-4df3-8dff-f7c3622990b7",
      "mode": "std"
    },
    "local_urls": [
      "https://cdn.example.com/uploads/generated/1/task-11-1.mp4"
    ],
    "usage_duration": 10,
    "submitted_at": "2026-07-30T09:26:25.938+08:00",
    "finished_at": "2026-07-30T11:24:22.563+08:00",
    "created_at": "2026-07-30T09:21:32.387+08:00",
    "updated_at": "2026-07-30T11:24:39.676+08:00"
  }
}
```

#### 生成任务响应字段

列表接口的任务字段位于 `data.list[]`，详情接口的相同字段位于 `data`。

| 字段 | 类型 | 必定返回 | 说明 |
|---|---|---:|---|
| `id` | integer | 是 | 任务记录 ID |
| `task_code` | string | 是 | 任务唯一编码 |
| `task_type` | integer | 是 | 任务类型：`1` 生成图片，`2` 生成视频 |
| `status` | integer | 是 | 任务状态，枚举见下表 |
| `progress` | integer | 是 | 任务进度，范围 `0-100` |
| `input` | object | 否 | 提交给生成模型的输入快照；可能包含 `prompt`、`images`、`video`、`first_frame`、`end_frame` 等字段 |
| `parameters` | object | 否 | 模型参数快照；可能包含 `aspect_ratio`、`duration`、`mode`、`external_task_id` 等字段 |
| `local_urls` | string[] | 是 | 已持久化的生成结果地址；没有结果时返回 `[]`，不会返回 `null` |
| `error_message` | string | 否 | 任务失败原因；没有错误时省略 |
| `usage_duration` | integer | 是 | 任务计费用时，单位秒 |
| `submitted_at` | string(date-time) | 否 | 任务提交到上游的时间，RFC 3339 格式 |
| `started_at` | string(date-time) | 否 | 上游开始处理的时间，RFC 3339 格式 |
| `finished_at` | string(date-time) | 否 | 任务成功或失败的结束时间，RFC 3339 格式 |
| `created_at` | string(date-time) | 是 | 任务创建时间，RFC 3339 格式 |
| `updated_at` | string(date-time) | 是 | 最近更新时间，RFC 3339 格式 |

任务状态：

| 值 | 状态 | 说明 |
|---:|---|---|
| `1` | 提交中 | 正在向上游提交任务 |
| `2` | 已提交 | 上游已接收任务 |
| `3` | 等待处理 | 上游任务处于等待状态 |
| `4` | 运行中 | 上游正在生成内容 |
| `5` | 下载中 | 正在持久化上游生成结果 |
| `6` | 成功 | 任务完成，结果见 `local_urls` |
| `7` | 失败 | 任务失败，原因见 `error_message` |

新版任务对象不再向客户端返回 `client_request_id`、`model_id`、`third_task_code`。`input`、`parameters`、`error_message` 以及三个业务时间字段没有值时会被省略，而不是返回 `null`。

### 积分商品与支付订单

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/points/list` | 查询当前客户端可购买的积分商品 |
| POST | `/api/orders` | 创建 VIP 或积分商品的待支付订单，并返回原生商店支付参数 |

`GET /api/points/list` 无业务 Query 参数。服务端根据当前登录用户类型以及公共请求上下文中的国家、应用、安装包、APP 版本、系统和渠道进行筛选，仅返回已启用且适用于当前客户端的商品；结果按默认商品优先、排序值升序和 ID 降序排列。

`POST /api/orders` 请求体：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `shop_type` | integer | 是 | 商品类型：`1` VIP 订阅，`2` 积分商品 |
| `product_id` | integer | 是 | 服务端商品 ID |
| `pay_type` | integer | 是 | 支付类型：`1` Apple IAP，`2` Google Play |
| `client_request_id` | string | 否 | 客户端幂等请求 ID，最长 64 个字符；未传时由服务端生成并返回 |

响应 `data` 包含服务端订单号、商品快照、应付金额、过期时间和 `payment_info`。Apple 场景返回 `bundle_id` 与 `confirm_path`，Google Play 场景返回 `package_name`；积分商品的 `payment_info.product_type` 为 `inapp`，VIP 商品为 `subscription`。

### 归因与数据上报

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | [`/api/attributions/adjust/report`](adjust-attribution-callback.md#app-客户端上报) | 上报 Adjust SDK 当前归因快照，并与服务端回调融合 |
| POST | [`/api/tracking/events`](tracking-events-api.md) | 上报单个客户端埋点事件；成功请求逐条新增记录，不去重 |

埋点请求使用 `tracking_type`、`extension_type` 和 `model_id`。事件名只支持当前约定的九个大小写敏感值；`Payment_Create`、`Payment_Suc` 和 `Case_create` 必须提供 `extension_type`。完整枚举和示例见 [客户端数据埋点上报](tracking-events-api.md)。

### Apple 支付与服务端通知

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | [`/api/payments/apple/pay`](apple-payment-api.md) | 校验 StoreKit 交易并发放商品 |
| POST | `/api/payments/apple/notification` | 接收并验签 Apple V2 通知，幂等处理退款、续费和订阅状态变化；该公开 Webhook 不由客户端调用 |

### 阿里云 OSS 服务端签名直传

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/uploads/oss/signature` | 获取短时效 OSS V4 预签名 PUT 地址；需要 Bearer Token |
| GET | `/api/uploads` | 分页查询当前登录用户的上传文件记录；`file_url` 返回半链接 |

请求中的 `media_type`、文件扩展名、MIME 和精确字节数会经过校验。响应中的 `upload_url` 与 `headers` 用于将文件原始字节直接 `PUT` 到 OSS，详细调用流程和 Bucket CORS 要求见 [阿里云 OSS 服务端签名直传 API](aliyun-oss-direct-upload-api.md)。

每个接口完整的 Header、路径参数、Query、JSON/Form 参数、响应参数和响应示例，以在线文档或静态 [openapi.json](openapi.json) 为准。
