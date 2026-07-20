import request from '@/utils/request'
import type { AppPackage } from '@/api/package'
import type { Channel } from '@/api/channel'

export interface PointsPackage {
  id: number
  product_id: string
  name: string
  package_id: number
  systems: string[]
  user_types: number[]
  resource_type: string
  points: number
  currency: string
  sale_price: number
  actual_revenue: number
  original_price: number
  badge_text: string
  description: string
  button_text: string
  is_default: boolean
  status: number
  sort: number
  package?: AppPackage
  channels: Channel[]
  created_at: string
  updated_at: string
}

export interface PointsPackagePayload {
  product_id: string
  name: string
  package_id: number
  systems: string[]
  user_types: number[]
  channel_ids: number[]
  resource_type: string
  points: number
  currency: string
  sale_price: number
  actual_revenue: number
  original_price: number
  badge_text: string
  description: string
  button_text: string
  is_default: boolean
  status: number
  sort: number
}

export function getPointsPackageList(params: Record<string, unknown>) {
  return request.get('/admin/points-packages', { params })
}

export function getPointsPackage(id: number) {
  return request.get(`/admin/points-packages/${id}`)
}

export function getPointsPackageOptions() {
  return request.get('/admin/points-packages/options')
}

export function createPointsPackage(data: PointsPackagePayload) {
  return request.post('/admin/points-packages', data)
}

export function updatePointsPackage(id: number, data: PointsPackagePayload) {
  return request.put(`/admin/points-packages/${id}`, data)
}

export function deletePointsPackage(id: number) {
  return request.delete(`/admin/points-packages/${id}`)
}

export function updatePointsPackageStatus(id: number, status: number) {
  return request.patch(`/admin/points-packages/${id}/status`, { status })
}

export function setDefaultPointsPackage(id: number) {
  return request.patch(`/admin/points-packages/${id}/default`)
}
