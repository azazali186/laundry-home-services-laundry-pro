import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil, Trash2, UserPlus } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { listUsers, deleteUser, type UserSummary } from '@/services/users.service'
import type { UserView } from '@/services/auth.service'
import { errorMessage } from '@/lib/api'
import { formatDate, formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge, StatusBadge } from '@/components/ui/badge'
import { PageHeader } from '@/components/ui/page-header'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { ConfirmDialog } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'
import { UserForm } from './user-form'

const columns: Column<UserView>[] = [
  {
    key: 'name',
    header: 'User',
    sortable: true,
    render: (u) => (
      <div className="flex items-center gap-3">
        <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-brand-500 to-brand-700 text-[11px] font-bold text-white">
          {u.name
            .split(/\s+/)
            .slice(0, 2)
            .map((p) => p[0]?.toUpperCase())
            .join('')}
        </span>
        <div className="min-w-0">
          <p className="truncate font-semibold text-surface-800">{u.name}</p>
          <p className="truncate text-xs text-surface-400">{u.email}</p>
        </div>
      </div>
    ),
  },
  {
    key: 'roles',
    header: 'Roles',
    render: (u) =>
      (u.roles?.length ?? 0) > 0 ? (
        <div className="flex flex-wrap gap-1">
          {u.roles!.map((r) => (
            <Badge key={r.id} tone={r.is_system ? 'brand' : 'slate'}>
              {r.name}
            </Badge>
          ))}
        </div>
      ) : (
        <span className="text-xs text-surface-400">None</span>
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
    render: (u) => (
      <StatusBadge status={u.is_active ? 'Active' : 'Inactive'} />
    ),
  },
  {
    key: 'last_login_at',
    header: 'Last login',
    sortable: true,
    hideOnMobile: true,
    render: (u) => (
      <span className="text-xs text-surface-500">{formatDate(u.last_login_at)}</span>
    ),
  },
  {
    key: 'created_at',
    header: 'Created',
    sortable: true,
    hideOnMobile: true,
    render: (u) => <span className="text-xs text-surface-500">{formatDate(u.created_at)}</span>,
  },
]

export function UsersPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<UserView | null>(null)
  const [deleting, setDeleting] = useState<UserView | null>(null)

  const list = useListQuery<UserView, UserSummary>({
    queryKey: ['users'],
    fetcher: listUsers,
    defaultSortBy: 'created_at',
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteUser(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'User deleted', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['users'] })
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  const canWrite = can('POST /users')

  return (
    <div className="space-y-5">
      <PageHeader
        title="Users"
        description="Team members and their access."
        actions={
          canWrite && (
            <Button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              <UserPlus className="size-4" /> New user
            </Button>
          )
        }
      />

      <div className="space-y-4">
        {summary && (
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <MiniStat label="Active users" value={formatNumber(summary.total_active)} tone="text-emerald-600" />
            <MiniStat label="Inactive" value={formatNumber(summary.total_inactive)} tone="text-surface-600" />
            <MiniStat label="Admins" value={formatNumber(summary.total_admins)} tone="text-brand-600" />
            <MiniStat label="New this month" value={formatNumber(summary.new_this_month)} tone="text-blue-600" />
          </div>
        )}

        <DataTable
          columns={columns}
          list={list}
          rowKey={(u) => u.id}
          searchPlaceholder="Search users by name or email…"
          emptyTitle="No users found"
          emptyMessage="Try adjusting your search or filters."
          summary={
            <SummaryBar
              chips={[
                { label: 'Showing', value: formatNumber(list.total), tone: 'brand' },
              ]}
            />
          }
          rowActions={
            canWrite
              ? (u) => [
                  {
                    key: 'edit',
                    label: 'Edit',
                    icon: <Pencil className="size-3.5" />,
                    onClick: () => {
                      setEditing(u)
                      setFormOpen(true)
                    },
                  },
                  {
                    key: 'delete',
                    label: 'Delete',
                    icon: <Trash2 className="size-3.5" />,
                    danger: true,
                    disabled: u.is_active && (u.roles ?? []).some((r) => r.is_system),
                    onClick: () => setDeleting(u),
                  },
                ]
              : undefined
          }
        />
      </div>

      <UserForm open={formOpen} user={editing} onClose={() => setFormOpen(false)} />

      <ConfirmDialog
        open={!!deleting}
        title="Delete user?"
        message={`This permanently removes ${deleting?.name}. Their history remains in the audit trail.`}
        confirmLabel="Delete user"
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => void deleteMutation.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}

function MiniStat({ label, value, tone }: { label: string; value: string; tone: string }) {
  return (
    <div className="rounded-2xl border border-surface-200/80 bg-white p-4 shadow-card">
      <p className="text-xs font-medium text-surface-500">{label}</p>
      <p className={cn('mt-1 text-xl font-bold tabular-nums', tone)}>{value}</p>
    </div>
  )
}
