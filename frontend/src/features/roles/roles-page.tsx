import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil, Shield, Trash2 } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { listRoles, deleteRole } from '@/services/roles.service'
import type { RoleView } from '@/services/auth.service'
import { errorMessage } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { PageHeader } from '@/components/ui/page-header'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { ConfirmDialog } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'
import { RoleForm } from './role-form'

const columns: Column<RoleView>[] = [
  {
    key: 'name',
    header: 'Role',
    sortable: true,
    render: (r) => (
      <div className="flex items-center gap-3">
        <span className="flex size-8 items-center justify-center rounded-lg bg-brand-50 text-brand-600">
          <Shield className="size-4" />
        </span>
        <div className="min-w-0">
          <p className="flex items-center gap-2 font-semibold text-surface-800">
            {r.name}
            {r.is_system && <Badge tone="brand">system</Badge>}
          </p>
          <p className="font-mono text-[11px] text-surface-400">{r.code}</p>
        </div>
      </div>
    ),
  },
  {
    key: 'description',
    header: 'Description',
    render: (r) => (
      <span className="max-w-xs truncate text-xs text-surface-500">{r.description || '—'}</span>
    ),
  },
  {
    key: 'permissions',
    header: 'Permissions',
    render: (r) => (
      <Badge tone="slate">{r.permissions?.length ?? 0} granted</Badge>
    ),
  },
]

export function RolesPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<RoleView | null>(null)
  const [deleting, setDeleting] = useState<RoleView | null>(null)

  const list = useListQuery<RoleView>({
    queryKey: ['roles'],
    fetcher: listRoles,
    defaultSortBy: 'created_at',
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteRole(deleting!.id),
    onSuccess: () => {
      toast.toast({ title: 'Role deleted', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['roles'] })
      setDeleting(null)
    },
    onError: (err) => toast.toast({ title: 'Delete failed', description: errorMessage(err), variant: 'error' }),
  })

  const canWrite = can('POST /roles')

  return (
    <div className="space-y-5">
      <PageHeader
        title="Roles & permissions"
        description="Control what each role can do."
        actions={
          canWrite && (
            <Button
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              New role
            </Button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(r) => r.id}
        searchPlaceholder="Search roles…"
        emptyTitle="No roles found"
        emptyMessage="Create a role to start granting access."
        rowActions={
          canWrite
            ? (r) => [
                {
                  key: 'edit',
                  label: 'Edit role & permissions',
                  icon: <Pencil className="size-3.5" />,
                  onClick: () => {
                    setEditing(r)
                    setFormOpen(true)
                  },
                },
                {
                  key: 'delete',
                  label: 'Delete',
                  icon: <Trash2 className="size-3.5" />,
                  danger: true,
                  disabled: r.is_system,
                  onClick: () => setDeleting(r),
                },
              ]
            : undefined
        }
      />

      <RoleForm open={formOpen} role={editing} onClose={() => setFormOpen(false)} />

      <ConfirmDialog
        open={!!deleting}
        title="Delete role?"
        message={`Deleting ${deleting?.name} removes it from all assigned users. System roles cannot be deleted.`}
        confirmLabel="Delete role"
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => void deleteMutation.mutate()}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
