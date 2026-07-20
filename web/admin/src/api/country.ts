import request from '@/utils/request'

export interface Country {
  id: number
  code: string
  name_zh: string
  status: number
  created_at: string
  updated_at: string
}

export interface CountryPayload {
  code: string
  name_zh: string
  status: number
}

export function getCountryList(params: Record<string, unknown>) {
  return request.get('/admin/countries', { params })
}

export function getCountryOptions() {
  return request.get('/admin/countries/options')
}

export function createCountry(data: CountryPayload) {
  return request.post('/admin/countries', data)
}

export function updateCountry(id: number, data: CountryPayload) {
  return request.put(`/admin/countries/${id}`, data)
}

export function updateCountryStatus(id: number, status: number) {
  return request.patch(`/admin/countries/${id}/status`, { status })
}

export function deleteCountry(id: number) {
  return request.delete(`/admin/countries/${id}`)
}
