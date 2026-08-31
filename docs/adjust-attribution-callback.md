# Adjust 归因上报与回调融合

系统使用 Adjust `adid` 融合两类数据：APP SDK 当前归因快照提供已登录用户 ID，Adjust 服务端 callback 提供可信的安装、归因更新及再归因事件。两类数据可按任意顺序到达。

核心规则是“首次获客渠道永不被再归因覆盖”：

- 一个用户可以关联多个 ADID；同一个 ADID 不能绑定不同用户。
- 只有 APP 上报与 callback 融合完成后，首个有效 `install`（兼容旧配置的 `attribution`）才创建 `video_user_attribution` 首次获客快照。
- 用户尚无首次获客快照时，`install_update` 也可以创建；已有快照时只允许同一 ADID 的 `install_update` 纠正。
- 不同 ADID 不得覆盖用户已有的首次获客快照。
- `reattribution`、`reattribution_update`、`reattribution_reinstall` 只更新设备当前归因，不更新 `video_user.channel_id` 或 `video_user_attribution`。
- `session`、`event`、`click`、`impression`、`cost` 和 `rejected_*` 只更新融合行的最后回调元数据，不改变当前归因字段。
- `gdpr_forget_device` 删除该 ADID 的设备级融合数据与 callback payload，保留已经锁定的 `video_user_attribution` 首次获客快照。

## APP 客户端上报

```http
POST /api/attributions/adjust/report
Authorization: Bearer <client-jwt>
Content-Type: application/json
```

APP 应在 Adjust SDK attribution callback 每次触发时幂等上报，不要限制为只上报一次。用户 ID 只能来自 Bearer JWT，JSON 中传入的 `userId` 不参与绑定。

```json
{
  "trackerToken": "22hydf4k",
  "trackerName": "TikTok SAN::campaign::adgroup::creative",
  "campaign": "TT_productpoint_0728",
  "network": "TikTok SAN",
  "creative": "instruction.mp4",
  "adgroup": "TT_productpoint_0728",
  "clickLabel": "",
  "costType": "",
  "costAmount": null,
  "costCurrency": "",
  "fbInstallReferrer": "",
  "googleAdId": "928cdf5a-d453-45a6-8016-115481cbeaa5",
  "adid": "0a09e2a1de95add39162efdf3adff446",
  "idfa": "",
  "idfv": ""
}
```

`costAmount` 推荐发送 JSON number 或 `null`；服务端兼容历史客户端发送的字符串及 `"NaN"`，统一按原始文本保存。`adid` 必填，服务端去除首尾空白并转为小写。

响应状态：

- `1`（pending_app）：callback 已收到，尚未收到登录用户上报。
- `2`（pending_callback）：已绑定用户，尚未收到可处理的安装/再归因 callback。
- `3`（fused）：ADID 与用户已经融合。
- `4`（forgotten）：GDPR 设备数据已经清除。
- `5`（ignored）：事件已记录到融合行，但不参与归因。

## Adjust 服务端 callback

公开地址：

- `GET /api/attributions/adjust/callback`
- `POST /api/attributions/adjust/callback`

callback 不使用客户端 JWT，通过 `callback_token` 参数或 `X-Adjust-Callback-Token` Header 鉴权。推荐配置 GET URL：

```text
https://api.example.com/api/attributions/adjust/callback?callback_token=REDACTED&adid={adid}&app_token={app_token}&tracker_token={tracker}&tracker_name={tracker_name}&network={network_name}&campaign={campaign_name}&adgroup={adgroup_name}&creative={creative_name}&activity_kind={activity_kind}&installed_at={installed_at}&reattributed_at={reattributed_at}&attribution_updated_at={attribution_updated_at}&outdated_tracker={outdated_tracker}&outdated_tracker_name={outdated_tracker_name}&is_redownload={is_redownload}&created_at={created_at}
```

注意：Adjust 官方 tracker token 占位符是 `{tracker}`，因此参数应写成 `tracker_token={tracker}`，不是 `{tracker_token}`。

处理规则：

- token 在任何数据库写入前校验。
- 去除 token 后的规范 JSON 计算 SHA-256 幂等键，并保存到融合行的 `last_callback_key`；最近一次 callback 的连续重试不会重复计数或覆盖。
- callback 非空的 tracker/network/campaign/adgroup/creative 字段覆盖设备当前归因视图，空字段不覆盖已有值。
- `channel_code` 先按配置中的 tracker token、再按 tracker name 映射；映射到 `video_channel.channel_code` 后读取数字 `channel_id`。
- callback 的 `network` 与启用的 `video_media.name` 不区分大小写精确匹配，读取 `adjust_partner_id`；未匹配时保存 `0`。

## 数据表与部署

- `video_adjust_attribution`：APP 上报和 Adjust callback 共用的融合表；每个 ADID 一条设备当前视图，`user_id` 不是唯一键，以支持用户多设备。
- `video_user_attribution`：每个 `user_id` 一条首次获客快照；`adjust_adid` 锁定获客设备。首次创建后只接受同一 ADID 的 `install_update` 纠正。

版本化 SQL：

- `scripts/schema/20260817_001_create_adjust_attribution_callback.sql`
- `scripts/schema/20260818_001_create_adjust_attribution_fusion.sql`
- `scripts/schema/20260818_003_extend_adjust_attribution_lifecycle.sql`
- `scripts/schema/20260819_001_drop_adjust_attribution_callback.sql`

应用代码不会执行 DDL、`AutoMigrate` 或 schema `Migrator`。以上脚本必须单独审核并得到明确执行授权后才能应用；仓库中创建或修改脚本不代表已授权执行。

## 服务配置

```yaml
adjust:
  enabled: true
  # 应用自行定义的回调鉴权值；Adjust 官方没有规定最小长度。
  callback_token: "replace-with-a-random-secret"
  max_body_bytes: 65536
  tracker_channels:
    22hydf4k: tiktok
    TikTok SAN: tiktok
```

tracker token 映射优先于 tracker name 映射。配置的渠道编码必须存在于 `video_channel`；否则安装 callback 返回失败，让 Adjust 按其重试策略再次投递，避免写入无法关联的渠道快照。
