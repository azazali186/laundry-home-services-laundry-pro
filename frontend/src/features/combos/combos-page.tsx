import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Layers, Pencil, Plus, Trash2 } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { listCombos, deleteCombo, type ComboSummary, type ComboView } from '@/services/combos.service'
import { errorMessage } from '@/lib/api'
import { formatCurrency, formatNumber } from '@/lib/format'
import { humanize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge, StatusBadge } from '@/components/ui/badge'
import { IncludedList } from './included-list'
import { PageHeader } from '@/components/ui/page-header'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { ConfirmDialog } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'
import { ComboForm } from './combo-form'

export function CombosPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<ComboView | null>(null)
  const [deleting, setDeleting] = useState<ComboView | null>(null)

  const list = useListQuery<ComboView, ComboSummary>({
    queryKey: ['combos'],
    fetcher: listCombos,
    defaultSortBy: 'name',
    defaultSortOrder: 'asc',
  })

  const deleteMut = useMutation({
    mutationFn: () => deleteCombo(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'Combo deleted', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['combos'] })
      void queryClient.invalidateQueries({ queryKey: ['public-pricing'] })
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  // Currency is a shop setting; fall back to the first row's currency for
  // the money summary chips (never hardcode a shop currency here).
  const shopCurrency = useShopCurrency()
  const currency = list.items[0]?.currency ?? shopCurrency
  const canWrite = can('POST /combos')

  const columns: Column<ComboView>[] = [
    {
      key: 'name',
      header: 'Combo',
      sortable: true,
      render: (c) => (
        <div className="flex items-center gap-3">
          <span className="flex size-9 items-center justify-center rounded-xl bg-brand-50 text-brand-600">
            <Layers className="size-4" />
          </span>
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <p className="font-semibold text-surface-800">{c.name}</p>
              {c.is_featured && (
                <Badge tone="violet">Popular</Badge>
              )}
            </div>
            <p className="font-mono text-[11px] text-surface-400">{c.code}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'unit',
      header: 'Unit',
      filterOptions: ['per_week', 'per_month', 'per_kg', 'flat'].map((u) => ({ value: u, label: humanize(u) })),
      hideOnMobile: true,
      render: (c) => <span className="text-xs text-surface-500">{humanize(c.unit)}</span>,
    },
    {
      key: 'items',
      header: 'Includes',
      hideOnMobile: true,
      render: (c) => <IncludedList items={c.items} />,
    },
    {
      key: 'price',
      header: 'Bundle price',
      sortable: true,
      align: 'right',
      render: (c) => (
        <div className="text-right">
          <span className="font-semibold tabular-nums text-surface-800">{formatCurrency(c.price, c.currency)}</span>
          {typeof c.regular_price === 'number' && c.regular_price > c.price && (
            <span className="ml-1.5 text-xs font-normal text-surface-400 line-through">
              {formatCurrency(c.regular_price, c.currency)}
            </span>
          )}
        </div>
      ),
    },
    {
      key: 'savings',
      header: 'Savings',
      align: 'right',
      hideOnMobile: true,
      render: (c) =>
        (c.savings ?? 0) > 0 ? (
          <Badge tone="emerald">Save {formatCurrency(c.savings, c.currency)}</Badge>
        ) : (
          <span className="text-xs text-surface-300">—</span>
        ),
    },
    {
      key: 'is_active',
      header: 'Status',
      sortable: true,
      filterOptions: [
        { value: 'true', label: 'Active' },
        { value: 'false', label: 'Inactive' },
      ],
      render: (c) => <StatusBadge status={c.is_active ? 'Active' : 'Inactive'} />,
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Combos & bundles"
        description="Priced bundles of services (weekly passes, family plans) shown on your public pricing page."
        actions={
          canWrite && (
            <Button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              <Plus className="size-4" /> New combo
            </Button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(c) => c.id}
        searchPlaceholder="Search combos…"
        emptyTitle="No combos"
        emptyMessage="Create your first combo to sell passes and bundles."
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
            ? (c) => [
                {
                  key: 'edit',
                  label: 'Edit',
                  icon: <Pencil className="size-3.5" />,
                  onClick: () => {
                    setEditing(c)
                    setFormOpen(true)
                  },
                },
                {
                  key: 'delete',
                  label: 'Delete',
                  icon: <Trash2 className="size-3.5" />,
                  danger: true,
                  onClick: () => setDeleting(c),
                },
              ]
            : undefined
        }
      />

      <ComboForm open={formOpen} item={editing} onClose={() => setFormOpen(false)} />

      <ConfirmDialog
        open={!!deleting}
        title="Delete combo?"
        message={`Delete ${deleting?.name}? It will disappear from the public pricing page.`}
        confirmLabel="Delete"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => void deleteMut.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
