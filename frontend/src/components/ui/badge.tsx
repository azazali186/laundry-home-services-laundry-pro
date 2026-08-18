import { cn, humanize } from '@/lib/utils'
import { statusTone } from '@/lib/status'

export type BadgeTone = 'slate' | 'blue' | 'emerald' | 'red' | 'amber' | 'violet' | 'cyan' | 'brand'

const toneClasses: Record<BadgeTone, string> = {
  slate: 'bg-surface-100 text-surface-600 ring-surface-200',
  blue: 'bg-blue-50 text-blue-700 ring-blue-200',
  emerald: 'bg-emerald-50 text-emerald-700 ring-emerald-200',
  red: 'bg-red-50 text-red-700 ring-red-200',
  amber: 'bg-amber-50 text-amber-800 ring-amber-200',
  violet: 'bg-violet-50 text-violet-700 ring-violet-200',
  cyan: 'bg-cyan-50 text-cyan-700 ring-cyan-200',
  brand: 'bg-brand-50 text-brand-700 ring-brand-200',
}

export function Badge({
  tone = 'slate',
  className,
  children,
}: {
  tone?: BadgeTone
  className?: string
  children: React.ReactNode
}) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold ring-1 ring-inset',
        toneClasses[tone],
        className,
      )}
    >
      {children}
    </span>
  )
}

/** Dot indicator used inside status badges. */
export function StatusDot({ tone = 'slate' }: { tone?: BadgeTone }) {
  const dot: Record<BadgeTone, string> = {
    slate: 'bg-surface-400',
    blue: 'bg-blue-500',
    emerald: 'bg-emerald-500',
    red: 'bg-red-500',
    amber: 'bg-amber-500',
    violet: 'bg-violet-500',
    cyan: 'bg-cyan-500',
    brand: 'bg-brand-600',
  }
  return <span className={cn('size-1.5 rounded-full', dot[tone])} aria-hidden />
}

/** Status badge that derives its tone from the status string. */
export function StatusBadge({ status, className }: { status?: string | null; className?: string }) {
  if (!status) return <Badge tone="slate">—</Badge>
  return (
    <Badge tone={statusTone(status)} className={className}>
      <StatusDot tone={statusTone(status)} />
      {humanize(status)}
    </Badge>
  )
}
