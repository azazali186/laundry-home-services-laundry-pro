import { useState } from 'react'
import { KeyRound, ShieldCheck } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { errorMessage } from '@/lib/api'
import { useAuth } from '@/hooks/useAuth'
import { changePassword } from '@/services/auth.service'
import { Button } from '@/components/ui/button'
import { Field, Input } from '@/components/ui/form'
import { Card, CardBody, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { PageHeader } from '@/components/ui/page-header'
import { useToast } from '@/components/ui/toast'
import { formatDate } from '@/lib/format'
import { ShopConfigCard } from './shop-config-card'

export function SettingsPage() {
  const { user } = useAuth()
  const toast = useToast()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: () => changePassword({ current_password: current, new_password: next }),
    onSuccess: () => {
      toast.toast({ title: 'Password updated', variant: 'success' })
      setCurrent('')
      setNext('')
      setConfirm('')
    },
    onError: (err) => setError(errorMessage(err)),
  })

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    if (next.length < 8) {
      setError('New password must be at least 8 characters')
      return
    }
    if (next !== confirm) {
      setError('New passwords do not match')
      return
    }
    void mutation.mutate()
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <PageHeader title="Settings" description="Shop configuration, your profile and security preferences." />

      <ShopConfigCard />

      <Card>
        <CardHeader title="Profile" description="How you appear in the console." />
        <CardBody>
          <dl className="grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2">
            <div>
              <dt className="text-xs font-medium text-surface-400">Name</dt>
              <dd className="mt-0.5 text-sm font-medium text-surface-800">{user?.name ?? '—'}</dd>
            </div>
            <div>
              <dt className="text-xs font-medium text-surface-400">Email</dt>
              <dd className="mt-0.5 text-sm font-medium text-surface-800">{user?.email ?? '—'}</dd>
            </div>
            <div>
              <dt className="text-xs font-medium text-surface-400">Phone</dt>
              <dd className="mt-0.5 text-sm font-medium text-surface-800">{user?.phone || '—'}</dd>
            </div>
            <div>
              <dt className="text-xs font-medium text-surface-400">Member since</dt>
              <dd className="mt-0.5 text-sm font-medium text-surface-800">{formatDate(user?.created_at)}</dd>
            </div>
            <div className="sm:col-span-2">
              <dt className="mb-1.5 text-xs font-medium text-surface-400">Roles</dt>
              <dd className="flex flex-wrap gap-1.5">
                {(user?.roles?.length ?? 0) > 0 ? (
                  user!.roles!.map((r) => <Badge key={r.id} tone="brand">{r.name}</Badge>)
                ) : (
                  <Badge tone="slate">No roles</Badge>
                )}
              </dd>
            </div>
          </dl>
        </CardBody>
      </Card>

      <Card>
        <CardHeader
          title={
            <span className="inline-flex items-center gap-2">
              <KeyRound className="size-4 text-surface-400" /> Change password
            </span>
          }
          description={
            <span className="inline-flex items-center gap-1.5">
              <ShieldCheck className="size-3.5" /> Your session stays active after rotation.
            </span>
          }
        />
        <CardBody>
          <form onSubmit={onSubmit} className="space-y-4">
            <Field label="Current password" required>
              <Input type="password" value={current} onChange={(e) => setCurrent(e.target.value)} autoComplete="current-password" />
            </Field>
            <Field label="New password" required hint="Minimum 8 characters.">
              <Input type="password" value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" />
            </Field>
            <Field label="Confirm new password" required>
              <Input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" />
            </Field>
            {error && (
              <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2.5 text-xs font-medium text-red-700">
                {error}
              </div>
            )}
            <Button type="submit" loading={mutation.isPending} disabled={!current || !next || !confirm}>
              Update password
            </Button>
          </form>
        </CardBody>
      </Card>
    </div>
  )
}
