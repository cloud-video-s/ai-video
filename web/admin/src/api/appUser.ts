import request from '@/utils/request'

export interface AppUser {
  id: number
  imei: string
  username: string
  device_country: string
  channel_id: string
  app_version: string
  app_name: string
  first_opened_at: string | null
  last_opened_at: string | null
  login_type: number
  user_type: number
  active_days: number
  avg_daily_usage_seconds: number
  vip_expires_at: string | null
  points_balance: number
  subscription_status: number
  first_order_created_at: string | null
  first_paid_at: string | null
  order_count: number
  payment_count: number
  subscription_payment_count: number
  one_time_payment_count: number
  order_amount_money: number
  actual_amount_money: number
  last_paid_at: string | null
  refund_amount_money: number
  points_money: number
  ai_cots_money: number
  activated: number
  key_behavior_met: number
  payment_met: boolean
  first_payment_met: boolean
  registered: boolean
  attribution_clicked_at: string | null
  phone_model: string
  re_registered_from_id?: number | null
  appid_email?: string | null
  appid_third_code?: string | null
  google_email?: string | null
  google_third_code?: string | null
  token_version: number
  status: number
  last_login_at: string | null
  last_login_ip: string
  login_account: string
  created_at: string
  updated_at: string
}

export type AppUserPayload = Omit<AppUser, 'id' | 'token_version' | 'created_at' | 'updated_at'>

export function getAppUserList(params: Record<string, unknown>) {
  return request.get('/admin/app-users', { params })
}

export function getAppUser(id: number) {
  return request.get(`/admin/app-users/${id}`)
}

export function createAppUser(data: AppUserPayload) {
  return request.post('/admin/app-users', data)
}

export function updateAppUser(id: number, data: Partial<AppUserPayload>) {
  return request.put(`/admin/app-users/${id}`, data)
}

export function deleteAppUser(id: number) {
  return request.delete(`/admin/app-users/${id}`)
}
