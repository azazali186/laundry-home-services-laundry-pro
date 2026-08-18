import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, Pencil, Plus, Trash2 } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { listDeliveries, completeDelivery, deleteDelivery, type DeliverySummary, type DeliveryView } from '@/services/deliveries.service'
import { errorMessage } from '@/lib/api'
import { formatDateTime, formatNumber } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { StatusBadge } from '@/components/ui/badge'
import { PageHeader } from '@/components/ui/page-header'
import { ConfirmDialog } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'
import { DeliveryForm } from './delivery-form'

export function DeliveriesPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<DeliveryView | null>(null)
  const [completing, setCompleting] = useState<DeliveryView | null>(null)
  const [deleting, setDeleting] = useState<DeliveryView | null>(null)

  const list = useListQuery<DeliveryView, DeliverySummary>({ queryKey: ['deliveries'], fetcher: listDeliveries })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['deliveries'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }

  const completeMut = useMutation({
    mutationFn: () => completeDelivery(completing!.id),
    onSuccess: () => {
      toast.toast({ title: 'Delivery completed', variant: 'success' })
      invalidate()
      setCompleting(null)
    },
    onError: (err) => toast.toast({ title: 'Action failed', description: errorMessage(err), variant: 'error' }),
  })

  const deleteMut = useMutation({
    mutationFn: () => deleteDelivery(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'Delivery deleted', variant: 'success' })
      invalidate()
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  const canWrite = can('POST /deliveries')

  const columns: Column<DeliveryView>[] = [
    {
      key: 'id',
      header: 'Delivery',
      render: (d) => <p className="font-mono text-xs font-semibold text-brand-700">#{d.id.slice(0, 8)}</p>,
    },
    {
      key: 'order_id',
      header: 'Order',
      render: (d) => <p className="font-mono text-xs text-surface-500">{d.order_id.slice(0, 8)}</p>,
    },
    {
      key: 'address',
      header: 'Address',
      render: (d) => <span className="block max-w-56 truncate text-xs text-surface-600">{d.address}</span>,
    },
    {
      key: 'scheduled_at',
      header: 'Scheduled',
      sortable: true,
      render: (d) => <span className="text-xs text-surface-500">{formatDateTime(d.scheduled_at)}</span>,
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      filterOptions: [
        { value: 'scheduled', label: 'Scheduled' },
        { value: 'in_transit', label: 'In transit' },
        { value: 'delivered', label: 'Delivered' },
        { value: 'cancelled', label: 'Cancelled' },
      ],
      render: (d) => <StatusBadge status={d.status} />,
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Deliveries"
        description="Completed orders heading home."
        actions={
          canWrite && (
            <Button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              <Plus className="size-4" /> New delivery
            </Button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(d) => d.id}
        searchPlaceholder="Search deliveries by address…"
        emptyTitle="No deliveries scheduled"
        emptyMessage="Schedule a delivery for a ready order."
        summary={
          summary && (
            <SummaryBar
              chips={[
                { label: 'Scheduled', value: formatNumber(summary.scheduled), tone: 'amber' },
                { label: 'In transit', value: formatNumber(summary.in_transit), tone: 'blue' },
                { label: 'Delivered', value: formatNumber(summary.delivered), tone: 'emerald' },
                { label: 'Cancelled', value: formatNumber(summary.cancelled), tone: 'red' },
              ]}
            />
          )
        }
        rowActions={
          canWrite
            ? (d) => [
                ...(d.status === 'scheduled' || d.status === 'in_transit'
                  ? [{ key: 'complete', label: 'Mark delivered', icon: <CheckCircle2 className="size-3.5" />, onClick: () => setCompleting(d) }]
                  : []),
                {
                  key: 'edit',
                  label: 'Edit',
                  icon: <Pencil className="size-3.5" />,
                  onClick: () => {
                    setEditing(d)
                    setFormOpen(true)
                  },
                },
                { key: 'delete', label: 'Delete', icon: <Trash2 className="size-3.5" />, danger: true, onClick: () => setDeleting(d) },
              ]
            : undefined
        }
      />

      <DeliveryForm open={formOpen} delivery={editing} onClose={() => setFormOpen(false)} />

      <ConfirmDialog
        open={!!completing}
        title="Mark as delivered?"
        message="This completes the delivery and settles the order."
        confirmLabel="Mark delivered"
        loading={completeMut.isPending}
        onConfirm={() => void completeMut.mutate()}
        onClose={() => setCompleting(null)}
      />
      <ConfirmDialog
        open={!!deleting}
        title="Delete delivery?"
        message="This permanently removes the delivery record."
        confirmLabel="Delete"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => void deleteMut.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
