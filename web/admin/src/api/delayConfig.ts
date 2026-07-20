import request from '@/utils/request'

export interface DelayConfig {
  id: number
  group: string
  key: string
  value: string
  type: 'string' | 'int' | 'bool'
  options: string
  remark: string
  sort: number
  created_at: string
  updated_at: string
}

export interface DelayConfigPayload {
  group: string
  key?: string
  value: string
  type: DelayConfig['type']
  options: string
  remark: string
  sort: number
}

export function getDelayConfigList(params: { page: number; page_size: number; group?: string; keyword?: string }) {
  return request.get('/admin/delay-configs', { params })
}

export function getDelayConfigGroups() {
  return request.get('/admin/delay-configs/groups')
}

export function createDelayConfig(data: DelayConfigPayload & { key: string }) {
  return request.post('/admin/delay-configs', data)
}

export function updateDelayConfig(id: number, data: DelayConfigPayload) {
  return request.put(`/admin/delay-configs/${id}`, data)
}

export function batchUpdateDelayConfigValues(items: { key: string; value: string }[]) {
  return request.put('/admin/delay-configs/values', { items })
}

export function deleteDelayConfig(id: number) {
  return request.delete(`/admin/delay-configs/${id}`)
}

export function syncDelayConfigs() {
  return request.post('/admin/delay-configs/sync')
}
