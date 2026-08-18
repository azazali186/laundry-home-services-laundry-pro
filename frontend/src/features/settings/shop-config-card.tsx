import { useEffect, useMemo, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Store } from 'lucide-react'
import { errorMessage } from '@/lib/api'
import {
  CURRENCY_OPTIONS,
  TIMEZONE_OPTIONS,
  fetchShopSettings,
  updateShopSettings,
} from '@/services/settings.service'
import { Button } from '@/components/ui/button'
import { Card, CardBody, CardHeader } from '@/components/ui/card'
import { Field, Input, Select, Textarea } from '@/components/ui/form'
import { ErrorState, Skeleton } from '@/components/ui/feedback'
import { useToast } from '@/components/ui/toast'
import { formatDate } from '@/lib/format'

const schema = z.object({
  business_name: z.string().trim().min(1, 'Business name is required'),
  business_phone: z.string(),
  business_email: z.string().refine((v) => v === '' || v.includes('@'), 'Enter a valid email'),
  address: z.string(),
  city: z.string(),
  currency: z.string().regex(/^[A-Z]{3}$/, '3-letter currency code'),
  tax_rate: z.coerce
    .number()
    .min(0, 'Minimum 0%')
    .max(100, 'Maximum 100%'),
  tax_id: z.string(),
  invoice_prefix: z.string().trim().min(1, 'Prefix is required'),
  receipt_footer: z.string(),
  timezone: z.string().min(1, 'Timezone is required'),
})

type FormValues = z.infer<typeof schema>

const emptyForm: FormValues = {
  business_name: '',
  business_phone: '',
  business_email: '',
  address: '',
  city: '',
  currency: 'INR',
  tax_rate: 0,
  tax_id: '',
  invoice_prefix: 'INV',
  receipt_footer: '',
  timezone: 'Asia/Kolkata',
}

