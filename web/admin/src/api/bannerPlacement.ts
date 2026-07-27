import request from '@/utils/request'

export interface BannerPlacement {
  id: number
  placement_name: string
  placement_key: string
  description: string
  cover_image: string
  sort: number
  status: number
  created_at: string
  updated_at: string
}

export type BannerPlacementPayload = Omit<BannerPlacement, 'id' | 'created_at' | 'updated_at'>

export function getBannerPlacementList(params: Record<string, unknown>) {
  return request.get('/admin/banner-placements', { params })
}

export function getBannerPlacementOptions() {
  return request.get('/admin/banner-placements/options')
}

export function getBannerPlacement(id: number) {
  return request.get(`/admin/banner-placements/${id}`)
}

export function createBannerPlacement(data: BannerPlacementPayload) {
  return request.post('/admin/banner-placements', data)
}

export function updateBannerPlacement(id: number, data: BannerPlacementPayload) {
  return request.put(`/admin/banner-placements/${id}`, data)
}

export function deleteBannerPlacement(id: number) {
  return request.delete(`/admin/banner-placements/${id}`)
}
