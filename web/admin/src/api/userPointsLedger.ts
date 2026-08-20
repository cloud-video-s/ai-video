import request from '@/utils/request'
import type { AppUser } from '@/api/appUser'
import type { Points } from '@/api/points.ts'

export interface UserPointsLedger {
  id: number
  user_id: number
  direction: number
  points_change: number
  balance_before: number
  balance_after: number
  source_type: number
  order_code: string
  points_id: number
  vip_id: number
  admin_id: number
  description: string
  occurred_at: string
  created_at: string
  user?: AppUser
  points_package?: Points
}

export interface UserPointsLedgerSummary {
  income_total: number
  expense_total: number
}

export function getUserPointsLedgerList(params: Record<string, unknown>) {
  return request.get('/admin/user-points-ledgers', { params })
}

export function getUserPointsLedger(id: number) {
  return request.get(`/admin/user-points-ledgers/${id}`)
}
