import { useEffect, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@/lib/api'
import { fetchAllPages } from '@/lib/fetchAll'
import { createOrder, updateOrder, type CreateOrderRequest, type UpdateOrderRequest, type OrderView } from '@/services/orders.service'
import { listBookings, type BookingView } from '@/services/bookings.service'
import { listUsers } from '@/services/users.service'
import type { UserView } from '@/services/auth.service'
import { useServiceTypeOptions } from '@/hooks/useServiceTypeOptions'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { formatCurrency } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Field, Input, Select, Textarea } from '@/components/ui/form'
import { Modal } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'

const schema = z.object({
  booking_id: z.string().min(1, 'Booking is required'),
  customer_id: z.string().min(1, 'Customer is required'),
  service_type_id: z.string().min(1, 'Service type is required'),
  item_count: z.coerce.number().int().min(1, 'At least 1 item'),
  total_amount: z.coerce.number().positive('Enter a valid amount'),
})

interface OrderFormProps {
  open: boolean
  order: OrderView | null
  onClose: () => void
}

export function OrderForm({ open, order, onClose }: OrderFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const serviceTypes = useServiceTypeOptions()
  const shopCurrency = useShopCurrency()
  const isEdit = !!order

  const bookingsQuery = useQuery({
    queryKey: ['bookings', 'all'],
    queryFn: () => fetchAllPages<BookingView>(listBookings, { limit: 100, sort_by: 'booking_no', maxPages: 10 }),
    staleTime: 5 * 60_000,
  })
  const customersQuery = useQuery({
    queryKey: ['users', 'all'],
    queryFn: () => fetchAllPages<UserView>(listUsers, { limit: 100, sort_by: 'name', maxPages: 10 }),
    staleTime: 5 * 60_000,
  })

  const [bookingId, setBookingId] = useState('')
  const [customerId, setCustomerId] = useState('')
  const [serviceTypeId, setServiceTypeId] = useState('')
  const [priority, setPriority] = useState('normal')
  const [itemCount, setItemCount] = useState('')
  const [totalAmount, setTotalAmount] = useState('')
  const [note, setNote] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setBookingId(order?.booking_id ?? '')
    setCustomerId(order?.customer_id ?? '')
    setServiceTypeId(order?.service_type_id ?? '')
    setPriority(order?.priority ?? 'normal')
    setItemCount(order ? String(order.item_count) : '')
    setTotalAmount(order ? String(order.total_amount) : '')
    setNote(order?.note ?? '')
    setErrors({})
  }, [open, order])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && order) {
        const req: UpdateOrderRequest = {
          id: order.id,
          priority,
          item_count: Number(itemCount),
          total_amount: Number(totalAmount),
          note: note.trim() || undefined,
        }
        return updateOrder(req)
      }
      const req: CreateOrderRequest = {
        booking_id: bookingId,
        customer_id: customerId,
        service_type_id: serviceTypeId,
        priority,
        item_count: Number(itemCount),
        total_amount: Number(totalAmount),
        currency: serviceTypes.data?.find((s) => s.id === serviceTypeId)?.currency ?? shopCurrency,
        note: note.trim(),
      }
      return createOrder(req)
    },
    onSuccess: () => {
      toast.toast({ title: isEdit ? 'Order updated' : 'Order created', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['orders'] })
      void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      onClose()
    },
    onError: (err) => toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' }),
  })

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = schema.safeParse({ booking_id: bookingId, customer_id: customerId, service_type_id: serviceTypeId, item_count: itemCount, total_amount: totalAmount })
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
      title={isEdit ? `Edit ${order?.order_no}` : 'New order'}
      description={isEdit ? 'Update order details.' : 'Create an order from a confirmed booking.'}
      size="lg"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Create order'}
          </Button>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Booking" required error={errors.booking_id}>
            <Select value={bookingId} onChange={(e) => setBookingId(e.target.value)} disabled={isEdit}>
              <option value="">Select booking…</option>
              {bookingsQuery.data
                ?.filter((b) => b.status !== 'cancelled')
                .map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.booking_no}
                  </option>
                ))}
            </Select>
          </Field>
          <Field label="Customer" required error={errors.customer_id}>
            <Select value={customerId} onChange={(e) => setCustomerId(e.target.value)} disabled={isEdit}>
              <option value="">Select customer…</option>
              {customersQuery.data?.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.name}
                </option>
              ))}
            </Select>
          </Field>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Service type" required error={errors.service_type_id}>
            <Select value={serviceTypeId} onChange={(e) => setServiceTypeId(e.target.value)} disabled={isEdit}>
              <option value="">Select service…</option>
              {serviceTypes.data?.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name} — {formatCurrency(s.price, s.currency)}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Priority">
            <Select value={priority} onChange={(e) => setPriority(e.target.value)}>
              <option value="normal">Normal</option>
              <option value="express">Express</option>
              <option value="priority">Priority</option>
            </Select>
          </Field>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Item count" required error={errors.item_count}>
            <Input type="number" min="1" value={itemCount} onChange={(e) => setItemCount(e.target.value)} />
          </Field>
          <Field label="Total amount" required error={errors.total_amount}>
            <Input type="number" step="0.01" min="0" value={totalAmount} onChange={(e) => setTotalAmount(e.target.value)} />
          </Field>
        </div>
        <Field label="Note">
          <Textarea value={note} onChange={(e) => setNote(e.target.value)} rows={2} />
        </Field>
      </form>
    </Modal>
  )
}
