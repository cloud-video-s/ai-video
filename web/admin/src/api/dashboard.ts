import request from '@/utils/request'
import type { VideoOrder } from '@/api/order'

export interface DashboardCurrencyAmount {
  currency: string
  amount: number
}

export interface DashboardStatistics {
  month: string
  period_start: string
  period_end: string
  timezone: string
  registered_user_count: number
  successful_subscription_count: number
  transaction_amounts: DashboardCurrencyAmount[]
  successful_generation_task_count: number
}

export interface DashboardData {
  statistics: DashboardStatistics
  subscription_orders: {
    list: VideoOrder[]
    total: number
    page: number
    size: number
  }
}

export function getDashboard(params: Record<string, unknown>) {
  return request.get('/admin/dashboard', { params })
}
