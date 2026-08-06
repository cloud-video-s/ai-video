import request from '@/utils/request'

export interface OrderPurchaser {
  id: number
  username: string
  login_account: string
  email: string
  phone: string
  imei: string
  device_code: string
  user_type: number
  status: number
  deleted: boolean
}

export interface VideoOrder {
  id: number
  order_no: string
  client_request_id: string
  user_id: number
  product_type: number
  product_id: number
  product_code: string
  product_name: string
  currency: string
  product_amount: number
  discount_amount: number
  payable_amount: number
  paid_amount: number
  refunded_amount: number
  bonus_points: number
  vip_level: number
  vip_duration_days: number
  status: number
  pay_type: number
  third_order_no: string
  original_transaction_id: string
  payment_evidence?: string
  failure_code: string
  failure_message: string
  cancel_reason: string
  pay_time: string | null
  cancelled_at: string | null
  expires_at: string | null
  created_at: string
  updated_at: string
  deleted_at: string | null
  user?: OrderPurchaser
}

export interface OrderSummary {
  paid_order_count: number
  amounts: Array<{
    currency: string
    payable_total: number
    paid_total: number
    refunded_total: number
  }>
}

export function getOrderList(params: Record<string, unknown>) {
  return request.get('/admin/orders', { params })
}

export function getOrder(id: number) {
  return request.get(`/admin/orders/${id}`)
}
