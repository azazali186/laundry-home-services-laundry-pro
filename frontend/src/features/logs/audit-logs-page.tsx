import { useState } from 'react'
import { GitCompareArrows } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { listAuditLogs, type AuditLogView } from '@/services/logs.service'
import { formatDateTime, shortId } from '@/lib/format'
import { humanize } from '@/lib/utils'
import { Badge, type BadgeTone } from '@/components/ui/badge'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { Modal } from '@/components/ui/modal'
import { PageHeader } from '@/components/ui/page-header'

const ACTION_TONE: Record<string, BadgeTone> = {
  create: 'emerald',
  update: 'blue',
  delete: 'red',
  assign_permissions: 'violet',
  login: 'cyan',
  logout: 'slate',
}

function JsonDiff({ value }: { value: unknown }) {
  if (value == null) return <span className="text-surface-400">—</span>
  return (
    <pre className="max-h-56 overflow-auto rounded-lg bg-surface-900 p-3 text-[11px] leading-relaxed text-emerald-300">
      {JSON.stringify(value, null, 2)}
    </pre>
  )
}

export function AuditLogsPage() {
  const [selected, setSelected] = useState<AuditLogView | null>(null)
  const list = useListQuery<AuditLogView>({ queryKey: ['audit-logs'], fetcher: listAuditLogs })

  const columns: Column<AuditLogView>[] = [
    {
      key: 'entity',
      header: 'Entity',
      sortable: true,
      render: (a) => (
        <div>
          <p className="text-sm font-semibold text-surface-800">{humanize(a.entity)}</p>
          <p className="font-mono text-[11px] text-surface-400">#{shortId(a.entity_id)}</p>
        </div>
      ),
    },
    {
      key: 'action',
      header: 'Action',
      sortable: true,
      render: (a) => <Badge tone={ACTION_TONE[a.action] ?? 'slate'}>{humanize(a.action)}</Badge>,
    },
    {
      key: 'user_id',
      header: 'Actor',
      render: (a) => <span className="font-mono text-xs text-surface-500">{shortId(a.user_id)}</span>,
    },
    {
      key: 'method',
      header: 'Method',
      hideOnMobile: true,
      render: (a) => <Badge tone="slate">{a.method}</Badge>,
    },
    {
      key: 'path',
      header: 'Endpoint',
      hideOnMobile: true,
      render: (a) => <span className="block max-w-52 truncate font-mono text-xs text-surface-500">{a.path}</span>,
    },
    {
      key: 'ip',
      header: 'IP',
      hideOnMobile: true,
      render: (a) => <span className="font-mono text-xs text-surface-400">{a.ip || '—'}</span>,
    },
    {
      key: 'created_at',
      header: 'Time',
      sortable: true,
      render: (a) => <span className="text-xs text-surface-500">{formatDateTime(a.created_at)}</span>,
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Audit trail"
        description="Every mutating action, with before/after state."
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(a) => a.id}
        searchPlaceholder="Search audit trail…"
        emptyTitle="No audit entries"
        emptyMessage="Actions are recorded here as they happen."
        onRowClick={(a) => setSelected(a)}
        toolbarExtra={
          <span className="inline-flex items-center gap-1.5 rounded-lg bg-surface-100 px-2.5 py-1.5 text-xs font-medium text-surface-500">
            <GitCompareArrows className="size-3.5" /> Click a row for before/after
          </span>
        }
      />

      <Modal
        open={!!selected}
        onClose={() => setSelected(null)}
        title={`${humanize(selected?.action)} — ${humanize(selected?.entity)}`}
        description={`${selected?.method} ${selected?.path} · ${formatDateTime(selected?.created_at)}`}
        size="lg"
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <p className="mb-1.5 text-xs font-semibold text-surface-500">Before</p>
            <JsonDiff value={selected?.before} />
          </div>
          <div>
            <p className="mb-1.5 text-xs font-semibold text-surface-500">After</p>
            <JsonDiff value={selected?.after} />
          </div>
        </div>
        {selected?.before == null && selected?.after == null && (
          <p className="mt-4 text-center text-xs text-surface-400">
            No state change captured for this action (e.g. authentication events).
          </p>
        )}
      </Modal>
    </div>
  )
}
