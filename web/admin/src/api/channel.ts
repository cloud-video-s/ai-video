import request from '@/utils/request'

export interface Channel {
  channel_id: number
  channel_code: string
  channel_name: string
  agency_company: string
  ad_platform: string
  delivery_package: string
  tracking_url: string
  port_rebate: number
  service_order_fee: number
  upload_method: string
  status: number
  created_at: string
  updated_at: string
}

export type ChannelPayload = Omit<Channel, 'channel_id' | 'created_at' | 'updated_at'>

export function getChannelList(params: Record<string, unknown>) {
  return request.get('/admin/channels', { params })
}

export function getChannelOptions() {
  return request.get('/admin/channels/options')
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
