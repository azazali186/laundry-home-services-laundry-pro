import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Star, Trash2 } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { listRatings, deleteRating, type RatingSummary, type RatingView } from '@/services/ratings.service'
import { errorMessage } from '@/lib/api'
import { formatDateTime, formatNumber, formatScore } from '@/lib/format'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { Badge } from '@/components/ui/badge'
import { PageHeader } from '@/components/ui/page-header'
import { ConfirmDialog } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'
import { RatingForm } from './rating-form'

function ScoreStars({ score }: { score: number }) {
  return (
    <span className="inline-flex items-center gap-0.5" aria-label={`${score} out of 5`}>
      {[1, 2, 3, 4, 5].map((n) => (
        <Star
          key={n}
          className={cn('size-3.5', n <= score ? 'fill-amber-400 text-amber-400' : 'text-surface-200')}
          aria-hidden
        />
      ))}
    </span>
  )
}

export function RatingsPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<RatingView | null>(null)
  const [deleting, setDeleting] = useState<RatingView | null>(null)

  const list = useListQuery<RatingView, RatingSummary>({ queryKey: ['ratings'], fetcher: listRatings })

  const deleteMut = useMutation({
    mutationFn: () => deleteRating(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'Rating removed', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['ratings'] })
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  const canWrite = can('POST /ratings')

  const columns: Column<RatingView>[] = [
    {
      key: 'score',
      header: 'Rating',
      sortable: true,
      filterOptions: [
        { value: '5', label: '5 stars' },
        { value: '4', label: '4 stars' },
        { value: '3', label: '3 stars' },
        { value: '2', label: '2 stars' },
        { value: '1', label: '1 star' },
      ],
      render: (r) => <ScoreStars score={r.score} />,
    },
    {
      key: 'order_id',
      header: 'Order',
      render: (r) => <p className="font-mono text-xs text-surface-500">{r.order_id.slice(0, 8)}</p>,
    },
    {
      key: 'comment',
      header: 'Comment',
      render: (r) => <span className="block max-w-72 truncate text-xs text-surface-600">{r.comment || '—'}</span>,
    },
    {
      key: 'is_public',
      header: 'Visibility',
      sortable: true,
      filterOptions: [
        { value: 'true', label: 'Public' },
        { value: 'false', label: 'Private' },
      ],
      render: (r) => <Badge tone={r.is_public ? 'blue' : 'slate'}>{r.is_public ? 'Public' : 'Private'}</Badge>,
    },
    {
      key: 'created_at',
      header: 'Submitted',
      sortable: true,
      hideOnMobile: true,
      render: (r) => <span className="text-xs text-surface-500">{formatDateTime(r.created_at)}</span>,
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Ratings"
        description="Customer feedback on your service."
        actions={
          canWrite && (
            <Button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              <Plus className="size-4" /> New rating
            </Button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(r) => r.id}
        searchPlaceholder="Search comments…"
        emptyTitle="No ratings yet"
        emptyMessage="Customer ratings will appear here."
        summary={
          summary && (
            <SummaryBar
              chips={[
                { label: 'Average', value: `${formatScore(summary?.avg_score)} ★`, tone: 'amber' },
                { label: 'Total', value: formatNumber(summary?.total) },
                { label: '5★', value: formatNumber(summary?.five_star), tone: 'emerald' },
                { label: '1★', value: formatNumber(summary?.one_star), tone: 'red' },
              ]}
            />
          )
        }
        rowActions={
          canWrite
            ? (r) => [
                {
                  key: 'edit',
                  label: 'Edit',
                  icon: <Pencil className="size-3.5" />,
                  onClick: () => {
                    setEditing(r)
                    setFormOpen(true)
                  },
                },
                { key: 'delete', label: 'Remove', icon: <Trash2 className="size-3.5" />, danger: true, onClick: () => setDeleting(r) },
              ]
            : undefined
        }
      />

      <RatingForm open={formOpen} rating={editing} onClose={() => setFormOpen(false)} />

      <ConfirmDialog
        open={!!deleting}
        title="Remove rating?"
        message="This permanently removes the rating and its comment."
        confirmLabel="Remove"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => void deleteMut.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
