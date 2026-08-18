import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface ComboItemView {
  service_type_id: string
  quantity: number
  /** Service snapshot — what the bundle includes, straight from the backend. */
  service_name?: string
  unit?: string
  price?: number
  currency?: string
  icon?: string
}

export interface ComboView {
  id: string
  name: string
  code: string
  description: string
  unit: string
  price: number
  currency: string
  is_active: boolean
  is_featured: boolean
  validity_days: number
  renewal_type: string
  validity_text: string
  items: ComboItemView[]
  /** À-la-carte value of the included items; savings = regular_price − price. */
  regular_price?: number
  savings?: number
  created_at: string
  updated_at: string
}

export interface ComboSummary {
  total_active: number
  total_inactive: number
  avg_price: number
  max_price: number
  min_price: number
  by_unit: Record<string, number>
}

export interface ComboItemRequest {
  service_type_id: string
  quantity: number
}

export interface CreateComboRequest {
  name: string
  code: string
  description?: string
  unit: string
  price: number
  currency: string
  is_active?: boolean
  is_featured?: boolean
  validity_days?: number
  renewal_type?: string
  items: ComboItemRequest[]
}

export interface UpdateComboRequest {
  id: string
  code?: string
  name?: string
  description?: string
  unit?: string
  price?: number
  currency?: string
  is_active?: boolean
  is_featured?: boolean
  validity_days?: number
  renewal_type?: string
  items?: ComboItemRequest[]
}

export function listCombos(req: PageRequest) {
  return listRequest<ComboView, ComboSummary>('/combos/list', req)
}

export function createCombo(req: CreateComboRequest): Promise<ComboView> {
  return postRequest<CreateComboRequest, ComboView>('/combos', req)
}

export function updateCombo(req: UpdateComboRequest): Promise<ComboView> {
  return patchRequest<UpdateComboRequest, ComboView>('/combos', req)
}

export function deleteCombo(id: string): Promise<void> {
  return deleteRequest('/combos', { id })
}
