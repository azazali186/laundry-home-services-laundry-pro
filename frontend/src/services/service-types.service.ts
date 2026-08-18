import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface ServiceTypeView {
  id: string
  name: string
  code: string
  description: string
  unit: string
  price: number
  currency: string
  tax_rate: number
  estimated_days: number
  is_active: boolean
  icon: string
  created_at: string
  updated_at: string
}

export interface ServiceTypeSummary {
  total_active: number
  total_inactive: number
  avg_price: number
  max_price: number
  min_price: number
  by_unit: Record<string, number>
}

export interface CreateServiceTypeRequest {
  name: string
  code: string
  description?: string
  unit: string
  price: number
  currency: string
  tax_rate?: number
  estimated_days?: number
  is_active?: boolean
  icon?: string
}

export interface UpdateServiceTypeRequest {
  id: string
  name?: string
  description?: string
  unit?: string
  price?: number
  currency?: string
  tax_rate?: number
  estimated_days?: number
  is_active?: boolean
  icon?: string
}

export function listServiceTypes(req: PageRequest) {
  return listRequest<ServiceTypeView, ServiceTypeSummary>('/service-types/list', req)
}

export function createServiceType(req: CreateServiceTypeRequest): Promise<ServiceTypeView> {
  return postRequest<CreateServiceTypeRequest, ServiceTypeView>('/service-types', req)
}

export function updateServiceType(req: UpdateServiceTypeRequest): Promise<ServiceTypeView> {
  return patchRequest<UpdateServiceTypeRequest, ServiceTypeView>('/service-types', req)
}

export function deleteServiceType(id: string): Promise<void> {
  return deleteRequest('/service-types', { id })
}
