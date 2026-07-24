import request from '@/utils/request'

export interface AppUser {
  id: number
  device_code?: string
  email?: string
  imei: string
  username: string
  client_country: string
  server_country: string
  channel_id: string
  app_version: string
  app_name: string
  first_opened_at: string | null
  last_opened_at: string | null
  login_type: number
  user_type: number
  phone: string
  vip_level: number
  vip_started_at: string | null
  is_frozen: boolean | number
  is_blacklisted: boolean | number
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
  package_code: string
  created_at: string
  updated_at: string
}

export type AppUserPayload = Omit<AppUser, 'id' | 'token_version' | 'created_at' | 'updated_at'>

export interface UserIdentity {
  id: number
  provider: string
  email: string
  display_name: string
  last_login_at: string | null
}

export interface UserAttribution {
  id: number
  channel_code: string
  oaid: string
  imei: string
  android_id: string
  ip: string
  attributed_at: string | null
  remark: string
}

export interface UserPointsLedger {
  id: number
  direction: number
  points_change: number
  balance_after: number
  source_type: string
  description: string
  occurred_at: string
}

export interface UserCenterDetail {
  user: AppUser
  is_member: boolean
  identities: UserIdentity[]
  attribution: UserAttribution | null
  points_ledgers: UserPointsLedger[]
  points_summary: { income_total: number; expense_total: number }
}

export function getAppUserList(params: Record<string, unknown>) {
  return request.get('/admin/app-users', { params })
}

export function getAppUser(id: number) {
  return request.get(`/admin/app-users/${id}`)
}

export function lookupAppUser(query: string) {
  return request.get('/admin/app-users/lookup', { params: { query } })
}

export function getUserCenter(id: number) {
  return request.get(`/admin/app-users/${id}/center`)
}

export function setUserFrozen(id: number, enabled: boolean) {
  return request.patch(`/admin/app-users/${id}/frozen`, { enabled })
}

export function setUserBlacklisted(id: number, enabled: boolean) {
  return request.patch(`/admin/app-users/${id}/blacklisted`, { enabled })
}

export function bindUserPhone(id: number, phone: string) {
  return request.put(`/admin/app-users/${id}/phone`, { phone })
}

export function grantUserVIP(id: number, data: { level: number; started_at?: string | null; expires_at: string }) {
  return request.post(`/admin/app-users/${id}/vip`, data)
}

export function extendUserVIP(id: number, days: number) {
  return request.post(`/admin/app-users/${id}/vip/extend`, { days })
}

export function transferUserVIP(id: number, target_user_id: number) {
  return request.post(`/admin/app-users/${id}/vip/transfer`, { target_user_id })
}

export function terminateUserVIP(id: number) {
  return request.delete(`/admin/app-users/${id}/vip`)
}

export function clearUserDevice(id: number) {
  return request.delete(`/admin/app-users/${id}/device`)
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
