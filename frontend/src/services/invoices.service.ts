import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface InvoiceView {
  id: string
  order_id: string
  invoice_no: string
  issue_date: string
  due_date?: string | null
  subtotal: number
  tax_amount: number
  discount: number
  total: number
  status: string
  paid_at?: string | null
  billing_name: string
  created_at: string
  updated_at: string
}

export interface InvoiceSummary {
  total: number
  draft: number
  issued: number
  paid: number
  void: number
  outstanding: number
  collected: number
  by_status: Record<string, number>
}

export interface CreateInvoiceRequest {
  order_id: string
  due_date?: string | null
  subtotal: number
  tax_amount?: number
  discount?: number
  billing_name: string
}

export interface UpdateInvoiceRequest {
  id: string
  due_date?: string | null
  subtotal?: number
  tax_amount?: number
  discount?: number
  billing_name?: string
  status?: string
}

export function listInvoices(req: PageRequest) {
  return listRequest<InvoiceView, InvoiceSummary>('/invoices/list', req)
}

export function createInvoice(req: CreateInvoiceRequest): Promise<InvoiceView> {
  return postRequest<CreateInvoiceRequest, InvoiceView>('/invoices', req)
}

export function updateInvoice(req: UpdateInvoiceRequest): Promise<InvoiceView> {
  return patchRequest<UpdateInvoiceRequest, InvoiceView>('/invoices', req)
}

export function markInvoicePaid(id: string): Promise<InvoiceView> {
  return postRequest<{ id: string }, InvoiceView>('/invoices/mark-paid', { id })
}

export function deleteInvoice(id: string): Promise<void> {
  return deleteRequest('/invoices', { id })
}

export const INVOICE_STATUSES = ['draft', 'issued', 'paid', 'void'] as const
