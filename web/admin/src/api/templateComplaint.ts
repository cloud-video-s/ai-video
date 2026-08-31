import request from '@/utils/request'

export interface TemplateComplaintUser {
  id: number
  username: string
  login_account: string
  email: string
  phone: string
  imei: string
  device_code: string
  status: number
  deleted: boolean
}

export interface TemplateComplaintTemplate {
  id: number
  name: string
  template_type: number
  cover_image_url: string
  original_url: string
  thumbnail_url: string
  status: number
  deleted: boolean
}

export interface TemplateComplaint {
  id: number
  user_id: number
  template_id: number
  complaint_type: string
  content: string
  created_at: string
  updated_at: string
  user: TemplateComplaintUser | null
  template: TemplateComplaintTemplate | null
}

export function getTemplateComplaintList(params: Record<string, unknown>) {
  return request.get('/admin/template-complaints', { params })
}

export function getTemplateComplaint(id: number) {
  return request.get(`/admin/template-complaints/${id}`)
}
