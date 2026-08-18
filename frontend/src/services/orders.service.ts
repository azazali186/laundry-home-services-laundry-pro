import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface OrderView {
  id: string
  order_no: string
  booking_id: string
  customer_id: string
  service_type_id: string
  pickup_id: string
  status: string
  priority: string
  item_count: number
  total_amount: number
  tax_amount: number
  discount_amount: number
  grand_total: number
  currency: string
  note: string
  created_at: string
  updated_at: string
}

export interface OrderSummary {
  total: number
  by_status: Record<string, number>
  revenue: number
  collected: number
  pending_value: number
  avg_order_value: number
  items_processed: number
}

export interface CreateOrderRequest {
  booking_id: string
  customer_id: string
  service_type_id: string
  pickup_id?: string | null
  priority?: string
  item_count: number
  total_amount: number
  tax_rate?: number
  discount_amount?: number
  currency: string
  note?: string
}

export interface UpdateOrderRequest {
  id: string
  priority?: string
  item_count?: number
  total_amount?: number
  tax_amount?: number
  discount_amount?: number
  note?: string
}

export function listOrders(req: PageRequest) {
  return listRequest<OrderView, OrderSummary>('/orders/list', req)
}

export function createOrder(req: CreateOrderRequest): Promise<OrderView> {
  return postRequest<CreateOrderRequest, OrderView>('/orders', req)
}

export function updateOrder(req: UpdateOrderRequest): Promise<OrderView> {
  return patchRequest<UpdateOrderRequest, OrderView>('/orders', req)
}

export function updateOrderStatus(id: string, status: string, reason?: string): Promise<OrderView> {
  return postRequest<{ id: string; status: string; reason?: string }, OrderView>(
    '/orders/status',
    { id, status, reason },
  )
}

export function deleteOrder(id: string): Promise<void> {
  return deleteRequest('/orders', { id })
}

export const ORDER_STATUSES = [
  'pending',
  'processing',
  'cleaned',
  'ironed',
  'ready',
  'delivered',
  'cancelled',
] as const
