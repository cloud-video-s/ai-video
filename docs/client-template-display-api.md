# 按展示位置查询模板

## 接口

`GET /api/templates/by-position`

需要客户端用户 Bearer Token。接口只返回模板展示配置、展示位置、模板及模板分类均为启用状态，且国家、渠道、安装包、用户类型及订阅状态投放规则匹配当前用户的模板。

## 查询参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `position_key` | 是 | 展示位置标识，例如 `home_hot` |

国家、渠道、安装包及版本等投放上下文从客户端公共请求头读取，与现有模板接口保持一致。

## 返回字段

返回数组中的每一项都是一个模板，并额外包含：

| 字段 | 说明 |
| --- | --- |
| `display_config_id` | 模板展示配置 ID |
| `position_key` | 本次查询的展示位置标识 |
| `display_sort` | 该模板在此展示位置内的排序，数值越大越靠前 |

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 42,
      "video_template_type_id": 3,
      "name": "夏日写真",
      "template_type": "action",
      "cover_image": "https://cdn.example.com/templates/42.jpg",
      "template_video": "https://cdn.example.com/templates/42.mp4",
      "thumbnail_video": "https://cdn.example.com/templates/42-thumb.mp4",
      "display_config_id": 7,
      "position_key": "home_hot",
      "display_sort": 100
    }
  ]
}
```
