import request from '@/utils/request'

export interface DisplayPosition {
  id: number
  position_name: string
  position_key: string
  description: string
  cover_image: string
  sort: number
  status: number
  created_at: string
  updated_at: string
}

export type DisplayPositionPayload = Omit<DisplayPosition, 'id' | 'created_at' | 'updated_at'>

export function getDisplayPositionList(params: Record<string, unknown>) {
  return request.get('/admin/display-positions', { params })
}

export function getDisplayPositionOptions() {
  return request.get('/admin/display-positions/options')
}

export function getDisplayPosition(id: number) {
  return request.get(`/admin/display-positions/${id}`)
}

export function createDisplayPosition(data: DisplayPositionPayload) {
  return request.post('/admin/display-positions', data)
}

export function updateDisplayPosition(id: number, data: DisplayPositionPayload) {
  return request.put(`/admin/display-positions/${id}`, data)
}

export function deleteDisplayPosition(id: number) {
  return request.delete(`/admin/display-positions/${id}`)
}