export function ShopConfigCard() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const [form, setForm] = useState<FormValues>(emptyForm)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [loaded, setLoaded] = useState(false)

  const query = useQuery({
    queryKey: ['settings'],
    queryFn: fetchShopSettings,
    staleTime: 60_000,
  })

  useEffect(() => {
    if (!query.data || loaded) return
    setForm({
      business_name: query.data.business_name,
      business_phone: query.data.business_phone ?? '',
      business_email: query.data.business_email ?? '',
      address: query.data.address ?? '',
      city: query.data.city ?? '',
      currency: query.data.currency,
      tax_rate: query.data.tax_rate,
      tax_id: query.data.tax_id ?? '',
      invoice_prefix: query.data.invoice_prefix,
      receipt_footer: query.data.receipt_footer ?? '',
      timezone: query.data.timezone,
    })
    setLoaded(true)
  }, [query.data, loaded])

  const mutation = useMutation({
    mutationFn: () =>
      updateShopSettings({
        business_name: form.business_name.trim(),
        business_phone: form.business_phone.trim(),
        business_email: form.business_email.trim(),
        address: form.address.trim(),
        city: form.city.trim(),
        currency: form.currency,
        tax_rate: form.tax_rate,
        tax_id: form.tax_id.trim(),
        invoice_prefix: form.invoice_prefix.trim().toUpperCase(),
        receipt_footer: form.receipt_footer.trim(),
        timezone: form.timezone,
      }),
    onSuccess: (updated) => {
      toast.toast({ title: 'Shop configuration saved', variant: 'success' })
      setForm(formFromView(updated))
      void queryClient.setQueryData(['settings'], updated)
    },
    onError: (err) => {
      toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' })
    },
  })

  const dirty = useMemo(() => {
    const current: FormValues = {
      business_name: form.business_name.trim(),
      business_phone: form.business_phone.trim(),
      business_email: form.business_email.trim(),
      address: form.address.trim(),
      city: form.city.trim(),
      currency: form.currency,
      tax_rate: form.tax_rate,
      tax_id: form.tax_id.trim(),
      invoice_prefix: form.invoice_prefix.trim().toUpperCase(),
      receipt_footer: form.receipt_footer.trim(),
      timezone: form.timezone,
    }
    return query.data !== undefined && JSON.stringify(current) !== JSON.stringify(formFromView(query.data))
  }, [form, query.data])

  function set<K extends keyof FormValues>(key: K, value: FormValues[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
    setErrors((prev) => (prev[key] ? { ...prev, [key]: '' } : prev))
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = schema.safeParse(form)
    if (!result.success) {
      const fieldErrors: Record<string, string> = {}
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as string] = issue.message
      }
      setErrors(fieldErrors)
      return
    }
    setErrors({})
    void mutation.mutate()
  }

  if (query.isError) {
    return (
      <Card>
        <CardHeader title="Shop configuration" />
        <CardBody>
          <ErrorState
            message="Couldn't load shop configuration."
            detail={errorMessage(query.error)}
            onRetry={() => void query.refetch()}
          />
        </CardBody>
      </Card>
    )
  }

  if (query.isLoading) {
    return (
      <Card>
        <CardHeader title="Shop configuration" />
        <CardBody>
          <div className="space-y-4">
            <Skeleton className="h-9 w-full" />
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Skeleton className="h-9" />
              <Skeleton className="h-9" />
            </div>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <Skeleton className="h-9" />
              <Skeleton className="h-9" />
              <Skeleton className="h-9" />
            </div>
            <Skeleton className="h-9 w-32" />
          </div>
        </CardBody>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader
        title={
          <span className="inline-flex items-center gap-2">
            <Store className="size-4 text-surface-400" /> Shop configuration
          </span>
        }
        description="Business identity, default currency and the default tax rate applied to new orders."
        action={
          query.data?.updated_at ? (
            <span className="hidden text-xs text-surface-400 sm:block">
              Last updated {formatDate(query.data.updated_at)}
            </span>
          ) : undefined
        }
      />
      <CardBody>
        <form onSubmit={onSubmit} className="space-y-5" noValidate>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Business name" required error={errors.business_name}>
              <Input
                value={form.business_name}
                onChange={(e) => set('business_name', e.target.value)}
                placeholder="Sparkle Laundry"
                autoFocus
              />
            </Field>
            <Field label="Business phone" error={errors.business_phone}>
              <Input
                value={form.business_phone}
                onChange={(e) => set('business_phone', e.target.value)}
                placeholder="+91 98765 43210"
              />
            </Field>
            <Field label="Business email" error={errors.business_email}>
              <Input
                type="email"
                value={form.business_email}
                onChange={(e) => set('business_email', e.target.value)}
                placeholder="hello@sparklelaundry.com"
              />
            </Field>
            <Field label="City" error={errors.city}>
              <Input value={form.city} onChange={(e) => set('city', e.target.value)} placeholder="Mumbai" />
            </Field>
          </div>

          <Field label="Address" error={errors.address}>
            <Textarea
              value={form.address}
              onChange={(e) => set('address', e.target.value)}
              placeholder="Store or service address shown on invoices"
              rows={2}
            />
          </Field>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <Field label="Default currency" required error={errors.currency}>
              <Select
                options={CURRENCY_OPTIONS}
                value={form.currency}
                onChange={(e) => set('currency', e.target.value.toUpperCase())}
              />
            </Field>
            <Field label="Default tax rate (%)" required error={errors.tax_rate} hint="Applied to new orders unless overridden.">
              <Input
                type="number"
                min={0}
                max={100}
                step={0.1}
                value={form.tax_rate}
                onChange={(e) => set('tax_rate', e.target.value === '' ? 0 : Number(e.target.value))}
              />
            </Field>
            <Field label="Tax ID" error={errors.tax_id} hint="e.g. GST / VAT registration number.">
              <Input value={form.tax_id} onChange={(e) => set('tax_id', e.target.value)} placeholder="27ABCDE1234F1Z5" />
            </Field>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Invoice prefix" required error={errors.invoice_prefix} hint="Auto-normalized to uppercase.">
              <Input
                value={form.invoice_prefix}
                onChange={(e) => set('invoice_prefix', e.target.value)}
                placeholder="INV"
                maxLength={12}
              />
            </Field>
            <Field label="Timezone" required error={errors.timezone}>
              <Select options={TIMEZONE_OPTIONS} value={form.timezone} onChange={(e) => set('timezone', e.target.value)} />
            </Field>
          </div>

          <Field label="Receipt footer" error={errors.receipt_footer} hint="Shown at the bottom of receipts and invoices.">
            <Textarea
              value={form.receipt_footer}
              onChange={(e) => set('receipt_footer', e.target.value)}
              placeholder="Thank you for your business!"
              rows={2}
            />
          </Field>

          <div className="flex items-center gap-3 border-t border-surface-100 pt-4">
            <Button type="submit" loading={mutation.isPending} disabled={!dirty}>
              Save changes
            </Button>
            {!dirty && <span className="text-xs text-surface-400">No unsaved changes</span>}
          </div>
        </form>
      </CardBody>
    </Card>
  )
}

/** Project a view back to the normalized form shape for dirty comparison. */
function formFromView(view: {
  business_name: string
  business_phone?: string
  business_email?: string
  address?: string
  city?: string
  currency: string
  tax_rate: number
  tax_id?: string
  invoice_prefix: string
  receipt_footer?: string
  timezone: string
}): FormValues {
  return {
    business_name: view.business_name,
    business_phone: view.business_phone ?? '',
    business_email: view.business_email ?? '',
    address: view.address ?? '',
    city: view.city ?? '',
    currency: view.currency,
    tax_rate: view.tax_rate,
    tax_id: view.tax_id ?? '',
    invoice_prefix: view.invoice_prefix,
    receipt_footer: view.receipt_footer ?? '',
    timezone: view.timezone,
  }
}
