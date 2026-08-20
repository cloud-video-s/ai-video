import request from '@/utils/request'
import type { AppPackage } from '@/api/package'
import type { PackageVersion } from '@/api/packageVersion'
import type { Channel } from '@/api/channel'
import type { Country } from '@/api/country'
import type { VideoApp } from '@/api/videoApp'

export interface Points {
  id: number
  product_code: string
  name: string
  app_codes: string[]
  package_codes: string[]
  version_codes: string[]
  country_codes: string[]
  channel_codes: string[]
  systems: string[]
  user_types: number[]
  resource_type: string
  points: number
  currency: string
  sale_price: number
  actual_revenue: number
  original_price: number
  description: string
  icon: string
  is_default: boolean
  status: number
  sort: number
  apps: VideoApp[]
  packages: AppPackage[]
  package_version: PackageVersion[]
  country: Country[]
  channels: Channel[]
  created_at: string
  updated_at: string
}

export interface PointsPackagePayload {
  product_code: string
  name: string
  app_codes: string[]
  package_codes: string[]
  version_codes: string[]
  country_codes: string[]
  channel_codes: string[]
  systems: string[]
  user_types: number[]
  resource_type: string
  points: number
  currency: string
  sale_price: number
  actual_revenue: number
  original_price: number
  icon: string
  description: string
  button_text: string
  is_default: boolean
  status: number
  sort: number
}

export function getPointsPackageList(params: Record<string, unknown>) {
  return request.get('/admin/points', { params })
}

export function getPointsPackage(id: number) {
  return request.get(`/admin/points/${id}`)
}

export function getPointsPackageOptions() {
  return request.get('/admin/points/options')
}

export function createPointsPackage(data: PointsPackagePayload) {
  return request.post('/admin/points', data)
}

export function updatePointsPackage(id: number, data: PointsPackagePayload) {
  return request.put(`/admin/points/${id}`, data)
}

export function deletePointsPackage(id: number) {
  return request.delete(`/admin/points/${id}`)
}

export function updatePointsPackageStatus(id: number, status: number) {
  return request.patch(`/admin/points/${id}/status`, { status })
}

export function setDefaultPointsPackage(id: number) {
  return request.patch(`/admin/points/${id}/default`)
}
