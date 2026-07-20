import request from '@/utils/request'

export type MediaKind = 'image' | 'video'

export interface UploadSession {
  upload_id: string
  kind: MediaKind
  original_name: string
  extension: string
  content_type?: string
  total_size: number
  chunk_size: number
  total_chunks: number
  uploaded_chunks: number[]
  completed: boolean
  file_path?: string
  file_url?: string
  created_at: string
  expires_at: string
}

function mediaPath(kind: MediaKind) {
  return kind === 'image' ? 'images' : 'videos'
}

export function initiateUpload(kind: MediaKind, file: File) {
  return request.post(`/admin/uploads/${mediaPath(kind)}/batches`, {
    files: [{ file_name: file.name, size: file.size, content_type: file.type }],
  })
}

export function getUploadStatus(kind: MediaKind, uploadID: string) {
  return request.get(`/admin/uploads/${mediaPath(kind)}/${uploadID}`, { silentError: true } as any)
}

export function uploadChunk(
  kind: MediaKind,
  uploadID: string,
  index: number,
  chunk: Blob,
  checksum: string,
  signal: AbortSignal,
  onProgress: (loaded: number) => void,
) {
  return request.put(`/admin/uploads/${mediaPath(kind)}/${uploadID}/chunks/${index}`, chunk, {
    headers: {
      'Content-Type': 'application/octet-stream',
      'X-Chunk-SHA256': checksum,
    },
    timeout: 0,
    signal,
    onUploadProgress: (event) => onProgress(event.loaded),
  })
}

export function completeUpload(kind: MediaKind, uploadID: string) {
  return request.post(`/admin/uploads/${mediaPath(kind)}/${uploadID}/complete`, undefined, { timeout: 0 })
}

async function blobSHA256(blob: Blob) {
  if (!globalThis.crypto?.subtle) return ''
  const digest = await crypto.subtle.digest('SHA-256', await blob.arrayBuffer())
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, '0')).join('')
}

// Compatibility helper for simple image pickers. Template media uses the
// resumable MediaUploader component, while this helper still uploads in chunks.
export async function uploadImage(file: File, onProgress?: (percentage: number) => void): Promise<string> {
  const response: any = await initiateUpload('image', file)
  const session = response.data.uploads?.[0] as UploadSession | undefined
  if (!session) throw new Error('图片上传初始化失败')

  let uploadedBytes = 0
  for (let index = 0; index < session.total_chunks; index++) {
    const start = index * session.chunk_size
    const chunk = file.slice(start, Math.min(start + session.chunk_size, file.size))
    const checksum = await blobSHA256(chunk)
    const controller = new AbortController()
    await uploadChunk('image', session.upload_id, index, chunk, checksum, controller.signal, (loaded) => {
      onProgress?.(Math.min(99, Math.round((uploadedBytes + loaded) * 100 / file.size)))
    })
    uploadedBytes += chunk.size
    onProgress?.(Math.min(99, Math.round(uploadedBytes * 100 / file.size)))
  }

  const completedResponse: any = await completeUpload('image', session.upload_id)
  const completed = completedResponse.data as UploadSession
  onProgress?.(100)
  return completed.file_url || (completed.file_path ? `/uploads/${completed.file_path}` : '')
}
