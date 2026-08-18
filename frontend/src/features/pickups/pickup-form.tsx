import { useEffect, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@/lib/api'
import {
  createPickup,
  updatePickup,
  type CreatePickupRequest,
  type UpdatePickupRequest,
  type PickupView,
} from '@/services/pickups.service'
import { useBookingOptions } from '@/hooks/useBookingOptions'
import { Button } from '@/components/ui/button'
import { Field, Input, Select, Textarea } from '@/components/ui/form'
import { Modal } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'

const schema = z.object({
  booking_id: z.string().min(1, 'Booking is required'),
  scheduled_at: z.string().min(1, 'Scheduled time is required'),
  address: z.string().min(5, 'Pickup address is required'),
})

interface PickupFormProps {
  open: boolean
  pickup: PickupView | null
  onClose: () => void
}

export function PickupForm({ open, pickup, onClose }: PickupFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const bookings = useBookingOptions()
  const isEdit = !!pickup

  const [bookingId, setBookingId] = useState('')
  const [scheduledAt, setScheduledAt] = useState('')
  const [address, setAddress] = useState('')
  const [notes, setNotes] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setBookingId(pickup?.booking_id ?? '')
    setScheduledAt(pickup?.scheduled_at ? pickup.scheduled_at.slice(0, 16) : '')
    setAddress(pickup?.address ?? '')
    setNotes(pickup?.notes ?? '')
    setErrors({})
  }, [open, pickup])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && pickup) {
        const req: UpdatePickupRequest = {
          id: pickup.id,
          scheduled_at: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
          address: address.trim(),
          notes: notes.trim() || undefined,
        }
        return updatePickup(req)
      }
      const req: CreatePickupRequest = {
        booking_id: bookingId,
        scheduled_at: new Date(scheduledAt).toISOString(),
        address: address.trim(),
        notes: notes.trim() || undefined,
      }
      return createPickup(req)
    },
    onSuccess: () => {
      toast.toast({ title: isEdit ? 'Pickup updated' : 'Pickup scheduled', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['pickups'] })
      void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      onClose()
    },
    onError: (err) => toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' }),
  })

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = schema.safeParse({ booking_id: bookingId, scheduled_at: scheduledAt, address })
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
      title={isEdit ? 'Edit pickup' : 'Schedule pickup'}
      description={isEdit ? 'Update the pickup details.' : 'Collect garments from a confirmed booking.'}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Schedule pickup'}
          </Button>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <Field label="Booking" required error={errors.booking_id}>
          <Select value={bookingId} onChange={(e) => setBookingId(e.target.value)} disabled={isEdit}>
            <option value="">Select booking…</option>
            {bookings.data?.map((b) => (
              <option key={b.id} value={b.id}>
                {b.booking_no} — {b.status}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Scheduled time" required error={errors.scheduled_at}>
          <Input type="datetime-local" value={scheduledAt} onChange={(e) => setScheduledAt(e.target.value)} />
        </Field>
        <Field label="Pickup address" required error={errors.address}>
          <Input value={address} onChange={(e) => setAddress(e.target.value)} placeholder="Street, city, region" />
        </Field>
        <Field label="Notes">
          <Textarea value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} />
        </Field>
      </form>
    </Modal>
  )
}
