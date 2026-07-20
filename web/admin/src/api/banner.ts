import request from '@/utils/request'
import type { Country } from '@/api/country'
import type { Channel } from '@/api/channel'
import type { AppPackage } from '@/api/package'
import type { VideoTemplate } from '@/api/template'
import type { DisplayPosition } from '@/api/displayPosition'

export type BannerJumpType = 1 | 2 | 3 | 4

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
  template?: VideoTemplate | null
  countries: Country[]
  channels: Channel[]
  packages: AppPackage[]
  created_at: string
  updated_at: string
}

export interface VideoBannerPayload {
  name: string
  cover_image: string
  display_position_keys: string[]
  country_ids: number[]
  channel_ids: number[]
  package_ids: number[]
  remark: string
  sort: number
  jump_type: BannerJumpType
  jump_url: string
  template_id: number | null
  status: number
}

export function getBannerList(params: Record<string, unknown>) {
  return request.get('/admin/banners', { params })
}

export function getBanner(id: number) {
  return request.get(`/admin/banners/${id}`)
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
