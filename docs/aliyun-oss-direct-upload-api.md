# 阿里云 OSS 服务端签名直传 API

客户端先向本服务申请短时效 OSS V4 签名，再把文件原始字节直接 `PUT` 到 OSS。文件内容不经过本服务。

## 申请直传签名

`POST /api/uploads/oss/signature`

该接口需要与其他客户端 API 相同的 Bearer Token 和公共请求头。

请求体：

```json
{
  "media_type": "image",
  "file_name": "avatar.png",
  "size": 12345,
  "content_type": "image/png"
}
```

- `media_type`：`image` 或 `video`。
- `file_name`：仅用于校验扩展名；OSS 对象名由服务端随机生成。
- `size`：文件精确字节数，会绑定到签名。
- `content_type`：文件 MIME，会绑定到签名。

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "upload_id": "0123456789abcdef0123456789abcdef",
    "provider": "aliyun_oss",
    "method": "PUT",
    "upload_url": "https://bucket.oss-cn-hangzhou.aliyuncs.com/uploads/images/...?...",
    "headers": {
      "Content-Length": "12345",
      "Content-Type": "image/png",
      "X-Oss-Forbid-Overwrite": "true"
    },
    "object_key": "uploads/images/2026/07/28/随机值.png",
    "file_url": "/uploads/images/2026/07/28/随机值.png",
    "expires_at": "2026-07-28T12:10:00+08:00"
  }
}
```

签名默认有效 600 秒，可通过 `upload.oss.signature_ttl_seconds` 配置为 60–3600 秒。接口仅在 `upload.storage_provider=aliyun_oss` 时可用。

`file_url` 始终返回以 `/` 开头、不包含协议和域名的半链接，供前端业务表单提交和后端入库。
`preview_url` 由后端在半链接前拼接 `upload.proxy_base_url`，默认返回形如
`https://test-cdn.zdrawai.com/uploads/images/example.png` 的完整代理/CDN 预览地址。

## 直接上传到 OSS

原生客户端或命令行必须原样携带响应中的全部签名头：

```bash
curl -X PUT "$UPLOAD_URL" \
  -H "Content-Length: 12345" \
  -H "Content-Type: image/png" \
  -H "x-oss-forbid-overwrite: true" \
  --data-binary @avatar.png
```

浏览器不允许 JavaScript 手动设置 `Content-Length`，使用 `File`/`Blob` 作为请求体时浏览器会自动生成正确值：

```js
const signedHeaders = { ...signature.data.headers }
delete signedHeaders['Content-Length']

const response = await fetch(signature.data.upload_url, {
  method: signature.data.method,
  headers: signedHeaders,
  body: file,
})
if (!response.ok) throw new Error(`OSS upload failed: ${response.status}`)
```

签名成功时服务端会先在 `video_upload` 写入状态 `1`（未完成）的文件记录。上传成功后，业务请求使用签名响应中的半链接 `file_url`；`preview_url` 只用于展示。业务记录创建成功后，对应文件状态更新为 `2`（已完成）。若文件大小、MIME、对象键或禁止覆盖头与签名不一致，OSS 会拒绝请求。

## 服务端上传

管理端分片上传、生成结果下载和封面生成等服务端上传统一写入阿里云 OSS。OSS 返回成功后，服务端在 `video_upload` 中写入状态 `2`（已完成）。上传响应的 `file_url` 和数据库、业务表均使用半链接；`preview_url` 以及模板查询的展示字段使用完整代理/CDN 地址。

## OSS CORS 要求

Web 端直传前需在对应 Bucket 配置 CORS，至少允许业务前端域名使用 `PUT`，并允许 `Content-Type`、`x-oss-forbid-overwrite` 请求头。Bucket CORS 是 OSS 侧配置，本接口不会自动修改。

常见错误：

- `400`：文件类型、MIME、大小或请求参数不符合上传策略。
- `413`：文件超过图片或视频大小上限。
- `503`：当前存储方式不是 `aliyun_oss`，或 OSS/签名有效期配置不完整。
