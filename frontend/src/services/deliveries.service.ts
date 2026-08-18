import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface DeliveryView {
  id: string
  order_id: string
  agent_id: string
  scheduled_at: string
  delivered_at?: string | null
  address: string
  status: string
  notes: string
  created_at: string
  updated_at: string
}

export interface DeliverySummary {
  total: number
  scheduled: number
  in_transit: number
  delivered: number
  cancelled: number
}

export interface CreateDeliveryRequest {
  order_id: string
  agent_id?: string | null
  scheduled_at: string
  address: string
  notes?: string
}

export interface UpdateDeliveryRequest {
  id: string
  agent_id?: string | null
  scheduled_at?: string
  address?: string
  notes?: string
}

export function listDeliveries(req: PageRequest) {
  return listRequest<DeliveryView, DeliverySummary>('/deliveries/list', req)
}

export function createDelivery(req: CreateDeliveryRequest): Promise<DeliveryView> {
  return postRequest<CreateDeliveryRequest, DeliveryView>('/deliveries', req)
}

export function updateDelivery(req: UpdateDeliveryRequest): Promise<DeliveryView> {
  return patchRequest<UpdateDeliveryRequest, DeliveryView>('/deliveries', req)
}

export function completeDelivery(id: string): Promise<DeliveryView> {
  return postRequest<{ id: string }, DeliveryView>('/deliveries/complete', { id })
}

export function deleteDelivery(id: string): Promise<void> {
  return deleteRequest('/deliveries', { id })
}
