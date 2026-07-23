import request from '@/utils/request'

export interface VideoApp {
  id: number
  name: string
  app_code: string
  status: number
  sort: number
  description: string
  created_at: string
  updated_at: string
}

export type VideoAppPayload = Pick<VideoApp, 'name' | 'app_code' | 'status' | 'sort' | 'description'>

export function getVideoAppList(params: Record<string, unknown>) {
  return request.get('/admin/apps', { params })
}

export function getVideoAppOptions() {
  return request.get('/admin/apps/options')
}

export function createVideoApp(data: VideoAppPayload) {
  return request.post('/admin/apps', data)
}

export function updateVideoApp(id: number, data: VideoAppPayload) {
  return request.put(`/admin/apps/${id}`, data)
}

export function deleteVideoApp(id: number) {
  return request.delete(`/admin/apps/${id}`)
}
