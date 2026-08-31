import request from '@/utils/request'

export type ChannelCallbackEvent = 'activation' | 'login' | 'order_created' | 'payment' | 'subscription'

export interface ChannelCallbackRule {
  trigger_event: ChannelCallbackEvent
  callback_events: ChannelCallbackEvent[]
  order_count_threshold: number
  payment_minimum_amount: number
  subscription_delay_minutes: number
  amount_deduction_enabled: boolean
  amount_deduction_percent: number
}

export interface ChannelCallbackConfig {
  rules: ChannelCallbackRule[]
}

export interface Channel {
  id: number
  channel_code: string
  channel_name: string
  account_channel: string
  agency_company: string
  ad_platform: string
  ad_media: string
  delivery_package: string
  delivery_package_name: string
  system_type: string
  owner_admin_id: number
  owner_username: string
  owner_nickname: string
  ad_account: string
  tracking_url: string
  landing_page: string
  port_rebate: number
  service_order_fee: number
  upload_method: string
  callback_config: ChannelCallbackConfig
  status: number
  created_at: string
  updated_at: string
}

export interface ChannelPayload {
  channel_code?: string
  channel_name: string
  account_channel: string
  agency_company: string
  ad_platform: string
  ad_media: string
  delivery_package: string
  system_type: string
  owner_admin_id: number
  ad_account: string
  tracking_url: string
  landing_page: string
  port_rebate: number
  service_order_fee: number
  upload_method: string
  callback_config: ChannelCallbackConfig
  status?: number
}

export interface ChannelListParams {
  page: number
  page_size: number
  agency_company?: string
  delivery_package?: string
  ad_platform?: string
  keyword?: string
}

export interface MediaOption {
  id: number
  name: string
  adjust_partner_id: number
}

export function getChannelList(params: ChannelListParams) {
  return request.get('/admin/channels', { params })
}

export function getChannelOptions() {
  return request.get('/admin/channels/options')
}

export function getMediaOptions() {
  return request.get('/admin/channels/media-options')
}

export function createChannel(data: ChannelPayload) {
  return request.post('/admin/channels', data)
}

export function updateChannel(id: number, data: ChannelPayload) {
  return request.put(`/admin/channels/${id}`, data)
}

export function updateChannelStatus(id: number, status: number) {
  return request.patch(`/admin/channels/${id}/status`, { status })
}

export function deleteChannel(id: number) {
  return request.delete(`/admin/channels/${id}`)
}
