import { useEffect, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@/lib/api'
import {
  createGarment,
  updateGarment,
  GARMENT_STATUSES,
  type CreateGarmentRequest,
  type UpdateGarmentRequest,
  type GarmentView,
} from '@/services/garments.service'
import { useOrderOptions } from '@/hooks/useOrderOptions'
import { humanize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Field, Input, Select, Textarea } from '@/components/ui/form'
import { Modal } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'

const schema = z.object({
  order_id: z.string().min(1, 'Order is required'),
  name: z.string().min(1, 'Garment name is required'),
  quantity: z.coerce.number().int().min(1, 'Quantity must be at least 1'),
})

interface GarmentFormProps {
  open: boolean
  garment: GarmentView | null
  onClose: () => void
}

export function GarmentForm({ open, garment, onClose }: GarmentFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const orders = useOrderOptions()
  const isEdit = !!garment

  const [orderId, setOrderId] = useState('')
  const [name, setName] = useState('')
  const [type, setType] = useState('')
  const [color, setColor] = useState('')
  const [fabric, setFabric] = useState('')
  const [quantity, setQuantity] = useState('1')
  const [status, setStatus] = useState<string>('received')
  const [instructions, setInstructions] = useState('')
  const [damageNotes, setDamageNotes] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setOrderId(garment?.order_id ?? '')
    setName(garment?.name ?? '')
    setType(garment?.type ?? '')
    setColor(garment?.color ?? '')
    setFabric(garment?.fabric ?? '')
    setQuantity(garment ? String(garment.quantity) : '1')
    setStatus(garment?.status ?? 'received')
    setInstructions(garment?.instructions ?? '')
    setDamageNotes(garment?.damage_notes ?? '')
    setErrors({})
  }, [open, garment])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && garment) {
        const req: UpdateGarmentRequest = {
          id: garment.id,
          name: name.trim(),
          type: type.trim() || undefined,
          color: color.trim() || undefined,
          fabric: fabric.trim() || undefined,
          quantity: Number(quantity),
          status,
          instructions: instructions.trim() || undefined,
          damage_notes: damageNotes.trim() || undefined,
        }
        return updateGarment(req)
      }
      const req: CreateGarmentRequest = {
        order_id: orderId,
        name: name.trim(),
        type: type.trim(),
        color: color.trim(),
        fabric: fabric.trim(),
        quantity: Number(quantity),
        instructions: instructions.trim() || undefined,
        damage_notes: damageNotes.trim() || undefined,
      }
      return createGarment(req)
    },
    onSuccess: () => {
      toast.toast({ title: isEdit ? 'Garment updated' : 'Garment added', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['garments'] })
      void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      onClose()
    },
    onError: (err) => toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' }),
  })

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = schema.safeParse({ order_id: orderId, name, quantity })
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
      title={isEdit ? `Edit ${garment?.name}` : 'Add garment'}
      description={isEdit ? 'Update garment details and status.' : 'Track a garment against an order.'}
      size="lg"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Add garment'}
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

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <Field label="Name" required error={errors.name}>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Blue shirt" />
          </Field>
          <Field label="Type">
            <Input value={type} onChange={(e) => setType(e.target.value)} placeholder="e.g. shirt" />
          </Field>
          <Field label="Fabric">
            <Input value={fabric} onChange={(e) => setFabric(e.target.value)} placeholder="e.g. cotton" />
          </Field>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <Field label="Color">
            <Input value={color} onChange={(e) => setColor(e.target.value)} placeholder="e.g. navy" />
          </Field>
          <Field label="Quantity" required error={errors.quantity}>
            <Input type="number" min="1" step="1" value={quantity} onChange={(e) => setQuantity(e.target.value)} />
          </Field>
          {isEdit && (
            <Field label="Status">
              <Select value={status} onChange={(e) => setStatus(e.target.value)}>
                {GARMENT_STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {humanize(s)}
                  </option>
                ))}
              </Select>
            </Field>
          )}
        </div>

        <Field label="Instructions">
          <Textarea value={instructions} onChange={(e) => setInstructions(e.target.value)} rows={2} placeholder="Care instructions, special handling…" />
        </Field>
        <Field label="Damage notes">
          <Textarea value={damageNotes} onChange={(e) => setDamageNotes(e.target.value)} rows={2} placeholder="Pre-existing damage or stains…" />
        </Field>
      </form>
    </Modal>
  )
}
