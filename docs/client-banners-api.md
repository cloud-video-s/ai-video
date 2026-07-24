# APP Banner 查询接口

## 接口

`GET /api/banners/list`

该接口必须在请求头携带登录后获得的令牌：

```http
Authorization: Bearer <token>
```

只返回 `status = 1` 且符合当前客户端投放范围的 Banner，并按 `sort ASC, id DESC` 排序。

## 查询参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `position_key` | 是 | 展示位置标识，如 `home_banner` |

除 `Authorization` 外，接口使用以下公共请求头识别客户端环境：

| Header | 必填 | 用途 |
| --- | ---: | --- |
| `Video_App_Code` | 是 | 匹配 Banner 关联的应用 `app_code` |
| `Video_App_Package_Code` | 是 | 匹配 Banner 关联的应用包 `package_code` |
| `Video_App_Version` | 是 | 匹配 Banner 关联的包版本 `version_code` |
| `Video_Phone_Model` | 是 | 公共客户端上下文；当前不参与 Banner 定向 |
| `Video_Channel_Code` | 是 | 公共客户端上下文；当前不参与 Banner 定向 |
| `Video_Device_Country` | 否 | ISO 3166-1 alpha-2 国家或地区代码；未传时依次使用登录用户国家和 IP 推断结果 |

会员类型不由客户端参数指定，服务端根据当前登录用户的订阅状态自动判断为会员或非会员。

## 投放匹配规则

Banner 按以下维度依次匹配：展示位置、国家、应用、应用包、版本和会员类型。所有维度之间是“并且”（AND）关系。

- 某个维度没有有效关联记录时，表示该维度选择“全部”，任何客户端值都可命中。
- 某个维度存在关联记录时，客户端必须命中至少一条该维度的记录。
- 管理端选择“全部国家”“全部展示位置”或“全部应用、包和版本”时，不写入对应关联数据。
- 指定应用包但不选择版本时，只绑定应用和包，不绑定版本，表示该包下全部版本。
- 客户端某个环境值无法识别时，只返回该维度未绑定关联数据的 Banner。

当 `jump_type = 2` 时，目标模板也必须处于启用且未删除状态。

## 响应字段

| 字段 | 说明 |
| --- | --- |
| `id` | Banner ID |
| `name` | Banner 名称 |
| `position_key` | 展示位置标识 |
| `status` | 状态，接口固定返回 `1` |
| `jump_type` | 跳转方式：`1` 链接、`2` 模板、`3` 文生图、`4` 文生视频 |
| `cover_image` | 封面图地址 |
| `route` | 前端跳转路由或外部链接 |
| `template_id` | 目标模板 ID，仅模板跳转时返回 |
| `target_template` | 启用的目标模板摘要，仅模板跳转时返回 |
| `sort` | 排序值 |

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 12,
      "name": "首页夏日活动",
      "position_key": "home_banner",
      "status": 1,
      "jump_type": 2,
      "cover_image": "https://cdn.example.com/banners/summer.jpg",
      "route": "/templates/42",
      "template_id": 42,
      "target_template": {
        "id": 42,
        "name": "夏日视频模板",
        "template_type": "action",
        "cover_image": "https://cdn.example.com/templates/42.jpg",
        "template_video": "https://cdn.example.com/templates/42.mp4",
        "thumbnail_video": "https://cdn.example.com/templates/42-thumb.mp4",
        "status": 1
      },
      "sort": 10
    }
  ]
}
```
