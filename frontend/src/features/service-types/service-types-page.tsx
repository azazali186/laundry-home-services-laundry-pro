import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { listServiceTypes, deleteServiceType, type ServiceTypeSummary, type ServiceTypeView } from '@/services/service-types.service'
import { errorMessage } from '@/lib/api'
import { formatCurrency, formatNumber } from '@/lib/format'
import { humanize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { StatusBadge } from '@/components/ui/badge'
import { PageHeader } from '@/components/ui/page-header'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { ConfirmDialog } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'
import { ServiceTypeForm } from './service-type-form'

export function ServiceTypesPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<ServiceTypeView | null>(null)
  const [deleting, setDeleting] = useState<ServiceTypeView | null>(null)

  const list = useListQuery<ServiceTypeView, ServiceTypeSummary>({ queryKey: ['service-types'], fetcher: listServiceTypes, defaultSortBy: 'name', defaultSortOrder: 'asc' })

  const deleteMut = useMutation({
    mutationFn: () => deleteServiceType(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'Service type deleted', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['service-types'] })
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  // Currency is a shop setting; fall back to the first row's currency for
  // the money summary chips (never hardcode a shop currency here).
  const shopCurrency = useShopCurrency()
  const currency = list.items[0]?.currency ?? shopCurrency
  const canWrite = can('POST /service-types')

  const columns: Column<ServiceTypeView>[] = [
    {
      key: 'name',
      header: 'Service',
      sortable: true,
      render: (s) => (
        <div className="flex items-center gap-3">
          <span className="flex size-9 items-center justify-center rounded-xl bg-surface-100 text-lg">
            {s.icon || '🧺'}
          </span>
          <div className="min-w-0">
            <p className="font-semibold text-surface-800">{s.name}</p>
            <p className="font-mono text-[11px] text-surface-400">{s.code}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'unit',
      header: 'Unit',
      filterOptions: ['per_item', 'per_kg', 'per_bundle', 'flat'].map((u) => ({ value: u, label: humanize(u) })),
      hideOnMobile: true,
      render: (s) => <span className="text-xs text-surface-500">{humanize(s.unit)}</span>,
    },
    {
      key: 'price',
      header: 'Price',
      sortable: true,
      align: 'right',
      render: (s) => (
        <span className="font-semibold tabular-nums text-surface-800">{formatCurrency(s.price, s.currency)}</span>
      ),
    },
    {
      key: 'tax_rate',
      header: 'Tax',
      sortable: true,
      align: 'right',
      hideOnMobile: true,
      render: (s) => <span className="tabular-nums text-xs">{s.tax_rate}%</span>,
    },
    {
      key: 'estimated_days',
      header: 'Est. days',
      sortable: true,
      align: 'right',
      hideOnMobile: true,
      render: (s) => <span className="tabular-nums text-xs">{s.estimated_days}</span>,
    },
    {
      key: 'is_active',
      header: 'Status',
      sortable: true,
      filterOptions: [
        { value: 'true', label: 'Active' },
        { value: 'false', label: 'Inactive' },
      ],
      render: (s) => <StatusBadge status={s.is_active ? 'Active' : 'Inactive'} />,
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Service types"
        description="Pricing and availability of your services."
        actions={
          canWrite && (
            <Button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              <Plus className="size-4" /> New service
            </Button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(s) => s.id}
        searchPlaceholder="Search services…"
        emptyTitle="No service types"
        emptyMessage="Create your first service to start taking bookings."
        summary={
          summary && (
            <SummaryBar
              chips={[
                { label: 'Active', value: formatNumber(summary.total_active), tone: 'emerald' },
                { label: 'Inactive', value: formatNumber(summary.total_inactive) },
                { label: 'Avg price', value: formatCurrency(summary.avg_price, currency), tone: 'brand' },
                { label: 'Max', value: formatCurrency(summary.max_price, currency) },
                { label: 'Min', value: formatCurrency(summary.min_price, currency) },
              ]}
            />
          )
        }
        rowActions={
          canWrite
            ? (s) => [
                {
                  key: 'edit',
                  label: 'Edit',
                  icon: <Pencil className="size-3.5" />,
                  onClick: () => {
                    setEditing(s)
                    setFormOpen(true)
                  },
                },
                {
                  key: 'delete',
                  label: 'Delete',
                  icon: <Trash2 className="size-3.5" />,
                  danger: true,
                  onClick: () => setDeleting(s),
                },
              ]
            : undefined
        }
      />

      <ServiceTypeForm open={formOpen} item={editing} onClose={() => setFormOpen(false)} />

      <ConfirmDialog
        open={!!deleting}
        title="Delete service type?"
        message={`Delete ${deleting?.name}? Existing orders keep their records.`}
        confirmLabel="Delete"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => void deleteMut.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
