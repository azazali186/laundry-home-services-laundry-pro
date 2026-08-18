import { AlertTriangle, Inbox, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from './button'

export function Spinner({ className }: { className?: string }) {
  return (
    <span
      role="status"
      aria-label="Loading"
      className={cn(
        'inline-block size-5 animate-spin rounded-full border-2 border-surface-300 border-t-brand-600',
        className,
      )}
    />
  )
}

const shimmer = 'animate-shimmer bg-gradient-to-r from-surface-100 via-surface-200 to-surface-100 bg-[length:200%_100%]'

export function Skeleton({ className }: { className?: string }) {
  return <div className={cn('rounded-md', shimmer, className)} />
}

export function SkeletonRows({ rows = 5, cols = 4 }: { rows?: number; cols?: number }) {
  return (
    <div className="divide-y divide-surface-100">
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} className="flex items-center gap-6 px-6 py-4">
          {Array.from({ length: cols }).map((_, c) => (
            <Skeleton key={c} className={cn('h-4', c === 0 ? 'w-40' : 'flex-1')} />
          ))}
        </div>
      ))}
    </div>
  )
}

export function EmptyState({
  title = 'Nothing here yet',
  message,
  action,
  icon,
}: {
  title?: string
  message?: string
  action?: React.ReactNode
  icon?: React.ReactNode
}) {
  return (
    <div className="flex flex-col items-center justify-center px-6 py-16 text-center animate-fade-in">
      <div className="mb-3 flex size-12 items-center justify-center rounded-2xl bg-surface-100 text-surface-400">
        {icon ?? <Inbox className="size-6" aria-hidden />}
      </div>
      <p className="text-sm font-semibold text-surface-800">{title}</p>
      {message && <p className="mt-1 max-w-sm text-xs text-surface-500">{message}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  )
}

export function ErrorState({
  message = 'We couldn\u2019t load this data.',
  detail,
  onRetry,
  className,
}: {
  message?: string
  detail?: string
  onRetry?: () => void
  className?: string
}) {
  return (
    <div className={cn('flex flex-col items-center justify-center px-6 py-14 text-center', className)}>
      <div className="mb-3 flex size-12 items-center justify-center rounded-2xl bg-red-50 text-red-500">
        <AlertTriangle className="size-6" aria-hidden />
      </div>
      <p className="text-sm font-semibold text-surface-800">{message}</p>
      {detail && <p className="mt-1 max-w-sm text-xs text-surface-500">{detail}</p>}
      {onRetry && (
        <Button variant="outline" size="sm" className="mt-4" onClick={onRetry}>
          <RefreshCw className="size-3.5" aria-hidden /> Try again
        </Button>
      )}
    </div>
  )
}
