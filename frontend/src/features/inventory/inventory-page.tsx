import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowDownUp, Pencil, Plus, Trash2 } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { listInventory, adjustInventory, deleteInventory, type InventorySummary, type InventoryView } from '@/services/inventory.service'
import { errorMessage } from '@/lib/api'
import { formatCurrency, formatNumber } from '@/lib/format'
import { humanize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge, StatusBadge } from '@/components/ui/badge'
import { PageHeader } from '@/components/ui/page-header'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { Modal, ConfirmDialog } from '@/components/ui/modal'
import { Field, Input } from '@/components/ui/form'
import { useToast } from '@/components/ui/toast'
import { InventoryForm } from './inventory-form'

export function InventoryPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<InventoryView | null>(null)
  const [adjusting, setAdjusting] = useState<InventoryView | null>(null)
  const [delta, setDelta] = useState('')
  const [reason, setReason] = useState('')
  const [deleting, setDeleting] = useState<InventoryView | null>(null)

  const list = useListQuery<InventoryView, InventorySummary>({ queryKey: ['inventory'], fetcher: listInventory, defaultSortBy: 'name', defaultSortOrder: 'asc' })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['inventory'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }

  const adjustMut = useMutation({
    mutationFn: () => adjustInventory({ id: adjusting!.id, delta: Number(delta), reason }),
    onSuccess: () => {
      toast.toast({ title: 'Stock adjusted', variant: 'success' })
      invalidate()
      setAdjusting(null)
      setDelta('')
      setReason('')
    },
    onError: (err) => toast.toast({ title: 'Adjustment failed', description: errorMessage(err), variant: 'error' }),
  })

  const deleteMut = useMutation({
    mutationFn: () => deleteInventory(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'Item deleted', variant: 'success' })
      invalidate()
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  const currency = useShopCurrency()
  const canWrite = can('POST /inventory')

  const columns: Column<InventoryView>[] = [
    {
      key: 'name',
      header: 'Item',
      sortable: true,
      render: (i) => (
        <div>
          <p className="font-semibold text-surface-800">{i.name}</p>
          <p className="font-mono text-[11px] text-surface-400">{i.sku}</p>
        </div>
      ),
    },
    {
      key: 'category',
      header: 'Category',
      sortable: true,
      render: (i) => <Badge tone="slate">{humanize(i.category) || '—'}</Badge>,
    },
    {
      key: 'quantity_on_hand',
      header: 'On hand',
      sortable: true,
      align: 'right',
      render: (i) => (
        <span className={`font-semibold tabular-nums ${i.low_stock ? 'text-amber-600' : 'text-surface-800'}`}>
          {formatNumber(i.quantity_on_hand)} <span className="text-[10px] font-normal text-surface-400">{i.unit}</span>
        </span>
      ),
    },
    {
      key: 'reorder_level',
      header: 'Reorder at',
      sortable: true,
      align: 'right',
      hideOnMobile: true,
      render: (i) => <span className="tabular-nums text-xs">{formatNumber(i.reorder_level)}</span>,
    },
    {
      key: 'stock_value',
      header: 'Value',
      align: 'right',
      hideOnMobile: true,
      render: (i) => <span className="tabular-nums text-xs text-surface-600">{formatCurrency(i.stock_value, currency)}</span>,
    },
    {
      key: 'is_active',
      header: 'Stock status',
      sortable: true,
      filterOptions: [
        { value: 'true', label: 'Active items' },
        { value: 'false', label: 'Inactive' },
      ],
      render: (i) => (
        <StatusBadge status={i.quantity_on_hand === 0 ? 'Out of stock' : i.low_stock ? 'Low stock' : 'In stock'} />
      ),
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Inventory"
        description="Supplies and stock levels."
        actions={
          canWrite && (
            <Button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              <Plus className="size-4" /> Add item
            </Button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(i) => i.id}
        searchPlaceholder="Search items by name or SKU…"
        emptyTitle="No inventory items"
        emptyMessage="Add supplies to track stock levels."
        summary={
          summary && (
            <SummaryBar
              chips={[
                { label: 'Total value', value: formatCurrency(summary.total_value, currency), tone: 'brand' },
                { label: 'Items', value: formatNumber(summary.total_items) },
                { label: 'Low stock', value: formatNumber(summary.low_stock), tone: 'amber' },
                { label: 'Out of stock', value: formatNumber(summary.out_of_stock), tone: 'red' },
                { label: 'Active', value: formatNumber(summary.active_items), tone: 'emerald' },
              ]}
            />
          )
        }
        rowActions={
          canWrite
            ? (i) => [
                { key: 'adjust', label: 'Adjust stock', icon: <ArrowDownUp className="size-3.5" />, onClick: () => { setAdjusting(i); setDelta(''); setReason('') } },
                { key: 'edit', label: 'Edit', icon: <Pencil className="size-3.5" />, onClick: () => { setEditing(i); setFormOpen(true) } },
                { key: 'delete', label: 'Delete', icon: <Trash2 className="size-3.5" />, danger: true, onClick: () => setDeleting(i) },
              ]
            : undefined
        }
      />

      <InventoryForm open={formOpen} item={editing} onClose={() => setFormOpen(false)} />

      <Modal
        open={!!adjusting}
        onClose={() => setAdjusting(null)}
        title={`Adjust ${adjusting?.name}`}
        description={`Current stock: ${formatNumber(adjusting?.quantity_on_hand)} ${adjusting?.unit}. Positive restocks, negative consumes.`}
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={() => setAdjusting(null)} disabled={adjustMut.isPending}>
              Cancel
            </Button>
            <Button
              onClick={() => void adjustMut.mutate()}
              loading={adjustMut.isPending}
              disabled={!delta || !reason.trim()}
            >
              Apply adjustment
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Field label="Delta" required hint="e.g. +5 restock, -2 consumed.">
            <Input type="number" step="0.01" value={delta} onChange={(e) => setDelta(e.target.value)} placeholder="+5" className="font-mono" autoFocus />
          </Field>
          <Field label="Reason" required>
            <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="e.g. stock count, damaged batch" />
          </Field>
        </div>
      </Modal>

      <ConfirmDialog
        open={!!deleting}
        title="Delete item?"
        message={`This permanently removes ${deleting?.name} from inventory.`}
        confirmLabel="Delete"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => void deleteMut.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
