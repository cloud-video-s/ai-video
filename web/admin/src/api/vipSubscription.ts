import request from '@/utils/request'
import type { BannerDeliveryApp } from '@/api/banner'
import type { Channel } from '@/api/channel'
import type { Country } from '@/api/country'
import type { AppPackage } from '@/api/package'
import type { PackageVersion } from '@/api/packageVersion'
import type { VideoApp } from '@/api/videoApp'
import type { VIPSubscriptionLevel } from '@/api/vipSubscriptionLevel'

export type VIPSubscriptionPeriod = 1 | 2 | 3 | 4

export interface VIPSubscriptionPayload {
  app_codes: string[]
  package_codes: string[]
  version_codes: string[]
  country_codes: string[]
  channel_codes: string[]
  level_id: number
  vip_type: number
  suk_code: string
  name: string
  currency: string
  first_subscription_price: number
  first_subscription_revenue: number
  first_bonus_points: number
  original_price: number
  vip_duration_days: number
  trial_days: number
  renewal_text: string
  badge_text: string
  agreement_default_checked: boolean
  display_mode: number
  status: number
  free_trial: boolean
  is_subscription: boolean
  is_default: boolean
  subscription_description: string
  subscription_price: number
  subscription_revenue: number
  subscription_points: number
  subscription_period: VIPSubscriptionPeriod
  sort: number
  description: string
  remark: string
}

export interface VIPSubscription extends VIPSubscriptionPayload {
  id: number
  subscription_level: VIPSubscriptionLevel
  apps: VideoApp[]
  packages: AppPackage[]
  package_version: PackageVersion[]
  country: Country[]
  channels: Channel[]
  created_at: string
  updated_at: string
}

export type VIPDeliveryOptions = BannerDeliveryApp[]

export function getVIPSubscriptionList(params: Record<string, unknown>) {
  return request.get('/admin/vip-subscriptions', { params })
}

export function getVIPSubscription(id: number) {
  return request.get(`/admin/vip-subscriptions/${id}`)
}

export function createVIPSubscription(data: VIPSubscriptionPayload) {
  return request.post('/admin/vip-subscriptions', data)
}

export function updateVIPSubscription(id: number, data: VIPSubscriptionPayload) {
  return request.put(`/admin/vip-subscriptions/${id}`, data)
}

export function deleteVIPSubscription(id: number) {
  return request.delete(`/admin/vip-subscriptions/${id}`)
}

export function updateVIPSubscriptionStatus(id: number, status: number) {
  return request.patch(`/admin/vip-subscriptions/${id}/status`, { status })
}

export function updateVIPSubscriptionDisplay(id: number, displayMode: number) {
  return request.patch(`/admin/vip-subscriptions/${id}/display`, { display_mode: displayMode })
}

export function setDefaultVIPSubscription(id: number) {
  return request.patch(`/admin/vip-subscriptions/${id}/default`)
}

export function cloneVIPSubscription(id: number, sukCode: string, name = '') {
  return request.post(`/admin/vip-subscriptions/${id}/clone`, { suk_code: sukCode, name })
}
