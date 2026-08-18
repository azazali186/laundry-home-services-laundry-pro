import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { BellPlus, Check, Send } from 'lucide-react'
import { useListQuery } from '@/hooks/useListQuery'
import { usePermissions } from '@/hooks/usePermissions'
import { listNotifications, createNotification, markNotificationRead, NOTIFICATION_TYPES, NOTIFICATION_CHANNELS, type NotificationSummary, type NotificationView } from '@/services/notifications.service'
import { fetchAllPages } from '@/lib/fetchAll'
import { listUsers } from '@/services/users.service'
import type { UserView } from '@/services/auth.service'
import { errorMessage } from '@/lib/api'
import { formatDateTime, formatNumber } from '@/lib/format'
import { humanize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { DataTable, type Column } from '@/components/data-table/data-table'
import { SummaryBar } from '@/components/data-table/summary-bar'
import { Modal } from '@/components/ui/modal'
import { Field, Input, Select, Textarea } from '@/components/ui/form'
import { PageHeader } from '@/components/ui/page-header'
import { useToast } from '@/components/ui/toast'

export function NotificationsPage() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { can } = usePermissions()

  const [createOpen, setCreateOpen] = useState(false)
  const [userId, setUserId] = useState('')
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [type, setType] = useState('info')
  const [channel, setChannel] = useState('in_app')

  const list = useListQuery<NotificationView, NotificationSummary>({ queryKey: ['notifications'], fetcher: listNotifications })

  const usersQuery = useQuery({
    queryKey: ['users', 'all'],
    queryFn: () => fetchAllPages<UserView>(listUsers, { limit: 100, sort_by: 'name', maxPages: 10 }),
    staleTime: 5 * 60_000,
  })

  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['notifications'] })

  const markReadMut = useMutation({
    mutationFn: (id: string) => markNotificationRead(id),
    // Optimistic: flip unread → read immediately across every cached page.
    onMutate: (id: string) => {
      queryClient.setQueriesData<import('@/lib/types').ListResult<NotificationView, NotificationSummary>>(
        { queryKey: ['notifications'] },
        (old) => {
          if (!old) return old
          const now = new Date().toISOString()
          return {
            ...old,
            items: old.items.map((n) => (n.id === id ? { ...n, read_at: n.read_at ?? now } : n)),
          }
        },
      )
    },
    onSuccess: () => invalidate(),
    onError: (err) => toast.toast({ title: 'Action failed', description: errorMessage(err), variant: 'error' }),
  })

  const createMut = useMutation({
    mutationFn: () =>
      createNotification({ user_id: userId, title: title.trim(), body: body.trim(), type, channel }),
    onSuccess: () => {
      toast.toast({ title: 'Notification sent', variant: 'success' })
      invalidate()
      setCreateOpen(false)
      setTitle('')
      setBody('')
    },
    onError: (err) => toast.toast({ title: 'Send failed', description: errorMessage(err), variant: 'error' }),
  })

  const summary = list.summary
  const canWrite = can('POST /notifications')

  const columns: Column<NotificationView>[] = [
    {
      key: 'title',
      header: 'Notification',
      render: (n) => (
        <div className="flex items-start gap-2.5">
          <span className={`mt-1.5 size-2 shrink-0 rounded-full ${n.read_at ? 'bg-surface-200' : 'bg-brand-500'}`} />
          <div className="min-w-0">
            <p className={`text-sm ${n.read_at ? 'text-surface-600' : 'font-semibold text-surface-900'}`}>{n.title}</p>
            <p className="truncate text-xs text-surface-400">{n.body}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'type',
      header: 'Type',
      sortable: true,
      filterOptions: NOTIFICATION_TYPES.map((t) => ({ value: t, label: humanize(t) })),
      render: (n) => <Badge tone={n.type === 'error' ? 'red' : n.type === 'warning' ? 'amber' : n.type === 'success' ? 'emerald' : 'blue'}>{humanize(n.type)}</Badge>,
    },
    {
      key: 'channel',
      header: 'Channel',
      sortable: true,
      filterOptions: NOTIFICATION_CHANNELS.map((c) => ({ value: c, label: humanize(c) })),
      render: (n) => <Badge tone="slate">{humanize(n.channel)}</Badge>,
    },
    {
      key: 'status',
      header: 'Delivery',
      sortable: true,
      render: (n) => <Badge tone={n.status === 'failed' ? 'red' : n.status === 'sent' ? 'emerald' : 'amber'}>{humanize(n.status)}</Badge>,
    },
    {
      key: 'created_at',
      header: 'Created',
      sortable: true,
      hideOnMobile: true,
      render: (n) => <span className="text-xs text-surface-500">{formatDateTime(n.created_at)}</span>,
    },
  ]

  return (
    <div className="space-y-5">
      <PageHeader
        title="Notifications"
        description="Outbound messages to users."
        actions={
          canWrite && (
            <Button onClick={() => setCreateOpen(true)}>
              <BellPlus className="size-4" /> Send notification
            </Button>
          )
        }
      />

      <DataTable
        columns={columns}
        list={list}
        rowKey={(n) => n.id}
        searchPlaceholder="Search notifications…"
        emptyTitle="No notifications"
        emptyMessage="Send a notification to a user."
        summary={
          summary && (
            <SummaryBar
              chips={[
                { label: 'Unread', value: formatNumber(summary.unread), tone: 'brand' },
                { label: 'Pending', value: formatNumber(summary.pending), tone: 'amber' },
                { label: 'Sent', value: formatNumber(summary.sent), tone: 'emerald' },
                { label: 'Failed', value: formatNumber(summary.failed), tone: 'red' },
              ]}
            />
          )
        }
        rowActions={
          canWrite
            ? (n) =>
                n.read_at
                  ? []
                  : [
                      {
                        key: 'read',
                        label: 'Mark as read',
                        icon: <Check className="size-3.5" />,
                        onClick: () => void markReadMut.mutate(n.id),
                      },
                    ]
            : undefined
        }
      />

      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Send notification"
        description="Deliver a message through the chosen channel."
        size="md"
        footer={
          <>
            <Button variant="ghost" onClick={() => setCreateOpen(false)} disabled={createMut.isPending}>
              Cancel
            </Button>
            <Button onClick={() => void createMut.mutate()} loading={createMut.isPending} disabled={!userId || !title.trim()}>
              <Send className="size-3.5" /> Send
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Field label="Recipient" required>
            <Select value={userId} onChange={(e) => setUserId(e.target.value)}>
              <option value="">Select user…</option>
              {usersQuery.data?.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.name} — {u.email}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Title" required>
            <Input value={title} onChange={(e) => setTitle(e.target.value)} autoFocus />
          </Field>
          <Field label="Body">
            <Textarea value={body} onChange={(e) => setBody(e.target.value)} rows={3} />
          </Field>
          <div className="grid grid-cols-2 gap-4">
            <Field label="Type">
              <Select value={type} onChange={(e) => setType(e.target.value)} options={NOTIFICATION_TYPES.map((t) => ({ value: t, label: humanize(t) }))} />
            </Field>
            <Field label="Channel">
              <Select value={channel} onChange={(e) => setChannel(e.target.value)} options={NOTIFICATION_CHANNELS.map((c) => ({ value: c, label: humanize(c) }))} />
            </Field>
          </div>
        </div>
      </Modal>
    </div>
  )
}
