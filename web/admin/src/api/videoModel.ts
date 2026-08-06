import request from '@/utils/request'

export interface VideoPlatform {
  id: number
  name: string
  code: string
  base_url: string
  description: string
  status: number
  created_at: string
  updated_at: string
}

export type VideoPlatformPayload = Pick<VideoPlatform, 'name' | 'code' | 'base_url' | 'description' | 'status'>

export interface VideoModel {
  id: number
  platform_id: number
  platform: Pick<VideoPlatform, 'id' | 'name' | 'code' | 'base_url'> | null
  name: string
  code: string
  model_type: number
  version: string
  host_url: string
  submit_endpoint: string
  status_endpoint: string
  request_method: 'GET' | 'POST'
  auth_type: 1 | 2
  api_key: string
  api_key_configured: boolean
  score: number
  icon: string
  description: string
  status: number
  created_at: string
  updated_at: string
}

export type VideoModelPayload = Omit<VideoModel, 'id' | 'platform' | 'api_key_configured' | 'created_at' | 'updated_at'>

export interface ModelParameter {
  id: number
  model_id: number
  param_key: string
  value_type: 'string' | 'integer' | 'number' | 'boolean' | 'object' | 'array'
  parameter_type: 1 | 2
  is_required: number
  default_value: unknown
  allowed_values: unknown[]
  constraints: Record<string, unknown>
  description: string
  sort_order: number
  alias: string
  display_type: 'string' | 'integer' | 'boolean' | 'object' | 'array' | 'select' | 'time'
  created_at: string
  updated_at: string
}

export type ModelParameterPayload = Omit<ModelParameter, 'id' | 'model_id' | 'created_at' | 'updated_at'>

export function getPlatformList(params: Record<string, unknown>) {
  return request.get('/admin/platforms', { params })
}

export function getPlatformOptions() {
  return request.get('/admin/platforms/options')
}

export function createPlatform(data: VideoPlatformPayload) {
  return request.post('/admin/platforms', data)
}

export function updatePlatform(id: number, data: VideoPlatformPayload) {
  return request.put(`/admin/platforms/${id}`, data)
}

export function deletePlatform(id: number) {
  return request.delete(`/admin/platforms/${id}`)
}

export function getModelList(params: Record<string, unknown>) {
  return request.get('/admin/models', { params })
}

export function createModel(data: VideoModelPayload) {
  return request.post('/admin/models', data)
}

export function updateModel(id: number, data: VideoModelPayload) {
  return request.put(`/admin/models/${id}`, data)
}

export function deleteModel(id: number) {
  return request.delete(`/admin/models/${id}`)
}

export function getModelParameters(modelId: number) {
  return request.get(`/admin/models/${modelId}/parameters`)
}

export function createModelParameter(modelId: number, data: ModelParameterPayload) {
  return request.post(`/admin/models/${modelId}/parameters`, data)
}

export function updateModelParameter(modelId: number, id: number, data: ModelParameterPayload) {
  return request.put(`/admin/models/${modelId}/parameters/${id}`, data)
}

export function deleteModelParameter(modelId: number, id: number) {
  return request.delete(`/admin/models/${modelId}/parameters/${id}`)
}
