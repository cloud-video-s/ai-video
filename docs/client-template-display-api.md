# 按展示位置查询推荐模板

## 接口

`GET /api/templates/recommend`

需要客户端用户 Bearer Token 和公共请求 Header。接口按展示位置、国家、应用、应用包和版本投放规则返回当前客户端可见的推荐模板。

## 查询参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `position_key` | 是 | 展示位置标识，例如 `home_hot` |

国家、应用、应用包及版本等投放上下文从客户端公共请求头读取，与其他模板接口保持一致。

## 返回字段

响应 `data` 是模板对象数组。每个模板对象返回：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | integer | 模板 ID |
| `template_type_id` | integer | 模板分类 ID |
| `name` | string | 模板名称 |
| `template_type` | integer | 模板类型：`1` 图片模板，`2` 视频模板 |
| `cover_image_url` | string | 模板封面图片地址 |
| `original_url` | string | 模板原始媒体地址 |
| `thumbnail_url` | string | 模板缩略媒体地址 |
| `prompt` | string | 模板提示词 |
| `description` | string | 模板说明 |
| `sort` | integer | 排序值 |
| `usage_count` | integer | 使用次数 |
| `favorite_count` | integer | 收藏次数 |
| `view_count` | integer | 浏览次数 |
| `is_favorite` | integer | 当前用户是否已收藏：`0` 否，`1` 是 |
| `model_score` | integer | 生成模型评分 |

当前接口不再返回旧字段 `video_template_type_id`、`cover_image`、`template_video`、`thumbnail_video`、`display_config_id`、`display_sort`、`model_id`、`model_code`、`model_name` 或 `model_parameters`。展示位置仅作为请求筛选条件，不会重复出现在响应对象中。

```json
{
  "code": 0,
  "message": "success",
  "data": [
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
```
