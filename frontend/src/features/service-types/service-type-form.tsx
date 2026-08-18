import { useEffect, useRef, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { errorMessage } from '@/lib/api'
import { createServiceType, updateServiceType, type CreateServiceTypeRequest, type UpdateServiceTypeRequest, type ServiceTypeView } from '@/services/service-types.service'
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

const UNITS = ['per_item', 'per_kg', 'per_bundle', 'flat'] as const

interface ServiceTypeFormProps {
  open: boolean
  item: ServiceTypeView | null
  onClose: () => void
}

export function ServiceTypeForm({ open, item, onClose }: ServiceTypeFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const shopCurrency = useShopCurrency()
  // Latest shop currency without re-running the reset effect when settings load.
  const shopCurrencyRef = useRef(shopCurrency)
  shopCurrencyRef.current = shopCurrency
  const isEdit = !!item

  const [name, setName] = useState('')
  const [code, setCode] = useState('')
  const [description, setDescription] = useState('')
  const [unit, setUnit] = useState('per_item')
  const [price, setPrice] = useState('')
  const [currency, setCurrency] = useState(shopCurrency)
  const [taxRate, setTaxRate] = useState('0')
  const [days, setDays] = useState('2')
  const [icon, setIcon] = useState('')
  const [isActive, setIsActive] = useState(true)
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setName(item?.name ?? '')
    setCode(item?.code ?? '')
    setDescription(item?.description ?? '')
    setUnit(item?.unit ?? 'per_item')
    setPrice(item ? String(item.price) : '')
    setCurrency(item?.currency ?? shopCurrencyRef.current)
    setTaxRate(item ? String(item.tax_rate) : '0')
    setDays(item ? String(item.estimated_days) : '2')
    setIcon(item?.icon ?? '')
    setIsActive(item?.is_active ?? true)
    setErrors({})
  }, [open, item])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && item) {
        const req: UpdateServiceTypeRequest = {
          id: item.id,
          name: name.trim(),
          description: description.trim() || undefined,
          unit,
          price: Number(price),
          currency,
          tax_rate: Number(taxRate),
          estimated_days: Number(days),
          is_active: isActive,
          icon: icon.trim() || undefined,
        }
        return updateServiceType(req)
      }
      const req: CreateServiceTypeRequest = {
        name: name.trim(),
        code: code.trim(),
        description: description.trim(),
        unit,
        price: Number(price),
        currency,
        tax_rate: Number(taxRate),
        estimated_days: Number(days),
        is_active: isActive,
        icon: icon.trim(),
      }
      return createServiceType(req)
    },
    onSuccess: () => {
      toast.toast({ title: isEdit ? 'Service type updated' : 'Service type created', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['service-types'] })
      void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
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
    setErrors({})
    void mutation.mutate()
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? `Edit ${item?.name}` : 'New service type'}
      description="Define a priced service your operation offers."
      size="lg"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Create service type'}
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
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <Field label="Unit" required error={errors.unit}>
            <Select value={unit} onChange={(e) => setUnit(e.target.value)} options={UNITS.map((u) => ({ value: u, label: u.replace('_', ' ') }))} />
          </Field>
          <Field label="Price" required error={errors.price}>
            <Input type="number" step="0.01" min="0" value={price} onChange={(e) => setPrice(e.target.value)} />
          </Field>
          <Field label="Currency">
            <Select value={currency} onChange={(e) => setCurrency(e.target.value)} options={CURRENCY_OPTIONS} />
          </Field>
        </div>
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <Field label="Tax rate %">
            <Input type="number" step="0.1" min="0" max="100" value={taxRate} onChange={(e) => setTaxRate(e.target.value)} />
          </Field>
          <Field label="Est. days">
            <Input type="number" min="0" value={days} onChange={(e) => setDays(e.target.value)} />
          </Field>
          <Field label="Icon (emoji)">
            <Input value={icon} onChange={(e) => setIcon(e.target.value)} placeholder="🧺" className="text-center" />
          </Field>
        </div>
        <label className="flex cursor-pointer items-center gap-2.5">
          <input type="checkbox" checked={isActive} onChange={(e) => setIsActive(e.target.checked)} className="size-4 accent-brand-600" />
          <span className="text-xs font-medium text-surface-700">Active — bookable by customers</span>
        </label>
      </form>
    </Modal>
  )
}
