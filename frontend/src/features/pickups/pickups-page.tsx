import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, Pencil, Plus, Trash2 } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { listPickups, completePickup, deletePickup, type PickupSummary, type PickupView } from '@/services/pickups.service'
import { errorMessage } from '@/lib/api'
import { formatDateTime, formatNumber } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { StatusBadge } from '@/components/ui/badge'
import { PageHeader } from '@/components/ui/page-header'
import { ConfirmDialog } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'
import { PickupForm } from './pickup-form'

export function PickupsPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<PickupView | null>(null)
  const [completing, setCompleting] = useState<PickupView | null>(null)
  const [deleting, setDeleting] = useState<PickupView | null>(null)

  const list = useListQuery<PickupView, PickupSummary>({ queryKey: ['pickups'], fetcher: listPickups })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['pickups'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }

  const completeMut = useMutation({
    mutationFn: () => completePickup(completing!.id),
    onSuccess: () => {
      toast.toast({ title: 'Pickup completed', variant: 'success' })
      invalidate()
      setCompleting(null)
    },
    onError: (err) => toast.toast({ title: 'Action failed', description: errorMessage(err), variant: 'error' }),
  })

  const deleteMut = useMutation({
    mutationFn: () => deletePickup(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'Pickup deleted', variant: 'success' })
      invalidate()
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  const canWrite = can('POST /pickups')

  const columns: Column<PickupView>[] = [
    {
      key: 'id',
      header: 'Pickup',
      render: (p) => <p className="font-mono text-xs font-semibold text-brand-700">#{p.id.slice(0, 8)}</p>,
    },
    {
      key: 'booking_id',
      header: 'Booking',
      render: (p) => <p className="font-mono text-xs text-surface-500">{p.booking_id.slice(0, 8)}</p>,
    },
    {
      key: 'address',
      header: 'Address',
      render: (p) => <span className="block max-w-56 truncate text-xs text-surface-600">{p.address}</span>,
    },
    {
      key: 'scheduled_at',
      header: 'Scheduled',
      sortable: true,
      render: (p) => <span className="text-xs text-surface-500">{formatDateTime(p.scheduled_at)}</span>,
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      filterOptions: [
        { value: 'scheduled', label: 'Scheduled' },
        { value: 'completed', label: 'Completed' },
        { value: 'cancelled', label: 'Cancelled' },
      ],
      render: (p) => <StatusBadge status={p.status} />,
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Pickups"
        description="Scheduled garment collection runs."
        actions={
          canWrite && (
            <Button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              <Plus className="size-4" /> New pickup
            </Button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(p) => p.id}
        searchPlaceholder="Search pickups by address…"
        emptyTitle="No pickups scheduled"
        emptyMessage="Schedule a pickup from a confirmed booking."
        summary={
          summary && (
            <SummaryBar
              chips={[
                { label: 'Scheduled', value: formatNumber(summary.scheduled), tone: 'amber' },
                { label: 'Completed', value: formatNumber(summary.completed), tone: 'emerald' },
                { label: 'Active', value: formatNumber(summary.active), tone: 'blue' },
              ]}
            />
          )
        }
        rowActions={
          canWrite
            ? (p) => [
                ...(p.status === 'scheduled'
                  ? [{ key: 'complete', label: 'Mark completed', icon: <CheckCircle2 className="size-3.5" />, onClick: () => setCompleting(p) }]
                  : []),
                {
                  key: 'edit',
                  label: 'Edit',
                  icon: <Pencil className="size-3.5" />,
                  onClick: () => {
                    setEditing(p)
                    setFormOpen(true)
                  },
                },
                { key: 'delete', label: 'Delete', icon: <Trash2 className="size-3.5" />, danger: true, onClick: () => setDeleting(p) },
              ]
            : undefined
        }
      />

      <PickupForm open={formOpen} pickup={editing} onClose={() => setFormOpen(false)} />

      <ConfirmDialog
        open={!!completing}
        title="Complete pickup?"
        message="This marks the pickup run as completed."
        confirmLabel="Complete"
        loading={completeMut.isPending}
        onConfirm={() => void completeMut.mutate()}
        onClose={() => setCompleting(null)}
      />
      <ConfirmDialog
        open={!!deleting}
        title="Delete pickup?"
        message="This permanently removes the pickup record."
        confirmLabel="Delete"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => void deleteMut.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
