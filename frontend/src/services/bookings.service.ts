import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface BookingView {
  id: string
  booking_no: string
  customer_id: string
  service_type_id: string
  pickup_address: string
  dropoff_address: string
  pickup_from?: string | null
  pickup_to?: string | null
  status: string
  notes: string
  estimated_amount: number
  currency: string
  source: string
  created_at: string
  updated_at: string
}

export interface BookingSummary {
  total: number
  pending: number
  confirmed: number
  cancelled: number
  estimated_value: number
  this_month: number
  by_status: Record<string, number>
}

export interface CreateBookingRequest {
  customer_id: string
  service_type_id: string
  pickup_address: string
  dropoff_address: string
  pickup_from?: string | null
  pickup_to?: string | null
  notes?: string
  estimated_amount: number
  currency: string
  source?: string
}

export interface UpdateBookingRequest {
  id: string
  pickup_address?: string
  dropoff_address?: string
  pickup_from?: string | null
  pickup_to?: string | null
  notes?: string
  estimated_amount?: number
}

export function listBookings(req: PageRequest) {
  return listRequest<BookingView, BookingSummary>('/bookings/list', req)
}

export function createBooking(req: CreateBookingRequest): Promise<BookingView> {
  return postRequest<CreateBookingRequest, BookingView>('/bookings', req)
}

export function updateBooking(req: UpdateBookingRequest): Promise<BookingView> {
  return patchRequest<UpdateBookingRequest, BookingView>('/bookings', req)
}

export function confirmBooking(id: string): Promise<BookingView> {
  return postRequest<{ id: string }, BookingView>('/bookings/confirm', { id })
}

export function cancelBooking(id: string, reason: string): Promise<BookingView> {
  return postRequest<{ id: string; reason: string }, BookingView>('/bookings/cancel', {
    id,
    reason,
  })
}

export function deleteBooking(id: string): Promise<void> {
  return deleteRequest('/bookings', { id })
}
