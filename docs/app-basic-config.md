# APP 基础信息配置

后台“系统配置”中的 `APP 基础信息` 分组提供以下内置配置：

| 配置键 | 说明 | 类型 |
| --- | --- | --- |
| `app.name` | 应用名称 | 字符串 |
| `app.about` | 关于我们内容 | 多行文本 |
| `app.customer_service_phone` | 客服电话 | 字符串 |
| `app.customer_service_email` | 客服邮箱 | 邮箱 |
| `app.website` | 官方网站 | HTTP/HTTPS 地址 |
| `app.privacy_policy` | 隐私政策（privacy-policy）文件 | 文件 URL |
| `app.terms` | 用户协议（terms）文件 | 文件 URL |
| `app.faq` | 常见问题（faq）文件 | 文件 URL |
| `app.theme_color` | APP 主题皮肤颜色 | `#RRGGBB` 颜色 |
| `app.theme_mode` | 跟随系统、浅色或深色 | 下拉选择 |
| `app.language` | APP 默认语言 | 下拉选择 |

这些配置均为公开配置，APP 可通过以下接口读取：

```http
GET /api/configs/list
```

响应中的 `data` 是配置键到字符串值的映射。后台保存后会同步刷新配置缓存，新请求立即生效。

三个协议配置在后台使用文件上传控件，也可以手动填写 HTTP/HTTPS 地址。上传入口为：

```http
POST /admin/uploads/config-files
Content-Type: multipart/form-data

config_key=app.privacy_policy
file=@privacy-policy.html
```

支持 `.txt`、`.html`、`.htm`、`.md`、`.json`、`.xml`、`.pdf`，单文件最大 5 MB。服务端会同时校验扩展名和文件实际内容。上传成功只回填文件地址，管理员仍需点击系统配置页的“保存”按钮才会发布。公开配置接口会将 `file` 类型的站内文件地址展开为可访问的完整 URL。
