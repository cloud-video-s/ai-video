# Google / Apple 第三方账号绑定

当前客户端只注册一个第三方账号入口：

```http
POST /api/third_binding
```

该接口需要当前有效的 Bearer Token，以及 [客户端 API 公共请求参数](API_DOCS.md#api-公共请求参数)。服务端会为当前账号绑定第三方身份，或切换到已经使用同一第三方标识的账号，并返回新的客户端 JWT。

## 请求体

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `third_type` | string | 是 | 第三方平台，只支持 `google` 或 `apple` |
| `id_token` | string | 条件必填 | Google 等平台签发的 ID Token；未传 `third_code` 时，必须与 `identity_token` 至少提供一个 |
| `identity_token` | string | 条件必填 | Apple Identity Token；未传 `third_code` 时，必须与 `id_token` 至少提供一个 |
| `nonce` | string | 否 | 客户端发起第三方认证时使用的随机值，最长 255 个字符；传入后服务端会校验 Token 中的 nonce |
| `third_code` | string | 否 | 第三方平台用户标识，最长 100 个字符；当前实现传入该字段时不会再次验证 ID Token，仅应由受信任客户端使用 |
| `email` | string | 条件必填 | 直接传 `third_code` 且该标识尚无本地账号时必填，最长 50 个字符；Token 验证流程会使用已验证 Token 中的邮箱覆盖该值 |
| `force_new` | boolean | 否 | 预留字段，当前第三方绑定流程未使用 |
| `first_opened_at` | string(date-time) | 否 | 首次打开时间；新绑定且未传时使用服务端当前时间 |
| `last_opened_at` | string(date-time) | 否 | 最近打开时间；新绑定且未传时使用服务端当前时间 |
| `attribution_clicked_at` | string(date-time) | 否 | 归因点击时间 |

推荐使用提供方签发的 Token，不直接提交 `third_code`：

```json
{
  "third_type": "apple",
  "identity_token": "provider-signed-jwt",
  "nonce": "optional-request-nonce"
}
```

Google 示例：

```json
{
  "third_type": "google",
  "id_token": "provider-signed-jwt",
  "nonce": "optional-request-nonce"
}
```

## 处理规则

- `third_code` 为空时，服务端验证 `id_token` / `identity_token`，并从已验证 Token 读取稳定的 `sub` 和邮箱。
- 没有本地账号使用该 `sub` 时，服务端把当前账号升级为对应的 Google 或 Apple 登录账号。
- 已有账号使用该 `sub` 时，服务端切换到该账号；目标账号必须处于启用状态。
- 当前账号已经绑定其他 `third_code` 时，不允许直接覆盖。
- 系统直接使用用户表中的 `third_code`、`email` 和 `login_type`，没有查询、绑定或解绑身份列表的客户端接口。
- 如果启用了单设备登录，签发新 Token 前会递增目标账号的 Token 版本，使旧 Token 失效。

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

## Token 校验

服务端使用缓存的 JWKS 公钥验证提供方签名，并校验 `kid`、签发方、受众、`azp`（存在时）、到期时间、签发时间、`sub`，以及客户端提供的 nonce。服务端不会存储第三方 ID Token、Access Token、Refresh Token 或 Apple Authorization Code。

相关配置位于 `third_party_auth`：

```yaml
third_party_auth:
  google:
    client_ids: [google-client-id.apps.googleusercontent.com]
  apple:
    client_ids: [com.example.app, com.example.web]
```

Google 通常需要配置 Android、iOS 和 Web OAuth Client ID；Apple 需要配置对应 Bundle ID 或 Services ID。

## 常见错误

| HTTP 状态 | `code` | 场景 |
|---:|---:|---|
| 400 | `10001` | 请求体格式错误，`third_type` 缺失，或 Token 与 `third_code` 均未提供 |
| 401 | `30001` | 第三方 Token 无效 |
| 503 | `10000` | 对应第三方验证器未配置 |
| 200 | `10000` | 第三方类型不支持、当前账号已绑定其他身份、目标账号已停用或服务端处理失败 |
