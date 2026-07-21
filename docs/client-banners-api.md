# APP Banner 查询接口

## 接口

`GET /api/banners/list`

该接口必须在请求头携带登录后获得的令牌：

```http
Authorization: Bearer <token>
```

只返回 `status = 1` 的 Banner，并按 `sort ASC, id DESC` 排序。

## 查询参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `position_key` | 是 | 展示位置标识，如 `home_banner`；只返回该启用位置下的 Banner |
| `country` | 否 | APP 所在国家二字码，如 `CN`；未传时依次使用用户的设备国家、IP 国家 |
| `channel` | 否 | 渠道唯一标识或渠道 ID；未传时使用登录用户的 `channel_id` |
| `channel_package` | 否 | 渠道包标识，对应渠道的 `delivery_package` |
| `package_code` / `app_package` | 否 | APP 包唯一标识，两个参数作用相同；未传时使用用户的 `app_name` |
| `package_version` / `app_package_version` | 否 | APP 包版本，两个参数作用相同 |

查询参数未传时，也支持从 `Video_Device_Country`、`Video_Channel_ID`、`Video_Channel_Package`、`Video_App_Package`、`Video_App_Version` 请求头读取当前 APP 环境。查询参数优先于请求头，请求头优先于登录用户保存的信息。

同一投放维度未配置关联数据时，Banner 在该维度为全局可见；配置了关联数据时，客户端必须命中。国家、渠道和 APP 包三个维度之间是“并且”关系。无法识别客户端某一维度时，只返回该维度为全局的 Banner。

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
