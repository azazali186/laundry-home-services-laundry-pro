import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Banknote, CheckCircle2, Pencil, RotateCcw, Trash2 } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { listPayments, capturePayment, refundPayment, deletePayment, PAYMENT_METHODS, type PaymentSummary, type PaymentView } from '@/services/payments.service'
import { errorMessage } from '@/lib/api'
import { formatCurrency, formatDateTime, formatNumber } from '@/lib/format'
import { humanize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { PageHeader } from '@/components/ui/page-header'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { Modal, ConfirmDialog } from '@/components/ui/modal'
import { Field, Input } from '@/components/ui/form'
import { StatusBadge } from '@/components/ui/badge'
import { useToast } from '@/components/ui/toast'
import { PaymentForm } from './payment-form'

export function PaymentsPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<PaymentView | null>(null)
  const [capturing, setCapturing] = useState<PaymentView | null>(null)
  const [refunding, setRefunding] = useState<PaymentView | null>(null)
  const [refundAmount, setRefundAmount] = useState('')
  const [deleting, setDeleting] = useState<PaymentView | null>(null)

  const list = useListQuery<PaymentView, PaymentSummary>({ queryKey: ['payments'], fetcher: listPayments })

  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['payments'] })

  const captureMut = useMutation({
    mutationFn: () => capturePayment(capturing!.id),
    onSuccess: () => {
      toast.toast({ title: 'Payment captured', variant: 'success' })
      invalidate()
      setCapturing(null)
    },
    onError: (err) => toast.toast({ title: 'Capture failed', description: errorMessage(err), variant: 'error' }),
  })

  const refundMut = useMutation({
    mutationFn: () => refundPayment(refunding!.id, refundAmount ? Number(refundAmount) : undefined),
    onSuccess: () => {
      toast.toast({ title: 'Refund recorded', variant: 'success' })
      invalidate()
      setRefunding(null)
    },
    onError: (err) => toast.toast({ title: 'Refund failed', description: errorMessage(err), variant: 'error' }),
  })

  const deleteMut = useMutation({
    mutationFn: () => deletePayment(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'Payment deleted', variant: 'success' })
      invalidate()
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  const currency = useShopCurrency()
  const canWrite = can('POST /payments')

  const columns: Column<PaymentView>[] = [
    {
      key: 'payment_no',
      header: 'Payment',
      sortable: true,
      render: (p) => (
        <div>
          <p className="font-mono text-xs font-semibold text-brand-700">{p.payment_no}</p>
          <p className="text-[11px] text-surface-400">{formatDateTime(p.created_at)}</p>
        </div>
      ),
    },
    {
      key: 'method',
      header: 'Method',
      sortable: true,
      filterOptions: PAYMENT_METHODS.map((m) => ({ value: m, label: humanize(m) })),
      render: (p) => (
        <span className="inline-flex items-center gap-1.5 text-xs text-surface-600">
          <Banknote className="size-3.5 text-surface-400" />
          {humanize(p.method)}
        </span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      filterOptions: ['pending', 'paid', 'failed', 'refunded'].map((s) => ({ value: s, label: humanize(s) })),
      render: (p) => <StatusBadge status={p.status} />,
    },
    {
      key: 'amount',
      header: 'Amount',
      sortable: true,
      align: 'right',
      render: (p) => (
        <span className="font-semibold tabular-nums text-surface-800">{formatCurrency(p.amount, p.currency)}</span>
      ),
    },
    {
      key: 'gateway',
      header: 'Gateway',
      hideOnMobile: true,
      render: (p) => <span className="text-xs text-surface-500">{p.gateway || '—'}</span>,
    },
    {
      key: 'card_last4',
      header: 'Card',
      hideOnMobile: true,
      render: (p) => <span className="font-mono text-xs text-surface-400">{p.card_last4 || '—'}</span>,
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Payments"
        description="Money in, money out."
        actions={
          canWrite && (
            <button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
              className="rounded-lg bg-brand-600 px-4 text-sm font-medium text-white shadow-sm transition-all hover:bg-brand-700 active:scale-[0.985] h-9.5"
            >
              Record payment
            </button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(p) => p.id}
        searchPlaceholder="Search payments…"
        emptyTitle="No payments found"
        emptyMessage="Record a payment against an order."
        summary={
          summary && (
            <SummaryBar
              chips={[
                { label: 'Collected', value: formatCurrency(summary.collected, currency), tone: 'emerald' },
                { label: 'Outstanding', value: formatCurrency(summary.outstanding, currency), tone: 'amber' },
                { label: 'Refunded', value: formatCurrency(summary.refunded_sum, currency), tone: 'violet' },
                { label: 'Paid', value: formatNumber(summary.paid), tone: 'blue' },
                { label: 'Pending', value: formatNumber(summary.pending), tone: 'amber' },
                { label: 'Failed', value: formatNumber(summary.failed), tone: 'red' },
              ]}
            />
          )
        }
        rowActions={
          canWrite
            ? (p) => [
                ...(p.status === 'pending'
                  ? [{ key: 'capture', label: 'Capture', icon: <CheckCircle2 className="size-3.5" />, onClick: () => setCapturing(p) }]
                  : []),
                ...(p.status === 'paid'
                  ? [{ key: 'refund', label: 'Refund', icon: <RotateCcw className="size-3.5" />, onClick: () => { setRefunding(p); setRefundAmount('') } }]
                  : []),
                { key: 'edit', label: 'Edit', icon: <Pencil className="size-3.5" />, onClick: () => { setEditing(p); setFormOpen(true) } },
                { key: 'delete', label: 'Delete', icon: <Trash2 className="size-3.5" />, danger: true, onClick: () => setDeleting(p) },
              ]
            : undefined
        }
      />

      <PaymentForm open={formOpen} payment={editing} onClose={() => setFormOpen(false)} />

      <ConfirmDialog
        open={!!capturing}
        title="Capture payment?"
        message={`Confirm capture of ${capturing?.payment_no}?`}
        confirmLabel="Capture"
        loading={captureMut.isPending}
        onConfirm={() => void captureMut.mutate()}
        onClose={() => setCapturing(null)}
      />

      <Modal
        open={!!refunding}
        onClose={() => setRefunding(null)}
        title={`Refund ${refunding?.payment_no}`}
        description="Leave amount empty for a full refund."
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={() => setRefunding(null)} disabled={refundMut.isPending}>
              Cancel
            </Button>
            <Button variant="danger" onClick={() => void refundMut.mutate()} loading={refundMut.isPending}>
              Process refund
            </Button>
          </>
        }
      >
        <Field label="Amount" hint={`Paid: ${formatCurrency(refunding?.amount, refunding?.currency)}`}>
          <Input type="number" step="0.01" min="0" value={refundAmount} onChange={(e) => setRefundAmount(e.target.value)} placeholder="Full amount by default" />
        </Field>
      </Modal>

      <ConfirmDialog
        open={!!deleting}
        title="Delete payment?"
        message={`This permanently removes ${deleting?.payment_no}.`}
        confirmLabel="Delete"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => void deleteMut.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
