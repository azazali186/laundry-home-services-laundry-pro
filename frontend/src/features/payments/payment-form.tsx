import { useEffect, useRef, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@/lib/api'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { fetchAllPages } from '@/lib/fetchAll'
import { createPayment, updatePayment, PAYMENT_METHODS, type CreatePaymentRequest, type UpdatePaymentRequest, type PaymentView } from '@/services/payments.service'
import { listOrders, type OrderView } from '@/services/orders.service'
import { CURRENCY_OPTIONS } from '@/services/settings.service'
import { formatCurrency } from '@/lib/format'
import { humanize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Field, Input, Select, Textarea } from '@/components/ui/form'
import { Modal } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'

const schema = z.object({
  order_id: z.string().min(1, 'Order is required'),
  amount: z.coerce.number().positive('Enter a valid amount'),
  method: z.string().min(1, 'Method is required'),
})

interface PaymentFormProps {
  open: boolean
  payment: PaymentView | null
  onClose: () => void
}

export function PaymentForm({ open, payment, onClose }: PaymentFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const shopCurrency = useShopCurrency()
  // Latest shop currency without re-running the reset effect when settings load.
  const shopCurrencyRef = useRef(shopCurrency)
  shopCurrencyRef.current = shopCurrency
  const isEdit = !!payment

  const ordersQuery = useQuery({
    queryKey: ['orders', 'all'],
    queryFn: () => fetchAllPages<OrderView>(listOrders, { limit: 100, sort_by: 'order_no', maxPages: 10 }),
    staleTime: 5 * 60_000,
  })

  const [orderId, setOrderId] = useState('')
  const [amount, setAmount] = useState('')
  const [currency, setCurrency] = useState(shopCurrency)
  const [method, setMethod] = useState('cash')
  const [gateway, setGateway] = useState('')
  const [transactionId, setTransactionId] = useState('')
  const [cardLast4, setCardLast4] = useState('')
  const [note, setNote] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setOrderId(payment?.order_id ?? '')
    setAmount(payment ? String(payment.amount) : '')
    setCurrency(payment?.currency ?? shopCurrencyRef.current)
    setMethod(payment?.method ?? 'cash')
    setGateway(payment?.gateway ?? '')
    setTransactionId('')
    setCardLast4(payment?.card_last4 ?? '')
    setNote(payment?.note ?? '')
    setErrors({})
  }, [open, payment])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && payment) {
        const req: UpdatePaymentRequest = { id: payment.id, method, gateway: gateway.trim() || undefined, note: note.trim() || undefined }
        return updatePayment(req)
      }
      const req: CreatePaymentRequest = {
        order_id: orderId,
        amount: Number(amount),
        currency,
        method,
        gateway: gateway.trim() || undefined,
        transaction_id: transactionId.trim() || undefined,
        card_last4: cardLast4.trim() || undefined,
        note: note.trim() || undefined,
      }
      return createPayment(req)
    },
    onSuccess: () => {
      toast.toast({ title: isEdit ? 'Payment updated' : 'Payment recorded', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['payments'] })
      onClose()
    },
    onError: (err) => toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' }),
  })

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = schema.safeParse({ order_id: orderId, amount, method })
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
      title={isEdit ? `Edit ${payment?.payment_no}` : 'Record payment'}
      description={isEdit ? 'Update payment metadata.' : 'Record a payment against an order.'}
      size="lg"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Record payment'}
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
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <Field label="Amount" required error={errors.amount}>
            <Input type="number" step="0.01" min="0" value={amount} onChange={(e) => setAmount(e.target.value)} />
          </Field>
          <Field label="Currency">
            <Select value={currency} onChange={(e) => setCurrency(e.target.value)} options={CURRENCY_OPTIONS} />
          </Field>
          <Field label="Method" required error={errors.method}>
            <Select value={method} onChange={(e) => setMethod(e.target.value)} options={PAYMENT_METHODS.map((m) => ({ value: m, label: humanize(m) }))} />
          </Field>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <Field label="Gateway">
            <Input value={gateway} onChange={(e) => setGateway(e.target.value)} placeholder="stripe, paystack…" />
          </Field>
          <Field label="Transaction ID">
            <Input value={transactionId} onChange={(e) => setTransactionId(e.target.value)} className="font-mono" />
          </Field>
          <Field label="Card last 4">
            <Input value={cardLast4} onChange={(e) => setCardLast4(e.target.value)} maxLength={4} placeholder="1234" className="font-mono" />
          </Field>
        </div>
        <Field label="Note">
          <Textarea value={note} onChange={(e) => setNote(e.target.value)} rows={2} />
        </Field>
      </form>
    </Modal>
  )
}
