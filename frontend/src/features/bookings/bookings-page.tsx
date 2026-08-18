import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Ban, CheckCircle2, Pencil, Plus, Trash2 } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { listBookings, confirmBooking, cancelBooking, deleteBooking, type BookingView, type BookingSummary } from '@/services/bookings.service'
import { errorMessage } from '@/lib/api'
import { formatCurrency, formatDateTime, formatNumber, shortId } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { StatusBadge } from '@/components/ui/badge'
import { PageHeader } from '@/components/ui/page-header'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { Modal, ConfirmDialog } from '@/components/ui/modal'
import { Field, Textarea } from '@/components/ui/form'
import { useToast } from '@/components/ui/toast'
import { BookingForm } from './booking-form'

const STATUS_OPTIONS = [
  { value: 'pending', label: 'Pending' },
  { value: 'confirmed', label: 'Confirmed' },
  { value: 'cancelled', label: 'Cancelled' },
]

export function BookingsPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<BookingView | null>(null)
  const [confirming, setConfirming] = useState<BookingView | null>(null)
  const [cancelling, setCancelling] = useState<BookingView | null>(null)
  const [cancelReason, setCancelReason] = useState('')
  const [deleting, setDeleting] = useState<BookingView | null>(null)

  const list = useListQuery<BookingView, BookingSummary>({
    queryKey: ['bookings'],
    fetcher: listBookings,
  })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['bookings'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }

  const confirmMut = useMutation({
    mutationFn: () => confirmBooking(confirming!.id),
    onSuccess: () => {
      toast.toast({ title: 'Booking confirmed', variant: 'success' })
      invalidate()
      setConfirming(null)
    },
    onError: (err) => toast.toast({ title: 'Action failed', description: errorMessage(err), variant: 'error' }),
  })

  const cancelMut = useMutation({
    mutationFn: () => cancelBooking(cancelling!.id, cancelReason),
    onSuccess: () => {
      toast.toast({ title: 'Booking cancelled', variant: 'success' })
      invalidate()
      setCancelling(null)
      setCancelReason('')
    },
    onError: (err) => toast.toast({ title: 'Action failed', description: errorMessage(err), variant: 'error' }),
  })

  const deleteMut = useMutation({
    mutationFn: () => deleteBooking(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'Booking deleted', variant: 'success' })
      invalidate()
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  const currency = useShopCurrency()
  const canWrite = can('POST /bookings')

  const columns: Column<BookingView>[] = [
    {
      key: 'booking_no',
      header: 'Booking',
      sortable: true,
      render: (b) => (
        <div>
          <p className="font-mono text-xs font-semibold text-brand-700">{b.booking_no}</p>
          <p className="text-[11px] text-surface-400">#{shortId(b.id)}</p>
        </div>
      ),
    },
    {
      key: 'customer_id',
      header: 'Customer',
      render: (b) => <span className="text-xs text-surface-500">{shortId(b.customer_id)}</span>,
    },
    {
      key: 'pickup_address',
      header: 'Pickup address',
      render: (b) => (
        <span className="block max-w-56 truncate text-xs text-surface-600">{b.pickup_address}</span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      filterOptions: STATUS_OPTIONS,
      render: (b) => <StatusBadge status={b.status} />,
    },
    {
      key: 'estimated_amount',
      header: 'Est. value',
      sortable: true,
      align: 'right',
      render: (b) => (
        <span className="font-semibold tabular-nums text-surface-800">
          {formatCurrency(b.estimated_amount, b.currency)}
        </span>
      ),
    },
    {
      key: 'pickup_from',
      header: 'Pickup window',
      hideOnMobile: true,
      render: (b) => (
        <span className="text-xs text-surface-500">
          {formatDateTime(b.pickup_from ?? b.created_at)}
        </span>
      ),
    },
    {
      key: 'created_at',
      header: 'Created',
      sortable: true,
      hideOnMobile: true,
      render: (b) => <span className="text-xs text-surface-500">{formatDateTime(b.created_at)}</span>,
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Bookings"
        description="Pickup requests and their lifecycle."
        actions={
          canWrite && (
            <Button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              <Plus className="size-4" /> New booking
            </Button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(b) => b.id}
        searchPlaceholder="Search bookings by number or address…"
        emptyTitle="No bookings found"
        emptyMessage="Create a booking to start the laundry flow."
        summary={
          summary && (
            <SummaryBar
              chips={[
                { label: 'Pending', value: formatNumber(summary.pending), tone: 'amber' },
                { label: 'Confirmed', value: formatNumber(summary.confirmed), tone: 'blue' },
                { label: 'Cancelled', value: formatNumber(summary.cancelled), tone: 'red' },
                { label: 'Estimated value', value: formatCurrency(summary.estimated_value, currency), tone: 'brand' },
                { label: 'This month', value: formatNumber(summary.this_month) },
              ]}
            />
          )
        }
        rowActions={
          canWrite
            ? (b) => [
                ...(b.status === 'pending'
                  ? [{ key: 'confirm', label: 'Confirm', icon: <CheckCircle2 className="size-3.5" />, onClick: () => setConfirming(b) }]
                  : []),
                ...(b.status !== 'cancelled'
                  ? [{ key: 'cancel', label: 'Cancel booking', icon: <Ban className="size-3.5" />, danger: true, onClick: () => { setCancelling(b); setCancelReason('') } }]
                  : []),
                { key: 'edit', label: 'Edit', icon: <Pencil className="size-3.5" />, onClick: () => { setEditing(b); setFormOpen(true) } },
                { key: 'delete', label: 'Delete', icon: <Trash2 className="size-3.5" />, danger: true, onClick: () => setDeleting(b) },
              ]
            : undefined
        }
      />

      <BookingForm open={formOpen} booking={editing} onClose={() => setFormOpen(false)} />

      <ConfirmDialog
        open={!!confirming}
        title="Confirm booking?"
        message={`Confirm ${confirming?.booking_no}? This advances it into the operational flow.`}
        confirmLabel="Confirm booking"
        loading={confirmMut.isPending}
        onConfirm={() => void confirmMut.mutate()}
        onClose={() => setConfirming(null)}
      />

      <Modal
        open={!!cancelling}
        onClose={() => setCancelling(null)}
        title={`Cancel ${cancelling?.booking_no}`}
        description="Cancelled bookings stay in the record with their reason."
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={() => setCancelling(null)} disabled={cancelMut.isPending}>
              Keep booking
            </Button>
            <Button
              variant="danger"
              onClick={() => void cancelMut.mutate()}
              loading={cancelMut.isPending}
              disabled={!cancelReason.trim()}
            >
              Cancel booking
            </Button>
          </>
        }
      >
        <Field label="Reason" required hint="Shown in the audit trail.">
          <Textarea value={cancelReason} onChange={(e) => setCancelReason(e.target.value)} rows={3} placeholder="e.g. customer changed their mind" />
        </Field>
      </Modal>

      <ConfirmDialog
        open={!!deleting}
        title="Delete booking?"
        message={`This permanently removes ${deleting?.booking_no}.`}
        confirmLabel="Delete"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => void deleteMut.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
