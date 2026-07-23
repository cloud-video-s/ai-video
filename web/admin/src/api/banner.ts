import request from '@/utils/request'
import type { Country } from '@/api/country'
import type { VideoTemplate } from '@/api/template'
import type { DisplayPosition } from '@/api/displayPosition'

export type BannerJumpType = 1 | 2 | 3 | 4

export interface BannerAppTarget {
  app_code: string
  app_name: string
  package_code: string
  package_name: string
  version_codes: string[]
}

export interface BannerDeliveryVersion { version_code: string }
export interface BannerDeliveryPackage { package_code: string; package_name: string; versions: BannerDeliveryVersion[] }
export interface BannerDeliveryApp { app_code: string; app_name: string; packages: BannerDeliveryPackage[] }

export interface VideoBanner {
  id: number
  name: string
  cover_image: string
  display_positions: DisplayPosition[]
  remark: string
  sort: number
  jump_type: BannerJumpType
  jump_url: string
  template_id: number | null
  status: number
  subscription_status: number
  template?: VideoTemplate | null
  countries: Country[]
  app_targets: BannerAppTarget[]
  created_at: string
  updated_at: string
}

export interface VideoBannerPayload {
  name: string
  cover_image: string
  display_position_keys: string[]
  country_ids: number[]
  app_targets: Array<Pick<BannerAppTarget, 'app_code' | 'package_code' | 'version_codes'>>
  remark: string
  sort: number
  jump_type: BannerJumpType
  jump_url: string
  template_id: number | null
  status: number
  subscription_status: number
}

export function getBannerList(params: Record<string, unknown>) {
  return request.get('/admin/banners', { params })
}

export function getBanner(id: number) {
  return request.get(`/admin/banners/${id}`)
}

export function getBannerDeliveryOptions() {
  return request.get('/admin/banners/delivery-options')
}

export function createBanner(data: VideoBannerPayload) {
  return request.post('/admin/banners', data)
}

export function updateBanner(id: number, data: VideoBannerPayload) {
  return request.put(`/admin/banners/${id}`, data)
}

export function deleteBanner(id: number) {
  return request.delete(`/admin/banners/${id}`)
}
