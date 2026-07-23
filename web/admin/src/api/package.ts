import request from '@/utils/request'

export interface AppPackage {
  id: number
  package_name: string
  package_code: string
  app_code: string
  description: string
  sort: number
  status: number
  system_type: number
  created_at: string
  updated_at: string
}

export type AppPackagePayload = Omit<AppPackage, 'id' | 'created_at' | 'updated_at'>

export function getPackageList(params: Record<string, unknown>) {
  return request.get('/admin/packages', { params })
}

export function getPackageOptions() {
  return request.get('/admin/packages/options')
}

export function createPackage(data: AppPackagePayload) {
  return request.post('/admin/packages', data)
}

export function updatePackage(id: number, data: AppPackagePayload) {
  return request.put(`/admin/packages/${id}`, data)
}

export function deletePackage(id: number) {
  return request.delete(`/admin/packages/${id}`)
}
