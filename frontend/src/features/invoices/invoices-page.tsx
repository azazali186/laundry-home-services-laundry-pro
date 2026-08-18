import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, FileText, Pencil, Plus, Trash2 } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { listInvoices, markInvoicePaid, deleteInvoice, INVOICE_STATUSES, type InvoiceSummary, type InvoiceView } from '@/services/invoices.service'
import { errorMessage } from '@/lib/api'
import { formatCurrency, formatDate, formatNumber } from '@/lib/format'
import { humanize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { StatusBadge } from '@/components/ui/badge'
import { PageHeader } from '@/components/ui/page-header'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { ConfirmDialog } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'
import { InvoiceForm } from './invoice-form'

export function InvoicesPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<InvoiceView | null>(null)
  const [paying, setPaying] = useState<InvoiceView | null>(null)
  const [deleting, setDeleting] = useState<InvoiceView | null>(null)

  const list = useListQuery<InvoiceView, InvoiceSummary>({ queryKey: ['invoices'], fetcher: listInvoices })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['invoices'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }

  const markPaidMut = useMutation({
    mutationFn: () => markInvoicePaid(paying!.id),
    onSuccess: () => {
      toast.toast({ title: 'Invoice marked as paid', variant: 'success' })
      invalidate()
      setPaying(null)
    },
    onError: (err) => toast.toast({ title: 'Action failed', description: errorMessage(err), variant: 'error' }),
  })

  const deleteMut = useMutation({
    mutationFn: () => deleteInvoice(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'Invoice deleted', variant: 'success' })
      invalidate()
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  const currency = useShopCurrency()
  const canWrite = can('POST /invoices')

  const columns: Column<InvoiceView>[] = [
    {
      key: 'invoice_no',
      header: 'Invoice',
      sortable: true,
      render: (i) => (
        <div className="flex items-center gap-3">
          <span className="flex size-8 items-center justify-center rounded-lg bg-brand-50 text-brand-600">
            <FileText className="size-4" />
          </span>
          <div>
            <p className="font-mono text-xs font-semibold text-brand-700">{i.invoice_no}</p>
            <p className="text-[11px] text-surface-400">{i.billing_name || '—'}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      filterOptions: INVOICE_STATUSES.map((s) => ({ value: s, label: humanize(s) })),
      render: (i) => <StatusBadge status={i.status} />,
    },
    {
      key: 'issue_date',
      header: 'Issued',
      sortable: true,
      hideOnMobile: true,
      render: (i) => <span className="text-xs text-surface-500">{formatDate(i.issue_date)}</span>,
    },
    {
      key: 'due_date',
      header: 'Due',
      hideOnMobile: true,
      render: (i) => <span className="text-xs text-surface-500">{formatDate(i.due_date)}</span>,
    },
    {
      key: 'total',
      header: 'Total',
      sortable: true,
      align: 'right',
      render: (i) => (
        <div className="text-right">
          <p className="font-semibold tabular-nums text-surface-800">{formatCurrency(i.total, currency)}</p>
          {i.discount > 0 && <p className="text-[10px] text-surface-400">−{formatCurrency(i.discount, currency)} discount</p>}
        </div>
      ),
    },
    {
      key: 'created_at',
      header: 'Created',
      sortable: true,
      hideOnMobile: true,
      render: (i) => <span className="text-xs text-surface-500">{formatDate(i.created_at)}</span>,
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Invoices"
        description="Billing records for your orders."
        actions={
          canWrite && (
            <Button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              <Plus className="size-4" /> New invoice
            </Button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(i) => i.id}
        searchPlaceholder="Search invoices…"
        emptyTitle="No invoices found"
        emptyMessage="Create an invoice for an order."
        summary={
          summary && (
            <SummaryBar
              chips={[
                { label: 'Outstanding', value: formatCurrency(summary.outstanding, currency), tone: 'amber' },
                { label: 'Collected', value: formatCurrency(summary.collected, currency), tone: 'emerald' },
                { label: 'Draft', value: formatNumber(summary.draft) },
                { label: 'Issued', value: formatNumber(summary.issued), tone: 'blue' },
                { label: 'Paid', value: formatNumber(summary.paid), tone: 'emerald' },
                { label: 'Void', value: formatNumber(summary.void), tone: 'red' },
              ]}
            />
          )
        }
        rowActions={
          canWrite
            ? (i) => [
                ...(i.status !== 'paid' && i.status !== 'void'
                  ? [{ key: 'mark-paid', label: 'Mark as paid', icon: <CheckCircle2 className="size-3.5" />, onClick: () => setPaying(i) }]
                  : []),
                { key: 'edit', label: 'Edit', icon: <Pencil className="size-3.5" />, onClick: () => { setEditing(i); setFormOpen(true) } },
                { key: 'delete', label: 'Delete', icon: <Trash2 className="size-3.5" />, danger: true, onClick: () => setDeleting(i) },
              ]
            : undefined
        }
      />

      <InvoiceForm open={formOpen} invoice={editing} onClose={() => setFormOpen(false)} />

      <ConfirmDialog
        open={!!paying}
        title="Mark invoice as paid?"
        message={`Mark ${paying?.invoice_no} (${formatCurrency(paying?.total, currency)}) as paid?`}
        confirmLabel="Mark paid"
        loading={markPaidMut.isPending}
        onConfirm={() => void markPaidMut.mutate()}
        onClose={() => setPaying(null)}
      />

      <ConfirmDialog
        open={!!deleting}
        title="Delete invoice?"
        message={`This permanently removes ${deleting?.invoice_no}.`}
        confirmLabel="Delete"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => void deleteMut.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
