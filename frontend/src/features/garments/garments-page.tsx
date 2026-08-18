import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowRight, Pencil, Plus, Trash2 } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { listGarments, updateGarment, deleteGarment, GARMENT_STATUSES, type GarmentSummary, type GarmentView } from '@/services/garments.service'
import { errorMessage } from '@/lib/api'
import { formatNumber } from '@/lib/format'
import { humanize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { StatusBadge } from '@/components/ui/badge'
import { PageHeader } from '@/components/ui/page-header'
import { ConfirmDialog } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'
import { GarmentForm } from './garment-form'

const NEXT: Record<string, string> = {
  received: 'cleaned',
  cleaned: 'ironed',
  ironed: 'ready',
}

export function GarmentsPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<GarmentView | null>(null)
  const [advancing, setAdvancing] = useState<GarmentView | null>(null)
  const [deleting, setDeleting] = useState<GarmentView | null>(null)

  const list = useListQuery<GarmentView, GarmentSummary>({ queryKey: ['garments'], fetcher: listGarments })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['garments'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }

  const advanceMut = useMutation({
    mutationFn: () => updateGarment({ id: advancing!.id, status: NEXT[advancing!.status] }),
    onSuccess: () => {
      toast.toast({ title: 'Garment advanced', variant: 'success' })
      invalidate()
      setAdvancing(null)
    },
    onError: (err) => toast.toast({ title: 'Update failed', description: errorMessage(err), variant: 'error' }),
  })

  const deleteMut = useMutation({
    mutationFn: () => deleteGarment(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'Garment removed', variant: 'success' })
      invalidate()
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  const canWrite = can('POST /garments')

  const columns: Column<GarmentView>[] = [
    {
      key: 'name',
      header: 'Garment',
      sortable: true,
      render: (g) => (
        <div>
          <p className="font-semibold text-surface-800">{g.name}</p>
          <p className="text-[11px] text-surface-400">{humanize(g.type)} · {g.color || '—'} · {g.fabric || '—'}</p>
        </div>
      ),
    },
    {
      key: 'order_id',
      header: 'Order',
      render: (g) => <p className="font-mono text-xs text-surface-500">{g.order_id.slice(0, 8)}</p>,
    },
    {
      key: 'quantity',
      header: 'Qty',
      sortable: true,
      align: 'right',
      render: (g) => <span className="tabular-nums">{g.quantity}</span>,
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      filterOptions: GARMENT_STATUSES.map((s) => ({ value: s, label: humanize(s) })),
      render: (g) => <StatusBadge status={g.status} />,
    },
    {
      key: 'instructions',
      header: 'Instructions',
      hideOnMobile: true,
      render: (g) => <span className="block max-w-52 truncate text-xs text-surface-500">{g.instructions || '—'}</span>,
    },
    {
      key: 'damage_notes',
      header: 'Damage',
      hideOnMobile: true,
      render: (g) => (
        <span className="text-xs text-surface-500">{g.damage_notes ? humanize(g.damage_notes) : '—'}</span>
      ),
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Garments"
        description="Items in the processing queue."
        actions={
          canWrite && (
            <Button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              <Plus className="size-4" /> New garment
            </Button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(g) => g.id}
        searchPlaceholder="Search garments…"
        emptyTitle="No garments tracked"
        emptyMessage="Add garments to orders as they're received."
        summary={
          summary && (
            <SummaryBar
              chips={[
                { label: 'Received', value: formatNumber(summary.received), tone: 'amber' },
                { label: 'Cleaned', value: formatNumber(summary.cleaned), tone: 'cyan' },
                { label: 'Ironed', value: formatNumber(summary.ironed), tone: 'cyan' },
                { label: 'Ready', value: formatNumber(summary.ready), tone: 'emerald' },
                { label: 'Damaged', value: formatNumber(summary.damaged), tone: 'red' },
              ]}
            />
          )
        }
        rowActions={
          canWrite
            ? (g) => [
                ...(NEXT[g.status]
                  ? [{ key: 'advance', label: `Move to ${humanize(NEXT[g.status])}`, icon: <ArrowRight className="size-3.5" />, onClick: () => setAdvancing(g) }]
                  : []),
                {
                  key: 'edit',
                  label: 'Edit',
                  icon: <Pencil className="size-3.5" />,
                  onClick: () => {
                    setEditing(g)
                    setFormOpen(true)
                  },
                },
                { key: 'delete', label: 'Remove', icon: <Trash2 className="size-3.5" />, danger: true, onClick: () => setDeleting(g) },
              ]
            : undefined
        }
      />

      <GarmentForm open={formOpen} garment={editing} onClose={() => setFormOpen(false)} />

      <ConfirmDialog
        open={!!advancing}
        title="Advance garment?"
        message={`Move ${advancing?.name} to ${humanize(NEXT[advancing?.status ?? ''])}?`}
        confirmLabel="Advance"
        loading={advanceMut.isPending}
        onConfirm={() => void advanceMut.mutate()}
        onClose={() => setAdvancing(null)}
      />
      <ConfirmDialog
        open={!!deleting}
        title="Remove garment?"
        message="This permanently removes the garment record."
        confirmLabel="Remove"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => void deleteMut.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
