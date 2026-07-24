# 客户端 API 接口文档

本文档对应当前项目实际注册的 `/api` 路由。OpenAPI 版本为 3.0.3，仅包含客户端接口，不包含 `/admin` 后台接口。

## 文档入口

- 在线文档：`http://localhost:8080/docs/ui`
- 在线 OpenAPI：`http://localhost:8080/docs/openapi.json`
- 静态 OpenAPI：[openapi.json](openapi.json)

重新生成静态文档：

```powershell
go run ./cmd/apidocgen -config config/config.yaml -output docs/openapi.json
```

## 鉴权

除健康检查和登录接口外，其他客户端接口均需要在 Header 中携带：

```http
Authorization: Bearer <JWT>
```

JWT 由 `POST /api/auth/login` 返回。`Authorization` 属于鉴权信息，不作为普通请求参数重复列在各接口参数表中。

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

`GET /api/generation/tasks/:id/events` 是 SSE 接口，响应类型为 `text/event-stream`，不使用普通 JSON 包装。

## 当前接口清单

### 公共接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/health` | 健康检查 |
| POST | `/api/auth/login` | 游客登录或创建设备账号 |

### 用户与认证

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/auth/logout` | 退出登录 |
| POST | `/api/third_binding` | 绑定或切换第三方身份 |
| GET | `/api/users/me` | 查询当前用户 |
| PUT | `/api/users/me/country` | 更新用户国家或地区 |
| GET | `/api/users/me/identities` | 查询第三方身份 |
| DELETE | `/api/users/me/identities/:provider` | 解绑第三方身份 |

### 客户端配置与内容

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/ob_delay` | 获取扁平数值格式的延迟配置对象 |
| GET | `/api/configs/list` | 获取客户端公开配置 |
| GET | `/api/banners/list` | 按展示位置、国家、应用、包、版本及会员状态查询 Banner；范围未绑定表示全部 |
| GET | `/api/templates/categories` | 查询模板分类 |
| GET | `/api/templates/recommend` | 查询推荐模板 |
| GET | `/api/templates/list` | 查询分类及其模板 |
| GET | `/api/templates/template_list` | 分页查询分类模板 |
| GET | `/api/templates/template_info` | 查询模板详情 |
| POST | `/api/templates/:id/favorite` | 收藏模板 |
| DELETE | `/api/templates/:id/favorite` | 取消收藏模板 |
| GET | `/api/vip/recommend` | 查询推荐 VIP 套餐 |

### 内容生成

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/generation/models` | 查询可用生成模型和默认参数 |
| POST | `/api/generation/tasks` | 创建生成任务 |
| GET | `/api/generation/tasks` | 分页查询当前用户生成任务 |
| GET | `/api/generation/tasks/:id` | 查询生成任务详情 |
| GET | `/api/generation/tasks/:id/events` | 通过 SSE 订阅任务状态 |
| DELETE | `/api/generation/tasks/:id` | 删除生成任务 |

### Apple 支付

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/payments/apple/confirm` | 校验 StoreKit 交易并发放商品 |

### 分片上传

图片和视频分别使用 `images`、`videos` 路径：

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/uploads/{images|videos}/batches` | 初始化批量上传 |
| PUT | `/api/uploads/{images|videos}/:upload_id/chunks/:index` | 上传分片 |
| GET | `/api/uploads/{images|videos}/:upload_id` | 查询上传进度 |
| POST | `/api/uploads/{images|videos}/:upload_id/complete` | 完成上传 |

每个接口完整的 Header、路径参数、Query、JSON/Form 参数、响应参数和响应示例，以在线文档或静态 [openapi.json](openapi.json) 为准。
