import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface PickupView {
  id: string
  booking_id: string
  agent_id: string
  scheduled_at: string
  completed_at?: string | null
  address: string
  status: string
  notes: string
  created_at: string
  updated_at: string
}

export interface PickupSummary {
  total: number
  scheduled: number
  completed: number
  active: number
}

export interface CreatePickupRequest {
  booking_id: string
  agent_id?: string | null
  scheduled_at: string
  address: string
  notes?: string
}

export interface UpdatePickupRequest {
  id: string
  agent_id?: string | null
  scheduled_at?: string
  address?: string
  notes?: string
}

export function listPickups(req: PageRequest) {
  return listRequest<PickupView, PickupSummary>('/pickups/list', req)
}

export function createPickup(req: CreatePickupRequest): Promise<PickupView> {
  return postRequest<CreatePickupRequest, PickupView>('/pickups', req)
}

export function updatePickup(req: UpdatePickupRequest): Promise<PickupView> {
  return patchRequest<UpdatePickupRequest, PickupView>('/pickups', req)
}

export function completePickup(id: string): Promise<PickupView> {
  return postRequest<{ id: string }, PickupView>('/pickups/complete', { id })
}

export function deletePickup(id: string): Promise<void> {
  return deleteRequest('/pickups', { id })
}
