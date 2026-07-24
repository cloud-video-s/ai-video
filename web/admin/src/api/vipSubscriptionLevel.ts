import request from '@/utils/request'

export interface VIPSubscriptionLevel {
  id: number
  level: string
  description: string
  status: number
  sort: number
  created_at: string
  updated_at: string
}

export type VIPSubscriptionLevelPayload = Pick<VIPSubscriptionLevel, 'level' | 'description' | 'status' | 'sort'>

export function getVIPSubscriptionLevelList(params: Record<string, unknown>) {
  return request.get('/admin/vip-subscription-levels', { params })
}

export function getVIPSubscriptionLevelOptions() {
  return request.get('/admin/vip-subscription-levels/options')
}

export function getVIPSubscriptionLevel(id: number) {
  return request.get(`/admin/vip-subscription-levels/${id}`)
}

export function createVIPSubscriptionLevel(data: VIPSubscriptionLevelPayload) {
  return request.post('/admin/vip-subscription-levels', data)
}

export function updateVIPSubscriptionLevel(id: number, data: VIPSubscriptionLevelPayload) {
  return request.put(`/admin/vip-subscription-levels/${id}`, data)
}

export function updateVIPSubscriptionLevelStatus(id: number, status: number) {
  return request.patch(`/admin/vip-subscription-levels/${id}/status`, { status })
}

export function deleteVIPSubscriptionLevel(id: number) {
  return request.delete(`/admin/vip-subscription-levels/${id}`)
}
