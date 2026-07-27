import request from '@/utils/request'
import type { AppPackage } from '@/api/package'
import type { PackageVersion } from '@/api/packageVersion'
import type { Country } from '@/api/country'
import type { DisplayPosition } from '@/api/displayPosition'
import type { VideoApp } from '@/api/videoApp'

export interface VideoTemplateType {
  id: number
  category_name: string
  display_positions: DisplayPosition[]
  countries: Country[]
  apps: VideoApp[]
  packages: AppPackage[]
  versions: PackageVersion[]
  sort: number
  status: number
  description: string
  created_at: string
  updated_at: string
}

export interface VideoTemplateTypePayload {
  category_name: string
  display_position_keys: string[]
  country_codes: string[]
  app_rules: Array<{ app_code: string }>
  package_codes: string[]
  version_codes: string[]
  sort: number
  status: number
  description: string
}

export interface VideoTemplate {
  id: number
  video_template_type_id: number
  name: string
  template_type: string
  sort: number
  cover_image: string
  template_video: string
  thumbnail_video: string
  prompt: string
  status: number
  description: string
  video_template_type?: VideoTemplateType
  created_at: string
  updated_at: string
}

export interface VideoTemplatePayload {
  video_template_type_id: number
  name: string
  template_type: string
  sort: number
  cover_image: string
  template_video: string
  thumbnail_video: string
  prompt: string
  status: number
  description: string
}

export function getTemplateTypeList(params: Record<string, unknown>) {
  return request.get('/admin/template-types', { params })
}

export function getTemplateTypeOptions() {
  return request.get('/admin/template-types/options')
}

export function getTemplateOptions() {
  return request.get('/admin/templates/options')
}

export function getTemplateType(id: number) {
  return request.get(`/admin/template-types/${id}`)
}

export function createTemplateType(data: VideoTemplateTypePayload) {
  return request.post('/admin/template-types', data)
}

export function updateTemplateType(id: number, data: VideoTemplateTypePayload) {
  return request.put(`/admin/template-types/${id}`, data)
}

export function deleteTemplateType(id: number) {
  return request.delete(`/admin/template-types/${id}`)
}

export function getTemplateList(params: Record<string, unknown>) {
  return request.get('/admin/templates', { params })
}

export function getTemplate(id: number) {
  return request.get(`/admin/templates/${id}`)
}

export function createTemplate(data: VideoTemplatePayload) {
  return request.post('/admin/templates', data)
}

export function updateTemplate(id: number, data: VideoTemplatePayload) {
  return request.put(`/admin/templates/${id}`, data)
}

export function deleteTemplate(id: number) {
  return request.delete(`/admin/templates/${id}`)
}

export interface TemplateDisplayConfig {
  id: number
  template_id: number
  position_key: string
  sort: number
  status: number
  remark: string
  template: VideoTemplate
  display_position: DisplayPosition
  created_at: string
  updated_at: string
}

export interface TemplateDisplayConfigPayload {
  template_id: number
  position_key: string
  sort: number
  status: number
  remark: string
}

export function getTemplateDisplayConfigList(params: Record<string, unknown>) {
  return request.get('/admin/template-display-configs', { params })
}

export function createTemplateDisplayConfig(data: TemplateDisplayConfigPayload) {
  return request.post('/admin/template-display-configs', data)
}

export function updateTemplateDisplayConfig(id: number, data: TemplateDisplayConfigPayload) {
  return request.put(`/admin/template-display-configs/${id}`, data)
}

export function deleteTemplateDisplayConfig(id: number) {
  return request.delete(`/admin/template-display-configs/${id}`)
}
