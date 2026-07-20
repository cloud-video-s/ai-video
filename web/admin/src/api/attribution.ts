import request from '@/utils/request'

export type AttributionEvent = 'activation' | 'key_behavior' | 'payment' | 'first_payment' | 'registration'
export type AttributionAction = 'callback' | 'deduct'

export interface AttributionUser {
  id: number
  username: string
  imei: string
  channel_id: string
  activated: number
  key_behavior_met: number
  payment_met: boolean
  first_payment_met: boolean
  registered: boolean
  last_login_ip: string
}

export interface AttributionRecord {
  id: number
  user_id: number
  channel_code: string
  oaid: string
  imei: string
  android_id: string
  ip: string
  user_agent: string
  activation_callback_count: number
  activation_deduct_count: number
  key_behavior_callback_count: number
  key_behavior_deduct_count: number
  payment_callback_count: number
  payment_deduct_count: number
  first_payment_callback_count: number
  first_payment_deduct_count: number
  registration_callback_count: number
  registration_deduct_count: number
  attributed_at?: string
  last_operated_at?: string
  remark: string
  created_at: string
  user: AttributionUser
  channel?: {
    channel_id: number
    channel_code: string
    channel_name: string
  }
}

export function getAttributionList(params: Record<string, unknown>) {
  return request.get('/admin/user-attributions', { params })
}

export function getAttribution(id: number) {
  return request.get(`/admin/user-attributions/${id}`)
}

export function updateAttribution(id: number, data: Record<string, unknown>) {
  return request.put(`/admin/user-attributions/${id}`, data)
}

export function recordAttributionEvent(id: number, event: AttributionEvent, action: AttributionAction) {
  return request.post(`/admin/user-attributions/${id}/events`, { event, action })
}

export function syncAttributionUsers() {
  return request.post('/admin/user-attributions/sync')
}
