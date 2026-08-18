import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface InventoryView {
  id: string
  name: string
  sku: string
  category: string
  quantity_on_hand: number
  reorder_level: number
  unit: string
  unit_cost: number
  is_active: boolean
  low_stock: boolean
  stock_value: number
  created_at: string
  updated_at: string
}

export interface InventorySummary {
  total_items: number
  low_stock: number
  out_of_stock: number
  total_value: number
  active_items: number
  by_category: Record<string, number>
}

export interface CreateInventoryRequest {
  name: string
  sku: string
  category: string
  quantity_on_hand: number
  reorder_level?: number
  unit: string
  unit_cost?: number
  is_active?: boolean
}

export interface UpdateInventoryRequest {
  id: string
  name?: string
  category?: string
  reorder_level?: number
  unit?: string
  unit_cost?: number
  is_active?: boolean
}

export interface AdjustInventoryRequest {
  id: string
  delta: number
  reason: string
}

export function listInventory(req: PageRequest) {
  return listRequest<InventoryView, InventorySummary>('/inventory/list', req)
}

export function createInventory(req: CreateInventoryRequest): Promise<InventoryView> {
  return postRequest<CreateInventoryRequest, InventoryView>('/inventory', req)
}

export function updateInventory(req: UpdateInventoryRequest): Promise<InventoryView> {
  return patchRequest<UpdateInventoryRequest, InventoryView>('/inventory', req)
}

export function adjustInventory(req: AdjustInventoryRequest): Promise<InventoryView> {
  return postRequest<AdjustInventoryRequest, InventoryView>('/inventory/adjust', req)
}

export function deleteInventory(id: string): Promise<void> {
  return deleteRequest('/inventory', { id })
}
