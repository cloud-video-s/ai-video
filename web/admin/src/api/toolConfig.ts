import request from '@/utils/request'

export interface ToolReferenceImageOption {
  name: string
  image: string
  sort: number
}

export interface ToolRatioOption {
  name: string
  value: string
  sort: number
}

export interface ToolConfigData {
  reference_images?: ToolReferenceImageOption[]
  ratio_options?: ToolRatioOption[]
  age_range?: { min: number; max: number }
}

export interface ToolModelOption {
  id: number
  name: string
  model_type: 1 | 2
}

export type ToolsType =
  | 'enhance'
  | 'outpaint'
  | 'hairstyle'
  | 'age_transform'
  | 'body_reshape'
  | 'colorful'
  | 'makeup'
  | 'outfit'
  | 'pose_transfer'

export interface ToolConfig {
  id: number
  name: string
  icon: string
  background_image: string
  tool_type: 1 | 2
  tools_type: ToolsType
  model_id: number
  model_name: string
  config_type: 1 | 2 | 3 | 4
  config_data: ToolConfigData
  badge_image: string
  sort: number
  prompt: string
  status: number
  created_at: string
  updated_at: string
}

export type ToolConfigPayload = Pick<
  ToolConfig,
  | 'name'
  | 'icon'
  | 'background_image'
  | 'tool_type'
  | 'tools_type'
  | 'model_id'
  | 'config_type'
  | 'config_data'
  | 'badge_image'
  | 'sort'
  | 'prompt'
  | 'status'
>

export function getToolConfigList(params: Record<string, unknown>) {
  return request.get('/admin/tool-configs', { params })
}

export function getToolConfigOptions() {
  return request.get('/admin/tool-configs/options')
}

export function getToolModelOptions(toolType: 1 | 2) {
  return request.get('/admin/tool-configs/model-options', { params: { tool_type: toolType } })
}

export function getToolConfig(id: number) {
  return request.get(`/admin/tool-configs/${id}`)
}

export function createToolConfig(data: ToolConfigPayload) {
  return request.post('/admin/tool-configs', data)
}

export function updateToolConfig(id: number, data: ToolConfigPayload) {
  return request.put(`/admin/tool-configs/${id}`, data)
}

export function updateToolConfigStatus(id: number, status: number) {
  return request.patch(`/admin/tool-configs/${id}/status`, { status })
}

export function deleteToolConfig(id: number) {
  return request.delete(`/admin/tool-configs/${id}`)
}
