import { useEffect, useRef, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, X } from 'lucide-react'
import { errorMessage } from '@/lib/api'
import { useServiceTypeOptions } from '@/hooks/useServiceTypeOptions'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { createCombo, updateCombo, type ComboItemRequest, type ComboView } from '@/services/combos.service'
import { CURRENCY_OPTIONS } from '@/services/settings.service'
import { Button } from '@/components/ui/button'
import { Field, Input, Select, Textarea } from '@/components/ui/form'
import { Modal } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'

const schema = z.object({
  name: z.string().min(2, 'Name is required'),
  code: z.string().min(1, 'Code is required'),
  unit: z.string().min(1, 'Unit is required'),
  price: z.coerce.number().positive('Enter a valid price'),
})

const UNITS = ['per_week', 'per_month', 'per_kg', 'flat'] as const

interface ComboFormProps {
  open: boolean
  item: ComboView | null
  onClose: () => void
}

interface ItemRow {
  key: string
  serviceTypeId: string
  quantity: string
}

let rowSeq = 0
function freshRow(): ItemRow {
  rowSeq += 1
  return { key: `item-${rowSeq}`, serviceTypeId: '', quantity: '1' }
}

export function ComboForm({ open, item, onClose }: ComboFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const serviceOptions = useServiceTypeOptions()
  const shopCurrency = useShopCurrency()
  // Latest shop currency without re-running the reset effect below (which
  // would wipe in-progress item rows when settings finish loading).
  const shopCurrencyRef = useRef(shopCurrency)
  shopCurrencyRef.current = shopCurrency
  const isEdit = !!item

  // Options for the item pickers: active services, plus any deactivated
  // service already referenced by the combo being edited (so its line item
  // never silently unmaps).
  const serviceOptionsList = serviceOptions.data ?? []
  const optionValueIds = new Set(serviceOptionsList.map((s) => s.id))
  const referencedIds = new Set((item?.items ?? []).map((it) => it.service_type_id))
  const missingIds = [...referencedIds].filter((id) => !optionValueIds.has(id))
  const itemOptions = [
    ...serviceOptionsList.map((s) => ({ value: s.id, label: s.name })),
    ...missingIds.map((id) => ({ value: id, label: 'Service (inactive)' })),
  ]

  const [name, setName] = useState('')
  const [code, setCode] = useState('')
  const [description, setDescription] = useState('')
  const [unit, setUnit] = useState('per_week')
  const [price, setPrice] = useState('')
  const [currency, setCurrency] = useState(shopCurrency)
  const [isActive, setIsActive] = useState(true)
  const [isFeatured, setIsFeatured] = useState(false)
  const [validityDays, setValidityDays] = useState('')
  const [renewalType, setRenewalType] = useState('none')
  const [items, setItems] = useState<ItemRow[]>([freshRow()])
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setName(item?.name ?? '')
    setCode(item?.code ?? '')
    setDescription(item?.description ?? '')
    setUnit(item?.unit ?? 'per_week')
    setPrice(item ? String(item.price) : '')
    setCurrency(item?.currency ?? shopCurrencyRef.current)
    setIsActive(item?.is_active ?? true)
    setIsFeatured(item?.is_featured ?? false)
    setValidityDays(item?.validity_days ? String(item.validity_days) : '')
    setRenewalType(item?.renewal_type ?? 'none')
    setItems(
      item && item.items.length > 0
        ? item.items.map((it) => ({ key: `item-${rowSeq++}`, serviceTypeId: it.service_type_id, quantity: String(it.quantity) }))
        : [freshRow()],
    )
    setErrors({})
  }, [open, item])

  function updateItem(key: string, patch: Partial<ItemRow>) {
    setItems((prev) => prev.map((r) => (r.key === key ? { ...r, ...patch } : r)))
  }

  const mutation = useMutation({
    mutationFn: async () => {
      const payload = items
        .filter((r) => r.serviceTypeId)
        .map((r): ComboItemRequest => ({ service_type_id: r.serviceTypeId, quantity: Math.max(1, Number(r.quantity) || 1) }))
      const base = {
        name: name.trim(),
        description: description.trim() || undefined,
        unit,
        price: Number(price),
        currency,
        is_active: isActive,
        is_featured: isFeatured,
        validity_days: validityDays ? Number(validityDays) : 0,
        renewal_type: renewalType,
        items: payload,
      }
      if (isEdit && item) {
        return updateCombo({ ...base, id: item.id, code: code.trim() })
      }
      return createCombo({ ...base, code: code.trim() })
    },
    onSuccess: () => {
      toast.toast({ title: isEdit ? 'Combo updated' : 'Combo created', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['combos'] })
      void queryClient.invalidateQueries({ queryKey: ['public-pricing'] })
      onClose()
    },
    onError: (err) => toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' }),
  })

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = schema.safeParse({ name, code, unit, price })
    if (!result.success) {
      const fieldErrors: Record<string, string> = {}
      for (const issue of result.error.issues) fieldErrors[issue.path[0] as string] = issue.message
      setErrors(fieldErrors)
      return
    }
    if (!items.some((r) => r.serviceTypeId)) {
      setErrors({ items: 'Add at least one service to the combo' })
      return
    }
    setErrors({})
    void mutation.mutate()
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? `Edit ${item?.name}` : 'New combo'}
      description="A combo bundles services with quantities and has its own bundle price."
      size="lg"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Create combo'}
          </Button>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Name" required error={errors.name}>
            <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
          </Field>
          <Field label="Code" required error={errors.code}>
            <Input value={code} onChange={(e) => setCode(e.target.value)} className="font-mono" />
          </Field>
        </div>
        <Field label="Description">
          <Textarea value={description} onChange={(e) => setDescription(e.target.value)} rows={2} />
        </Field>

        <Field label="Included services" required error={errors.items}>
          <div className="space-y-2">
            {items.map((row) => (
              <div key={row.key} className="flex items-center gap-2">
                <Select
                  value={row.serviceTypeId}
                  onChange={(e) => updateItem(row.key, { serviceTypeId: e.target.value })}
                  options={[{ value: '', label: 'Select a service…' }, ...itemOptions]}
                  className="flex-1"
                />
                <Input
                  type="number"
                  min="1"
                  value={row.quantity}
                  onChange={(e) => updateItem(row.key, { quantity: e.target.value })}
                  className="w-20 text-center"
                  aria-label="Quantity"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="shrink-0 px-2"
                  disabled={items.length === 1}
                  onClick={() => setItems((prev) => prev.filter((r) => r.key !== row.key))}
                  aria-label="Remove service"
                >
                  <X className="size-4" />
                </Button>
              </div>
            ))}
            <Button type="button" variant="outline" size="sm" onClick={() => setItems((prev) => [...prev, freshRow()])}>
              <Plus className="size-3.5" /> Add service
            </Button>
          </div>
        </Field>

        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <Field label="Unit" required error={errors.unit}>
            <Select value={unit} onChange={(e) => setUnit(e.target.value)} options={UNITS.map((u) => ({ value: u, label: u.replace('_', ' ') }))} />
          </Field>
          <Field label="Bundle price" required error={errors.price}>
            <Input type="number" step="0.01" min="0" value={price} onChange={(e) => setPrice(e.target.value)} />
          </Field>
          <Field label="Currency">
            <Select value={currency} onChange={(e) => setCurrency(e.target.value)} options={CURRENCY_OPTIONS} />
          </Field>
        </div>

        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <Field label="Validity (days)">
            <Input type="number" min="0" placeholder="0 = no expiry" value={validityDays} onChange={(e) => setValidityDays(e.target.value)} />
          </Field>
          <Field label="Renewal">
            <Select
              value={renewalType}
              onChange={(e) => setRenewalType(e.target.value)}
              options={[
                { value: 'none', label: 'One-time' },
                { value: 'auto', label: 'Auto-renew' },
                { value: 'manual', label: 'Manual renew' },
              ]}
            />
          </Field>
        </div>

        <div className="flex items-center gap-6">
          <label className="flex cursor-pointer items-center gap-2.5">
            <input type="checkbox" checked={isActive} onChange={(e) => setIsActive(e.target.checked)} className="size-4 accent-brand-600" />
            <span className="text-xs font-medium text-surface-700">Active — shown on the public pricing page</span>
          </label>
          <label className="flex cursor-pointer items-center gap-2.5">
            <input type="checkbox" checked={isFeatured} onChange={(e) => setIsFeatured(e.target.checked)} className="size-4 accent-brand-600" />
            <span className="text-xs font-medium text-surface-700">Most Popular badge</span>
          </label>
        </div>
      </form>
    </Modal>
  )
}
