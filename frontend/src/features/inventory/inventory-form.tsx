import { useEffect, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@/lib/api'
import { createInventory, updateInventory, type CreateInventoryRequest, type UpdateInventoryRequest, type InventoryView } from '@/services/inventory.service'
import { Button } from '@/components/ui/button'
import { Field, Input } from '@/components/ui/form'
import { Modal } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'

const schema = z.object({
  name: z.string().min(2, 'Name is required'),
  sku: z.string().min(1, 'SKU is required'),
  category: z.string().min(1, 'Category is required'),
  quantity_on_hand: z.coerce.number().min(0, 'Cannot be negative'),
  unit: z.string().min(1, 'Unit is required'),
})

interface InventoryFormProps {
  open: boolean
  item: InventoryView | null
  onClose: () => void
}

export function InventoryForm({ open, item, onClose }: InventoryFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const isEdit = !!item

  const [name, setName] = useState('')
  const [sku, setSku] = useState('')
  const [category, setCategory] = useState('')
  const [qty, setQty] = useState('')
  const [reorder, setReorder] = useState('0')
  const [unit, setUnit] = useState('units')
  const [cost, setCost] = useState('0')
  const [isActive, setIsActive] = useState(true)
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setName(item?.name ?? '')
    setSku(item?.sku ?? '')
    setCategory(item?.category ?? '')
    setQty(item ? String(item.quantity_on_hand) : '')
    setReorder(item ? String(item.reorder_level) : '0')
    setUnit(item?.unit ?? 'units')
    setCost(item ? String(item.unit_cost) : '0')
    setIsActive(item?.is_active ?? true)
    setErrors({})
  }, [open, item])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && item) {
        const req: UpdateInventoryRequest = {
          id: item.id,
          name: name.trim(),
          category: category.trim(),
          reorder_level: Number(reorder),
          unit: unit.trim(),
          unit_cost: Number(cost),
          is_active: isActive,
        }
        return updateInventory(req)
      }
      const req: CreateInventoryRequest = {
        name: name.trim(),
        sku: sku.trim(),
        category: category.trim(),
        quantity_on_hand: Number(qty),
        reorder_level: Number(reorder),
        unit: unit.trim(),
        unit_cost: Number(cost),
        is_active: isActive,
      }
      return createInventory(req)
    },
    onSuccess: () => {
      toast.toast({ title: isEdit ? 'Item updated' : 'Item added', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['inventory'] })
      onClose()
    },
    onError: (err) => toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' }),
  })

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = schema.safeParse({ name, sku, category, quantity_on_hand: qty, unit })
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
      title={isEdit ? `Edit ${item?.name}` : 'New inventory item'}
      description={isEdit ? 'Update item details.' : 'Add a supply item to track.'}
      size="lg"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Add item'}
          </Button>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Name" required error={errors.name}>
            <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
          </Field>
          <Field label="SKU" required error={errors.sku}>
            <Input value={sku} onChange={(e) => setSku(e.target.value)} className="font-mono" />
          </Field>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Category" required error={errors.category}>
            <Input value={category} onChange={(e) => setCategory(e.target.value)} placeholder="e.g. detergent, packaging" />
          </Field>
          <Field label="Unit" required error={errors.unit}>
            <Input value={unit} onChange={(e) => setUnit(e.target.value)} placeholder="litres, kg, units" />
          </Field>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <Field label="Quantity on hand" required error={errors.quantity_on_hand}>
            <Input type="number" step="0.01" min="0" value={qty} onChange={(e) => setQty(e.target.value)} />
          </Field>
          <Field label="Reorder level">
            <Input type="number" step="0.01" min="0" value={reorder} onChange={(e) => setReorder(e.target.value)} />
          </Field>
          <Field label="Unit cost">
            <Input type="number" step="0.01" min="0" value={cost} onChange={(e) => setCost(e.target.value)} />
          </Field>
        </div>
        <label className="flex cursor-pointer items-center gap-2.5">
          <input type="checkbox" checked={isActive} onChange={(e) => setIsActive(e.target.checked)} className="size-4 accent-brand-600" />
          <span className="text-xs font-medium text-surface-700">Active item</span>
        </label>
      </form>
    </Modal>
  )
}
