import request from '@/utils/request'

export type GenerationTaskMediaType = 'image' | 'video' | 'unknown'

export interface GenerationTaskUser {
  id: number
  username: string
  email: string
  login_account: string
  imei: string
  device_code: string
}

export interface GenerationTaskModel {
  id: number
  name: string
  code: string
  model_type: number
  version: string
}

export interface UserGenerationTask {
  id: number
  user_id: number
  model_id: number
  template_id: number
  client_request_id: string
  task_code: string
  third_task_code: string
  status: number
  status_name: string
  task_type: number
  progress: number
  media_type: GenerationTaskMediaType
  prompt: string
  request_payload?: unknown
  provider_response?: unknown
  remote_urls: string[]
  local_urls: string[]
  preview_urls: string[]
  cover_image_url: string
  result_count: number
  error_message: string
  usage_duration: number
  score: number
  submitted_at: string | null
  started_at: string | null
  finished_at: string | null
  last_polled_at: string | null
  created_at: string
  updated_at: string
  user: GenerationTaskUser | null
  model: GenerationTaskModel | null
}

export function getUserGenerationTaskList(params: Record<string, unknown>) {
  return request.get('/admin/user-generation-tasks', { params })
}

export function getUserGenerationTask(id: number) {
  return request.get(`/admin/user-generation-tasks/${id}`)
}
