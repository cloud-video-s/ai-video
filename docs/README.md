# 项目文档导航

## 运行时与接口

- [API 接口文档](API_DOCS.md)：当前客户端接口清单、公共参数、鉴权与语言规则
- [静态 OpenAPI](openapi.json)：根据当前 Gin 路由导出的 OpenAPI 3.0.3 文档
- [Podman 镜像构建与打包](PODMAN.md)：分别构建、导出和运行 Web 与 Go 后端镜像
- [第三方登录](third-party-auth.md)：Google/Apple ID Token 验证、配置与安全边界
- [客户端 Banner API](client-banners-api.md)：Banner 展示接口
- [客户端模板展示 API](client-template-display-api.md)：模板展示配置接口

## 业务配置

- [APP 基础信息配置](app-basic-config.md)：公开的应用名称、主题、客服和语言配置

## 维护规则

- 新增客户端接口时，应确认运行时 OpenAPI 能生成正确的请求 DTO。
- 接口行为、字段或鉴权变化必须与代码在同一提交中更新文档。
- 示例不得包含真实 Token、账号、密钥或生产地址。
- 文档统一使用 UTF-8 编码、LF 换行和相对链接。

## 已知资料问题

部分旧中文文档和源码字符串存在历史编码损坏（乱码）。修复时应根据业务语义或历史版本逐项校对，避免用自动转码批量覆盖后引入错误文案。
