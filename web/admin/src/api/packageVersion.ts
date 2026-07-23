import request from '@/utils/request'

export interface PackageVersion {
  id: number
  package_code: string
  version_code: string
  download_url: string
  install_count: number
  download_count: number
  device_count: number
  description: string
  status: number
  created_at: string
  updated_at: string
}

export type PackageVersionPayload = Omit<PackageVersion, 'id' | 'created_at' | 'updated_at'>

export function getPackageVersionList(params: Record<string, unknown>) {
  return request.get('/admin/package-versions', { params })
}

export function createPackageVersion(data: PackageVersionPayload) {
  return request.post('/admin/package-versions', data)
}

export function updatePackageVersion(id: number, data: PackageVersionPayload) {
  return request.put(`/admin/package-versions/${id}`, data)
}

export function deletePackageVersion(id: number) {
  return request.delete(`/admin/package-versions/${id}`)
}
