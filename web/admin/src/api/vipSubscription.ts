import request from '@/utils/request'
import type { AppPackage } from '@/api/package'
import type { DisplayPosition } from '@/api/displayPosition'
import type { Channel } from '@/api/channel'

export interface VIPSubscription {
  id: number
  package_id: number
  platform: string
  product_id: string
  name: string
  vip_level: string
  plan_type: string
  app_version: string
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
  subscription_period: string
  sort: number
  description: string
  remark: string
  package?: AppPackage
  display_positions?: DisplayPosition[]
  channels?: Channel[]
  excluded_channels?: Channel[]
  created_at: string
  updated_at: string
}

export interface VIPSubscriptionPayload {
  package_id: number
  platform: string
  product_id: string
  name: string
  vip_level: string
  plan_type: string
  display_position_ids: number[]
  channel_ids: number[]
  excluded_channel_ids: number[]
  app_version: string
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
  subscription_period: string
  sort: number
  description: string
  remark: string
}

export function getVIPSubscriptionList(params: Record<string, unknown>) { return request.get('/admin/vip-subscriptions', { params }) }
export function getVIPSubscription(id: number) { return request.get(`/admin/vip-subscriptions/${id}`) }
export function createVIPSubscription(data: VIPSubscriptionPayload) { return request.post('/admin/vip-subscriptions', data) }
export function updateVIPSubscription(id: number, data: VIPSubscriptionPayload) { return request.put(`/admin/vip-subscriptions/${id}`, data) }
export function deleteVIPSubscription(id: number) { return request.delete(`/admin/vip-subscriptions/${id}`) }
export function updateVIPSubscriptionStatus(id: number, status: number) { return request.patch(`/admin/vip-subscriptions/${id}/status`, { status }) }
export function updateVIPSubscriptionDisplay(id: number, displayMode: number) { return request.patch(`/admin/vip-subscriptions/${id}/display`, { display_mode: displayMode }) }
export function setDefaultVIPSubscription(id: number) { return request.patch(`/admin/vip-subscriptions/${id}/default`) }
export function cloneVIPSubscription(id: number, productId: string, name = '') { return request.post(`/admin/vip-subscriptions/${id}/clone`, { product_id: productId, name }) }
