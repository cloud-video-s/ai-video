# API 接口文档

服务启动后会根据 Gin 当前实际注册的路由生成 OpenAPI 3 文档，无需手工维护路由清单。

- API 文档界面：`http://localhost:8080/docs`
- OpenAPI JSON：`http://localhost:8080/docs/openapi.json`

文档仅展示客户端 `/api` 接口，不包含任何 `/admin` 后台管理接口。客户端鉴权使用 Bearer JWT，在文档界面点击 **Authorize**，输入登录接口返回的 token 即可调试受保护接口。

文档会自动覆盖当前注册的 `/api` 路由。已登记 DTO 的接口会展示具体请求字段、必填项和枚举；其他新客户端路由仍会自动出现在文档中，并提供通用 JSON 请求体，后续可在 `internal/apidoc/document.go` 的 `endpointTypes` 中补充 DTO 映射。

## 客户端错误提示语言

`/api` 接口的常规错误会返回统一且不暴露内部细节的多语言提示。服务会按以下顺序确定语言：

1. 使用请求头 `Video_App_Package` 和 `Video_App_Version` 查找“安装包管理”中该版本配置的接口语言。
2. 包版本没有配置或未找到时，读取标准请求头 `Accept-Language`。
3. 仍无法识别时使用简体中文 `zh-CN`。

当前支持 `zh-CN`、`en-US`、`ja-JP`、`ko-KR`、`es-ES`，响应头 `Content-Language` 会返回最终使用的语言。
