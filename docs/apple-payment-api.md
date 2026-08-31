# Apple 支付确认 API

## 接口说明

`POST /api/payments/apple/pay`

客户端完成 StoreKit 购买后调用此接口。服务端校验 Apple 交易凭证、匹配当前应用包下的商品、创建并支付订单，然后发放对应权益。相同 `transactionID` 的重复请求会按同一笔 Apple 交易幂等处理。

## 请求头

```http
Content-Type: application/json
Authorization: Bearer <JWT>
Video_App_Code: <应用代码>
Video_App_Package_Code: com.dola.ai.video.generator
Video_App_Version: <应用版本>
Video_Phone_Model: <设备型号>
Video_Channel_Code: <渠道代码>
```

`Video_App_Package_Code`、请求体 `bundleID` 和 `signedTransactionInfo` 中的 `bundleId` 必须一致。其他可选公共请求头见 [客户端 API 接口文档](API_DOCS.md#api-公共请求参数)。

## 请求参数

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `shop_type` | integer | 是 | 商品类型：`1`=VIP 订阅，`2`=积分商品 |
| `bundleID` | string | 是 | Apple Bundle ID，最长 191 个字符 |
| `expirationDate` | string / null | 否 | 客户端上报的订阅到期时间，RFC 3339；存在时必须与交易凭证一致，允许最多 2 秒误差 |
| `isActive` | boolean | 否 | 客户端上报状态。服务端不信任该值，也不直接用它生成响应状态 |
| `originalTransactionID` | string | 是 | Apple 原始交易 ID，最长 191 个字符，必须与交易凭证一致 |
| `productID` | string | 是 | Apple 商品 ID，最长 191 个字符，必须与交易凭证及服务端商品配置一致 |
| `purchaseDate` | string | 是 | 客户端上报的购买时间，RFC 3339；必须与交易凭证一致，允许最多 2 秒误差 |
| `revocationDate` | string / null | 否 | 撤销时间，RFC 3339；未撤销时传 `null`，存在时必须与交易凭证一致 |
| `signedTransactionInfo` | string | 否 | App Store 签发的三段式 compact JWS。为空或不是三段式时，服务端使用 `transactionID` 调用 App Store Server API 获取已签名交易；客户端原值不会作为支付凭证使用 |
| `source` | string | 否 | 购买入口来源，最长 64 个字符，例如 `directPurchase` |
| `transactionID` | string | 是 | Apple 交易 ID，最长 191 个字符，必须与交易凭证一致 |

### 请求示例

```json
{
  "shop_type": 1,
  "bundleID": "com.dola.ai.video.generator",
  "expirationDate": "2026-07-22T08:47:39.000Z",
  "isActive": true,
  "originalTransactionID": "2000001209105682",
  "productID": "dolaai18",
  "purchaseDate": "2026-07-22T08:42:39.000Z",
  "revocationDate": null,
  "signedTransactionInfo": "<Apple compact JWS: header.payload.signature>",
  "source": "directPurchase",
  "transactionID": "2000001209105682"
}
```

标准的 `signedTransactionInfo` 是 Apple 签发的 `<header>.<payload>.<signature>`。客户端传入标准 JWS 时服务端直接完成证书链验签；字段为空或传入五段等非标准格式时，服务端不会尝试信任或解密该内容，而是用 `transactionID` 调用 Apple `Get Transaction Info`，并验签 Apple 返回的三段式 `signedTransactionInfo`。

### App Store Server API 配置

五段凭证回退查询需要以下服务端配置：

```yaml
app_store:
  bundle_id: "<App Bundle ID>"
  issuer_id: "<App Store Connect Issuer ID>"
  key_id: "<In-App Purchase Key ID>"
  private_key_path: config/appkey.p8
  http_timeout_ms: 10000
```

生产环境可使用环境变量 `APP_STORE_BUNDLE_ID`、`APP_STORE_ISSUER_ID`、`APP_STORE_KEY_ID` 和 `APP_STORE_PRIVATE_KEY_PATH`。支付请求的 Bundle ID 必须与服务端 `app_store.bundle_id` 一致。`.p8` 只用于生成访问 App Store Server API 的 ES256 JWT，不用于验证 Apple 回调；回调及交易 JWS 始终使用 Apple x5c 证书链和 Apple Root CA 验签。

## 成功响应

HTTP 状态码：`200`

| 参数 | 类型 | 说明 |
|---|---|---|
| `code` | integer | 业务状态码，成功固定为 `0` |
| `message` | string | 成功固定为 `success` |
| `data.order_no` | string | 服务端订单号 |
| `data.status` | integer | 订单状态；权益已发放时为 `4`（订单完成） |
| `data.product_type` | integer | 商品类型：`1`=VIP 订阅，`2`=积分商品 |
| `data.product_id` | integer | 服务端内部商品记录 ID |
| `data.product_code` | string | Apple 商品 ID |
| `data.transaction_id` | string | Apple 交易 ID |
| `data.original_transaction_id` | string | Apple 原始交易 ID |
| `data.currency` | string | ISO 4217 货币代码 |
| `data.paid_amount` | number | 实付金额；由凭证中的 `price / 1000` 得出并保留两位小数 |
| `data.purchase_date` | string | 已验签的购买时间，RFC 3339 |
| `data.expiration_date` | string | 已验签的订阅到期时间，RFC 3339；无到期时间时省略 |
| `data.is_active` | boolean | 服务端当前有效状态，计算规则见下文 |
| `data.environment` | string | App Store 环境，例如 `Sandbox` 或 `Production` |
| `data.evidence_mode` | string | 凭证模式，固定为 `jws` |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "order_no": "20260728090907cc7d7c1ffd15",
    "status": 4,
    "product_type": 1,
    "product_id": 1,
    "product_code": "dolaai18",
    "transaction_id": "2000001209105682",
    "original_transaction_id": "2000001209105682",
    "currency": "USD",
    "paid_amount": 19.99,
    "purchase_date": "2026-07-22T16:42:39+08:00",
    "expiration_date": "2026-07-22T16:47:39+08:00",
    "is_active": false,
    "environment": "Sandbox",
    "evidence_mode": "jws"
  }
}
```

## `is_active` 计算规则

请求字段 `isActive` 与响应字段 `is_active` 含义不同。响应值由服务端依据已验签的交易数据计算：

```text
is_active = 未撤销 AND（非订阅 OR 到期时间晚于服务端当前时间）
```

因此，请求中即使传入 `"isActive": true`，当凭证中的订阅到期时间已经早于服务端当前时间时，响应仍会返回 `"is_active": false`。示例交易在 `2026-07-22T16:47:39+08:00` 到期，而订单响应产生于 2026-07-28，所以返回 `false`。已正确验签的历史交易仍可完成订单确认，但不会把已过期权益恢复为有效状态。

## 错误响应

| HTTP 状态码 | `code` | 场景 |
|---:|---:|---|
| 401 | 401 | 缺少公共请求头、JWT 无效或登录状态失效 |
| 200 | 10001 | JSON/字段校验失败、凭证无效、签名失败、Bundle ID 不一致、商品未配置、交易已撤销或交易 ID 已被其他用户/商品使用 |
| 200 | 10000 | 创建订单、查询商品或发放权益时发生服务端错误 |

错误示例：

```json
{
  "code": 10001,
  "message": "invalid Apple transaction evidence",
  "data": null
}
```

## 校验与幂等规则

- `transactionID`、`originalTransactionID`、`productID`、`bundleID` 和关键时间必须与已验签交易一致。
- 标准三段式 JWS 会直接验签；空值、五段式或其他非标准客户端凭证只作为触发 App Store Server API 查询的信号，其内容不会被信任或持久化。
- 已撤销交易不会创建或支付订单。
- 同一 Apple `transactionID` 重复提交时，同一用户和商品返回原订单；被其他用户或其他商品占用时返回参数错误。
- 商品必须已配置在当前 `Video_App_Package_Code` 下。

# App Store Server Notifications V2 回调 API

## 接口说明

`POST /api/payments/apple/notification`

该地址配置到 App Store Connect 的版本 2 通知 URL。接口由 Apple 服务器直接调用，是公开端点，不需要 `Authorization` 或客户端公共请求头。请求体上限为 1 MiB。

旧地址 `POST /api/apy` 暂时保留为兼容别名，并进入完全相同的 V2 验签和业务处理流程；新的 App Store Connect 配置应使用上面的正式地址。

## 请求参数

```json
{
  "signedPayload": "<完整的 Apple compact JWS>"
}
```

`signedPayload` 必须是完整的 `<header>.<payload>.<signature>`，不能使用省略号或被日志系统截断的内容。外层已验签载荷支持 Apple 当前定义的互斥对象：`data`、`summary`、`externalPurchaseToken` 和 `appData`。

## 验证规则

- 外层 `signedPayload` 必须使用 `ES256`，`x5c` 必须是 Apple 的三段证书链。
- 证书链固定信任 Apple Root CA - G3，并校验 Apple 收据签名叶证书 OID `1.2.840.113635.100.6.11.1` 和 WWDR 中间证书 OID `1.2.840.113635.100.6.2.1`。
- `data.signedTransactionInfo`、`data.signedRenewalInfo` 和 `appData.signedAppTransactionInfo` 存在时会分别再次验签；内层 `bundleId`、`environment` 必须与外层通知一致。
- 外层 `bundleId` 必须对应服务端已有的 iOS 安装包。已禁用但仍存在的历史安装包仍允许处理退款、撤销和过期通知。
- Production 通知必须包含有效的 `appAppleId`。
- 服务端只记录验签后的通知元数据，不记录完整 JWS，避免将证书和交易载荷写入请求日志。

## 事件处理

| 通知类型 | 当前处理 |
|---|---|
| `REFUND`、`REVOKE` | 幂等撤销已发放权益，并将已支付订单标记为退款 |
| `REFUND_REVERSED` | 不再错误地执行撤权；确认接收并进入权益恢复对账流程 |
| `SUBSCRIBED` | 完成已由客户端创建的首订订单；`RESUBSCRIBE` 可根据原始交易链创建新订单。首订回调先到且尚无法关联用户时，等待已鉴权的客户端确认接口 |
| `DID_RENEW` | 每个新的 Apple `transactionId` 创建一张续期订单并完成支付、积分和权益发放；交易号与订单状态共同保证重复回调不重复发放 |
| `RENEWAL_EXTENDED` | 只延长 Apple 签名到期时间，不创建收费订单，也不增加支付次数 |
| `DID_CHANGE_RENEWAL_STATUS` | `AUTO_RENEW_DISABLED` 标记为已取消自动续订，但保留当前 VIP 到期时间；重新开启时仅在权益仍有效的情况下恢复订阅中状态 |
| `DID_FAIL_TO_RENEW` | 宽限期或尚未到期时保留现有权益；确认已到期后才执行过期处理，不取消已支付订单 |
| `EXPIRED`、`GRACE_PERIOD_EXPIRED` | 仅在不存在更晚本地权益时标记为已过期并收回 VIP 状态 |
| `ONE_TIME_CHARGE` | 对已关联本地订单的积分商品交易完成支付确认、积分入账和订单完结；重复通知不会重复发放。尚未关联用户/订单时等待客户端鉴权确认完成关联 |
| `TEST` | 验签后确认接收，无业务副作用 |
| 其他当前通知类型 | 验签、记录摘要并确认接收，不执行未定义的权益变更 |

`CONSUMPTION_REQUEST` 需要配合 App Store Server API 上报消费信息；当前回调只确认接收，尚未自动提交消费信息。`REFUND_REVERSED` 的自动权益恢复也需要单独的可审计对账流程。

## 响应

验签和业务处理成功时返回 HTTP `200`：

```json
{
  "code": 0,
  "message": "acknowledged",
  "data": {
    "notification_type": "DID_RENEW",
    "notification_uuid": "f8fdbf4f-0000-0000-0000-000000000000",
    "bundle_id": "com.dola.ai.video.generator",
    "environment": "Sandbox",
    "transaction_id": "2000001234567890",
    "version": "2.0",
    "processed": true,
    "action": "renew"
  }
}
```

| HTTP 状态码 | 场景 |
|---:|---|
| `200` | 验签成功且通知已处理或安全确认接收 |
| `400` | JSON 无效、JWS/证书链验签失败、Bundle ID 或环境不匹配 |
| `500` | 数据库查询或业务事务失败；返回非 2xx 以便 Apple 重试 |
