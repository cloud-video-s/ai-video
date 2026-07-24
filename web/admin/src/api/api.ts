import request from '@/utils/request'

export interface AdminAPI {
  id: number
  path: string
  method: string
  group: string
  description: string
  created_at: string
  updated_at: string
}

export interface AdminAPIPayload {
  path: string
  method: string
  group: string
  description: string
}

export function getAPIList(params: Record<string, unknown>) {
  return request.get('/admin/apis', { params })
}

export function createAPI(data: AdminAPIPayload) {
  return request.post('/admin/apis', data)
}

export function updateAPI(id: number, data: AdminAPIPayload) {
  return request.put(`/admin/apis/${id}`, data)
}

export function deleteAPI(id: number) {
  return request.delete(`/admin/apis/${id}`)
}
