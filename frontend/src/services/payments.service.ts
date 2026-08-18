import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface PaymentView {
  id: string
  order_id: string
  payment_no: string
  amount: number
  currency: string
  method: string
  status: string
  gateway: string
  card_last4: string
  paid_at?: string | null
  note: string
  created_at: string
  updated_at: string
}

export interface PaymentSummary {
  total: number
  pending: number
  paid: number
  failed: number
  refunded: number
  collected: number
  outstanding: number
  by_method: Record<string, number>
  refunded_sum: number
}

export interface CreatePaymentRequest {
  order_id: string
  amount: number
  currency: string
  method: string
  gateway?: string
  transaction_id?: string
  card_last4?: string
  note?: string
}

export interface UpdatePaymentRequest {
  id: string
  method?: string
  gateway?: string
  note?: string
}

export function listPayments(req: PageRequest) {
  return listRequest<PaymentView, PaymentSummary>('/payments/list', req)
}

export function createPayment(req: CreatePaymentRequest): Promise<PaymentView> {
  return postRequest<CreatePaymentRequest, PaymentView>('/payments', req)
}

export function capturePayment(id: string): Promise<PaymentView> {
  return postRequest<{ id: string }, PaymentView>('/payments/capture', { id })
}

export function refundPayment(id: string, amount?: number): Promise<PaymentView> {
  return postRequest<{ id: string; amount?: number }, PaymentView>('/payments/refund', {
    id,
    amount,
  })
}

export function updatePayment(req: UpdatePaymentRequest): Promise<PaymentView> {
  return patchRequest<UpdatePaymentRequest, PaymentView>('/payments', req)
}

export function deletePayment(id: string): Promise<void> {
  return deleteRequest('/payments', { id })
}

export const PAYMENT_METHODS = ['cash', 'card', 'bank_transfer', 'mobile_money'] as const
export const PAYMENT_STATUSES = ['pending', 'paid', 'failed', 'refunded'] as const
