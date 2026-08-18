import { Fingerprint } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { listLoginLogs, type LoginLogView } from '@/services/logs.service'
import { formatDateTime } from '@/lib/format'
import { humanize } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { PageHeader } from '@/components/ui/page-header'

const columns: Column<LoginLogView>[] = [
  {
    key: 'email',
    header: 'Email',
    sortable: true,
    render: (l) => <span className="text-sm font-medium text-surface-700">{l.email}</span>,
  },
  {
    key: 'status',
    header: 'Result',
    sortable: true,
    filterOptions: [
      { value: 'success', label: 'Success' },
      { value: 'failed', label: 'Failed' },
    ],
    render: (l) => <Badge tone={l.status === 'success' ? 'emerald' : 'red'}>{humanize(l.status)}</Badge>,
  },
  {
    key: 'reason',
    header: 'Reason',
    render: (l) => <span className="text-xs text-surface-500">{l.reason || '—'}</span>,
  },
  {
    key: 'ip',
    header: 'IP address',
    sortable: true,
    hideOnMobile: true,
    render: (l) => <span className="font-mono text-xs text-surface-500">{l.ip || '—'}</span>,
  },
  {
    key: 'user_agent',
    header: 'User agent',
    hideOnMobile: true,
    render: (l) => <span className="block max-w-64 truncate text-xs text-surface-400">{l.user_agent || '—'}</span>,
  },
  {
    key: 'created_at',
    header: 'Time',
    sortable: true,
    render: (l) => <span className="text-xs text-surface-500">{formatDateTime(l.created_at)}</span>,
  },
]

export function LoginLogsPage() {
  const list = useListQuery<LoginLogView>({ queryKey: ['login-logs'], fetcher: listLoginLogs })

  return (
    <div className="space-y-5">
      <PageHeader
        title="Login logs"
        description="Every authentication attempt against the API."
      />
      <DataTable
        columns={columns}
        list={list}
        rowKey={(l) => l.id}
        searchPlaceholder="Search by email or IP…"
        emptyTitle="No login attempts"
        emptyMessage="Login activity will appear here."
        toolbarExtra={
          <span className="inline-flex items-center gap-1.5 rounded-lg bg-surface-100 px-2.5 py-1.5 text-xs font-medium text-surface-500">
            <Fingerprint className="size-3.5" /> Sensitive fields redacted
          </span>
        }
      />
    </div>
  )
}
