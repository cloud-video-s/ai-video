# APP 基础信息配置

后台“系统配置”中的 `APP 基础信息` 分组提供以下内置配置：

| 配置键 | 说明 | 类型 |
| --- | --- | --- |
| `app.name` | 应用名称 | 字符串 |
| `app.about` | 关于我们内容 | 多行文本 |
| `app.customer_service_phone` | 客服电话 | 字符串 |
| `app.customer_service_email` | 客服邮箱 | 邮箱 |
| `app.website` | 官方网站 | HTTP/HTTPS 地址 |
| `app.theme_color` | APP 主题皮肤颜色 | `#RRGGBB` 颜色 |
| `app.theme_mode` | 跟随系统、浅色或深色 | 下拉选择 |
| `app.language` | APP 默认语言 | 下拉选择 |

这些配置均为公开配置，APP 可通过以下接口读取：

```http
GET /api/configs/public
```

响应中的 `data` 是配置键到字符串值的映射。后台保存后会同步刷新配置缓存，新请求立即生效。
