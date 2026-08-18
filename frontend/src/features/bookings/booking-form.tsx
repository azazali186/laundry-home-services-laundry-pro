import { useEffect, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@/lib/api'
import { fetchAllPages } from '@/lib/fetchAll'
import { createBooking, updateBooking, type CreateBookingRequest, type UpdateBookingRequest, type BookingView } from '@/services/bookings.service'
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
  customer_id: z.string().min(1, 'Customer is required'),
  service_type_id: z.string().min(1, 'Service type is required'),
  pickup_address: z.string().min(5, 'Pickup address is required'),
  estimated_amount: z.coerce.number().positive('Enter a valid amount'),
})

interface BookingFormProps {
  open: boolean
  booking: BookingView | null
  onClose: () => void
}

export function BookingForm({ open, booking, onClose }: BookingFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const serviceTypes = useServiceTypeOptions()
  const shopCurrency = useShopCurrency()
  const isEdit = !!booking

  const customersQuery = useQuery({
    queryKey: ['users', 'all'],
    queryFn: () => fetchAllPages<UserView>(listUsers, { limit: 100, sort_by: 'name', maxPages: 10 }),
    staleTime: 5 * 60_000,
  })

  const [customerId, setCustomerId] = useState('')
  const [serviceTypeId, setServiceTypeId] = useState('')
  const [pickupAddress, setPickupAddress] = useState('')
  const [dropoffAddress, setDropoffAddress] = useState('')
  const [pickupFrom, setPickupFrom] = useState('')
  const [pickupTo, setPickupTo] = useState('')
  const [notes, setNotes] = useState('')
  const [amount, setAmount] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setCustomerId(booking?.customer_id ?? '')
    setServiceTypeId(booking?.service_type_id ?? '')
    setPickupAddress(booking?.pickup_address ?? '')
    setDropoffAddress(booking?.dropoff_address ?? '')
    setPickupFrom(booking?.pickup_from ? booking.pickup_from.slice(0, 16) : '')
    setPickupTo(booking?.pickup_to ? booking.pickup_to.slice(0, 16) : '')
    setNotes(booking?.notes ?? '')
    setAmount(booking ? String(booking.estimated_amount) : '')
    setErrors({})
  }, [open, booking])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && booking) {
        const req: UpdateBookingRequest = {
          id: booking.id,
          pickup_address: pickupAddress.trim(),
          dropoff_address: dropoffAddress.trim() || undefined,
          pickup_from: pickupFrom ? new Date(pickupFrom).toISOString() : null,
          pickup_to: pickupTo ? new Date(pickupTo).toISOString() : null,
          notes: notes.trim() || undefined,
          estimated_amount: Number(amount),
        }
        return updateBooking(req)
      }
      const req: CreateBookingRequest = {
        customer_id: customerId,
        service_type_id: serviceTypeId,
        pickup_address: pickupAddress.trim(),
        dropoff_address: dropoffAddress.trim(),
        pickup_from: pickupFrom ? new Date(pickupFrom).toISOString() : undefined,
        pickup_to: pickupTo ? new Date(pickupTo).toISOString() : undefined,
        notes: notes.trim(),
        estimated_amount: Number(amount),
        currency: serviceTypes.data?.find((s) => s.id === serviceTypeId)?.currency ?? shopCurrency,
        source: 'admin',
      }
      return createBooking(req)
    },
    onSuccess: () => {
      toast.toast({ title: isEdit ? 'Booking updated' : 'Booking created', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['bookings'] })
      onClose()
    },
    onError: (err) => toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' }),
  })

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = schema.safeParse({ customer_id: customerId, service_type_id: serviceTypeId, pickup_address: pickupAddress, estimated_amount: amount })
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
      title={isEdit ? `Edit ${booking?.booking_no}` : 'New booking'}
      description={isEdit ? 'Update booking details.' : 'Create a pickup booking for a customer.'}
      size="lg"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Create booking'}
          </Button>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Customer" required error={errors.customer_id}>
            <Select value={customerId} onChange={(e) => setCustomerId(e.target.value)} disabled={isEdit}>
              <option value="">Select customer…</option>
              {customersQuery.data?.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.name} — {u.email}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Service type" required error={errors.service_type_id}>
            <Select
              value={serviceTypeId}
              onChange={(e) => {
                setServiceTypeId(e.target.value)
                const st = serviceTypes.data?.find((s) => s.id === e.target.value)
                if (st && !isEdit) setAmount(String(st.price))
              }}
            >
              <option value="">Select service…</option>
              {serviceTypes.data?.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name} — {formatCurrency(s.price, s.currency)}
                </option>
              ))}
            </Select>
          </Field>
        </div>

        <Field label="Pickup address" required error={errors.pickup_address}>
          <Input value={pickupAddress} onChange={(e) => setPickupAddress(e.target.value)} placeholder="Street, city, region" />
        </Field>
        <Field label="Dropoff address">
          <Input value={dropoffAddress} onChange={(e) => setDropoffAddress(e.target.value)} />
        </Field>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Pickup window from">
            <Input type="datetime-local" value={pickupFrom} onChange={(e) => setPickupFrom(e.target.value)} />
          </Field>
          <Field label="Pickup window to">
            <Input type="datetime-local" value={pickupTo} onChange={(e) => setPickupTo(e.target.value)} />
          </Field>
        </div>

        <Field label="Estimated amount" required error={errors.estimated_amount}>
          <Input type="number" step="0.01" min="0" value={amount} onChange={(e) => setAmount(e.target.value)} />
        </Field>
        <Field label="Notes">
          <Textarea value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} />
        </Field>
      </form>
    </Modal>
  )
}
