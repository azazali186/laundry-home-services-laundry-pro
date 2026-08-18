import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowRight, Pencil, Plus, Trash2 } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { listOrders, updateOrderStatus, deleteOrder, ORDER_STATUSES, type OrderSummary, type OrderView } from '@/services/orders.service'
import { errorMessage } from '@/lib/api'
import { formatCurrency, formatDateTime, formatNumber } from '@/lib/format'
import { humanize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { StatusBadge } from '@/components/ui/badge'
import { PageHeader } from '@/components/ui/page-header'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { Modal, ConfirmDialog } from '@/components/ui/modal'
import { Field, Select } from '@/components/ui/form'
import { useToast } from '@/components/ui/toast'
import { OrderForm } from './order-form'

const STATUS_OPTIONS = ORDER_STATUSES.map((s) => ({ value: s, label: humanize(s) }))

const NEXT_STATUS: Record<string, string> = {
  pending: 'processing',
  processing: 'cleaned',
  cleaned: 'ironed',
  ironed: 'ready',
  ready: 'delivered',
}

export function OrdersPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<OrderView | null>(null)
  const [transitioning, setTransitioning] = useState<OrderView | null>(null)
  const [nextStatus, setNextStatus] = useState('')
  const [deleting, setDeleting] = useState<OrderView | null>(null)

  const list = useListQuery<OrderView, OrderSummary>({ queryKey: ['orders'], fetcher: listOrders })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['orders'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }

  const transitionMut = useMutation({
    mutationFn: () => updateOrderStatus(transitioning!.id, nextStatus),
    onSuccess: () => {
      toast.toast({ title: `Order moved to ${humanize(nextStatus)}`, variant: 'success' })
      invalidate()
      setTransitioning(null)
    },
    onError: (err) => toast.toast({ title: 'Transition failed', description: errorMessage(err), variant: 'error' }),
  })

  const deleteMut = useMutation({
    mutationFn: () => deleteOrder(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'Order deleted', variant: 'success' })
      invalidate()
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  const currency = useShopCurrency()
  const canWrite = can('POST /orders')

  const columns: Column<OrderView>[] = [
    {
      key: 'order_no',
      header: 'Order',
      sortable: true,
      render: (o) => (
        <div>
          <p className="font-mono text-xs font-semibold text-brand-700">{o.order_no}</p>
          <p className="text-[11px] text-surface-400">{formatDateTime(o.created_at)}</p>
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      filterOptions: STATUS_OPTIONS,
      render: (o) => <StatusBadge status={o.status} />,
    },
    {
      key: 'priority',
      header: 'Priority',
      sortable: true,
      filterOptions: [
        { value: 'normal', label: 'Normal' },
        { value: 'express', label: 'Express' },
        { value: 'priority', label: 'Priority' },
      ],
      hideOnMobile: true,
      render: (o) => (
        <span className={`text-xs font-semibold ${o.priority === 'priority' ? 'text-red-600' : o.priority === 'express' ? 'text-amber-600' : 'text-surface-500'}`}>
          {humanize(o.priority)}
        </span>
      ),
    },
    {
      key: 'item_count',
      header: 'Items',
      align: 'right',
      render: (o) => <span className="tabular-nums">{o.item_count}</span>,
    },
    {
      key: 'grand_total',
      header: 'Total',
      sortable: true,
      align: 'right',
      render: (o) => (
        <span className="font-semibold tabular-nums text-surface-800">
          {formatCurrency(o.grand_total, o.currency)}
        </span>
      ),
    },
    {
      key: 'created_at',
      header: 'Created',
      sortable: true,
      hideOnMobile: true,
      render: (o) => <span className="text-xs text-surface-500">{formatDateTime(o.created_at)}</span>,
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Orders"
        description="The laundry processing pipeline."
        actions={
          canWrite && (
            <Button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              <Plus className="size-4" /> New order
            </Button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(o) => o.id}
        searchPlaceholder="Search orders by number…"
        emptyTitle="No orders found"
        emptyMessage="Create an order from a confirmed booking."
        summary={
          summary && (
            <SummaryBar
              chips={[
                { label: 'Revenue', value: formatCurrency(summary.revenue, currency), tone: 'emerald' },
                { label: 'Collected', value: formatCurrency(summary.collected, currency), tone: 'blue' },
                { label: 'Pending value', value: formatCurrency(summary.pending_value, currency), tone: 'amber' },
                { label: 'Avg order', value: formatCurrency(summary.avg_order_value, currency) },
                { label: 'Items', value: formatNumber(summary.items_processed) },
              ]}
            />
          )
        }
        rowActions={
          canWrite
            ? (o) => {
                const actions = []
                const nxt = NEXT_STATUS[o.status]
                if (nxt && o.status !== 'cancelled') {
                  actions.push({
                    key: 'advance',
                    label: `Mark ${humanize(nxt)}`,
                    icon: <ArrowRight className="size-3.5" />,
                    onClick: () => {
                      setTransitioning(o)
                      setNextStatus(nxt)
                    },
                  })
                }
                actions.push({
                  key: 'edit',
                  label: 'Edit',
                  icon: <Pencil className="size-3.5" />,
                  onClick: () => {
                    setEditing(o)
                    setFormOpen(true)
                  },
                })
                actions.push({
                  key: 'delete',
                  label: 'Delete',
                  icon: <Trash2 className="size-3.5" />,
                  danger: true,
                  onClick: () => setDeleting(o),
                })
                return actions
              }
            : undefined
        }
      />

      <OrderForm open={formOpen} order={editing} onClose={() => setFormOpen(false)} />

      <Modal
        open={!!transitioning}
        onClose={() => setTransitioning(null)}
        title={`Move ${transitioning?.order_no}`}
        description="Advance the order to the next stage."
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={() => setTransitioning(null)} disabled={transitionMut.isPending}>
              Cancel
            </Button>
            <Button onClick={() => void transitionMut.mutate()} loading={transitionMut.isPending}>
              Move to {humanize(nextStatus)}
            </Button>
          </>
        }
      >
        <Field label="New status">
          <Select
            value={nextStatus}
            onChange={(e) => setNextStatus(e.target.value)}
            options={ORDER_STATUSES.map((s) => ({ value: s, label: humanize(s) }))}
          />
        </Field>
      </Modal>

      <ConfirmDialog
        open={!!deleting}
        title="Delete order?"
        message={`This permanently removes ${deleting?.order_no}.`}
        confirmLabel="Delete"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => void deleteMut.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
