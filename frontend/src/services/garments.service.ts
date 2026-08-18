import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface GarmentView {
  id: string
  order_id: string
  name: string
  type: string
  color: string
  fabric: string
  quantity: number
  status: string
  instructions: string
  damage_notes: string
  created_at: string
  updated_at: string
}

export interface GarmentSummary {
  total_items: number
  received: number
  cleaned: number
  ironed: number
  ready: number
  damaged: number
  by_status: Record<string, number>
}

export interface CreateGarmentRequest {
  order_id: string
  name: string
  type: string
  color: string
  fabric: string
  quantity: number
  instructions?: string
  damage_notes?: string
}

export interface UpdateGarmentRequest {
  id: string
  name?: string
  type?: string
  color?: string
  fabric?: string
  quantity?: number
  status?: string
  instructions?: string
  damage_notes?: string
}

export function listGarments(req: PageRequest) {
  return listRequest<GarmentView, GarmentSummary>('/garments/list', req)
}

export function createGarment(req: CreateGarmentRequest): Promise<GarmentView> {
  return postRequest<CreateGarmentRequest, GarmentView>('/garments', req)
}

export function updateGarment(req: UpdateGarmentRequest): Promise<GarmentView> {
  return patchRequest<UpdateGarmentRequest, GarmentView>('/garments', req)
}

export function deleteGarment(id: string): Promise<void> {
  return deleteRequest('/garments', { id })
}

export const GARMENT_STATUSES = ['received', 'cleaned', 'ironed', 'ready', 'damaged'] as const
