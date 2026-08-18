import { useEffect, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@/lib/api'
import {
  createDelivery,
  updateDelivery,
  type CreateDeliveryRequest,
  type UpdateDeliveryRequest,
  type DeliveryView,
} from '@/services/deliveries.service'
import { useOrderOptions } from '@/hooks/useOrderOptions'
import { Button } from '@/components/ui/button'
import { Field, Input, Select, Textarea } from '@/components/ui/form'
import { Modal } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'

const schema = z.object({
  order_id: z.string().min(1, 'Order is required'),
  scheduled_at: z.string().min(1, 'Scheduled time is required'),
  address: z.string().min(5, 'Delivery address is required'),
})

interface DeliveryFormProps {
  open: boolean
  delivery: DeliveryView | null
  onClose: () => void
}

export function DeliveryForm({ open, delivery, onClose }: DeliveryFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const orders = useOrderOptions()
  const isEdit = !!delivery

  const [orderId, setOrderId] = useState('')
  const [scheduledAt, setScheduledAt] = useState('')
  const [address, setAddress] = useState('')
  const [notes, setNotes] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setOrderId(delivery?.order_id ?? '')
    setScheduledAt(delivery?.scheduled_at ? delivery.scheduled_at.slice(0, 16) : '')
    setAddress(delivery?.address ?? '')
    setNotes(delivery?.notes ?? '')
    setErrors({})
  }, [open, delivery])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && delivery) {
        const req: UpdateDeliveryRequest = {
          id: delivery.id,
          scheduled_at: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
          address: address.trim(),
          notes: notes.trim() || undefined,
        }
        return updateDelivery(req)
      }
      const req: CreateDeliveryRequest = {
        order_id: orderId,
        scheduled_at: new Date(scheduledAt).toISOString(),
        address: address.trim(),
        notes: notes.trim() || undefined,
      }
      return createDelivery(req)
    },
    onSuccess: () => {
      toast.toast({ title: isEdit ? 'Delivery updated' : 'Delivery scheduled', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['deliveries'] })
      void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      onClose()
    },
    onError: (err) => toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' }),
  })

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = schema.safeParse({ order_id: orderId, scheduled_at: scheduledAt, address })
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
      title={isEdit ? 'Edit delivery' : 'Schedule delivery'}
      description={isEdit ? 'Update the delivery details.' : 'Send a ready order home.'}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Schedule delivery'}
          </Button>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <Field label="Order" required error={errors.order_id}>
          <Select value={orderId} onChange={(e) => setOrderId(e.target.value)} disabled={isEdit}>
            <option value="">Select order…</option>
            {orders.data?.map((o) => (
              <option key={o.id} value={o.id}>
                {o.order_no} — {o.status}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Scheduled time" required error={errors.scheduled_at}>
          <Input type="datetime-local" value={scheduledAt} onChange={(e) => setScheduledAt(e.target.value)} />
        </Field>
        <Field label="Delivery address" required error={errors.address}>
          <Input value={address} onChange={(e) => setAddress(e.target.value)} placeholder="Street, city, region" />
        </Field>
        <Field label="Notes">
          <Textarea value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} />
        </Field>
      </form>
    </Modal>
  )
}
