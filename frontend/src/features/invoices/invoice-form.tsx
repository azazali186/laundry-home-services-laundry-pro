import { useEffect, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@/lib/api'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { fetchAllPages } from '@/lib/fetchAll'
import { createInvoice, updateInvoice, type CreateInvoiceRequest, type UpdateInvoiceRequest, type InvoiceView } from '@/services/invoices.service'
import { listOrders, type OrderView } from '@/services/orders.service'
import { formatCurrency } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Field, Input, Select } from '@/components/ui/form'
import { Modal } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'

const schema = z.object({
  order_id: z.string().min(1, 'Order is required'),
  subtotal: z.coerce.number().positive('Enter a valid subtotal'),
  billing_name: z.string().min(1, 'Billing name is required'),
})

interface InvoiceFormProps {
  open: boolean
  invoice: InvoiceView | null
  onClose: () => void
}

export function InvoiceForm({ open, invoice, onClose }: InvoiceFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const isEdit = !!invoice

  const ordersQuery = useQuery({
    queryKey: ['orders', 'all'],
    queryFn: () => fetchAllPages<OrderView>(listOrders, { limit: 100, sort_by: 'order_no', maxPages: 10 }),
    staleTime: 5 * 60_000,
  })

  const [orderId, setOrderId] = useState('')
  const [subtotal, setSubtotal] = useState('')
  const [tax, setTax] = useState('0')
  const [discount, setDiscount] = useState('0')
  const [billingName, setBillingName] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setOrderId(invoice?.order_id ?? '')
    setSubtotal(invoice ? String(invoice.subtotal) : '')
    setTax(invoice ? String(invoice.tax_amount) : '0')
    setDiscount(invoice ? String(invoice.discount) : '0')
    setBillingName(invoice?.billing_name ?? '')
    setDueDate(invoice?.due_date ? invoice.due_date.slice(0, 10) : '')
    setErrors({})
  }, [open, invoice])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && invoice) {
        const req: UpdateInvoiceRequest = {
          id: invoice.id,
          due_date: dueDate || null,
          subtotal: Number(subtotal),
          tax_amount: Number(tax),
          discount: Number(discount),
          billing_name: billingName.trim(),
        }
        return updateInvoice(req)
      }
      const req: CreateInvoiceRequest = {
        order_id: orderId,
        due_date: dueDate || null,
        subtotal: Number(subtotal),
        tax_amount: Number(tax),
        discount: Number(discount),
        billing_name: billingName.trim(),
      }
      return createInvoice(req)
    },
    onSuccess: () => {
      toast.toast({ title: isEdit ? 'Invoice updated' : 'Invoice created', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['invoices'] })
      void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      onClose()
    },
    onError: (err) => toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' }),
  })

  const shopCurrency = useShopCurrency()
  const orderCurrency = ordersQuery.data?.find((o) => o.id === orderId)?.currency ?? shopCurrency
  const total = Number(subtotal || 0) + Number(tax || 0) - Number(discount || 0)

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = schema.safeParse({ order_id: orderId, subtotal, billing_name: billingName })
    if (!result.success) {
      const fieldErrors: Record<string, string> = {}
      for (const issue of result.error.issues) fieldErrors[issue.path[0] as string] = issue.message
      setErrors(fieldErrors)
      return
    }
    setErrors({})
    void mutation.mutate()
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? `Edit ${invoice?.invoice_no}` : 'New invoice'}
      description={isEdit ? 'Update invoice lines and status.' : 'Bill an order.'}
      size="lg"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Create invoice'}
          </Button>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <Field label="Order" required error={errors.order_id}>
          <Select value={orderId} onChange={(e) => setOrderId(e.target.value)} disabled={isEdit}>
            <option value="">Select order…</option>
            {ordersQuery.data?.map((o) => (
              <option key={o.id} value={o.id}>
                {o.order_no} — {formatCurrency(o.grand_total, o.currency)}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Billing name" required error={errors.billing_name}>
          <Input value={billingName} onChange={(e) => setBillingName(e.target.value)} />
        </Field>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <Field label="Subtotal" required error={errors.subtotal}>
            <Input type="number" step="0.01" min="0" value={subtotal} onChange={(e) => setSubtotal(e.target.value)} />
          </Field>
          <Field label="Tax">
            <Input type="number" step="0.01" min="0" value={tax} onChange={(e) => setTax(e.target.value)} />
          </Field>
          <Field label="Discount">
            <Input type="number" step="0.01" min="0" value={discount} onChange={(e) => setDiscount(e.target.value)} />
          </Field>
        </div>
        <div className="flex items-end justify-between gap-4">
          <Field label="Due date">
            <Input type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} />
          </Field>
          <p className="pb-2 text-sm">
            <span className="text-surface-400">Total: </span>
            <span className="font-bold tabular-nums text-surface-900">{formatCurrency(total, orderCurrency)}</span>
          </p>
        </div>
      </form>
    </Modal>
  )
}
