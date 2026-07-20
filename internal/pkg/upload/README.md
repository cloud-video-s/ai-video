# Chunked image and video uploads

Both entry points expose the same protocol and require a Bearer token:

- Client API: `/api/uploads`
- Admin API: `/admin/uploads` (permission: `system:upload`)
- Completed upload records: `GET /admin/uploads` (all users) and
  `GET /api/uploads` (current API user only).

Images and videos use separate routes, validation policies, and storage directories.

## 1. Initiate a batch

`POST {prefix}/images/batches` or `POST {prefix}/videos/batches`

```json
{
  "files": [
    {
      "file_name": "example.png",
      "size": 123456,
      "content_type": "image/png",
      "sha256": "optional full-file sha256"
    }
  ]
}
```

The response contains one upload session per file, including `upload_id`, `chunk_size`, and `total_chunks`.

## 2. Upload chunks

`PUT {prefix}/{images|videos}/{upload_id}/chunks/{index}`

Send the raw chunk bytes as the request body. Chunk indexes start at `0`. Every chunk except the last must contain exactly `chunk_size` bytes. The optional `X-Chunk-SHA256` header validates an individual chunk. Re-uploading an existing valid chunk is idempotent.

## 3. Resume or inspect progress

`GET {prefix}/{images|videos}/{upload_id}`

Use `uploaded_chunks` to skip chunks already stored by the server. Sessions are persisted on disk, so progress survives process restarts until `expires_at`.

## 4. Complete the upload

`POST {prefix}/{images|videos}/{upload_id}/complete`

The server requires every chunk, merges them in order, validates total size, optional full-file SHA-256, extension, MIME type, and file signature, then returns `file_path` and the computed `sha256`.

Runtime limits are configured under `upload` in `config/config.yaml`. Defaults are 20 MB per image, 2 GB per video, 20 files per batch, 5 MB chunks, and a 24-hour resumable session.
The active final storage provider is selected in Admin -> System Config with
`upload.storage_provider` (`local` or `aliyun_oss`). Provider settings are
resolved when a file is completed, so switching providers does not require a
service restart. OSS AccessKey values are masked by the admin config API.
