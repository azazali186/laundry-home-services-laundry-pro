import { cn } from '@/lib/utils'

export interface SummaryChip {
  label: string
  value: React.ReactNode
  tone?: 'default' | 'emerald' | 'amber' | 'red' | 'blue' | 'brand' | 'cyan' | 'violet'
}

const toneClasses = {
  default: 'text-surface-600',
  emerald: 'text-emerald-600',
  amber: 'text-amber-600',
  red: 'text-red-600',
  blue: 'text-blue-600',
  brand: 'text-brand-600',
  cyan: 'text-cyan-600',
  violet: 'text-violet-600',
}

/** Horizontal row of small summary chips summarizing the filtered dataset. */
export function SummaryBar({ chips }: { chips: SummaryChip[] }) {
  if (!chips.length) return null
  return (
    <div className="flex flex-wrap items-center gap-x-5 gap-y-1.5 border-b border-surface-100 bg-surface-50/60 px-5 py-2.5">
      {chips.map((chip) => (
        <span key={chip.label} className="flex items-baseline gap-1.5 text-xs">
          <span className="font-medium text-surface-500">{chip.label}</span>
          <span className={cn('font-semibold tabular-nums', toneClasses[chip.tone ?? 'default'])}>
            {chip.value}
          </span>
        </span>
      ))}
    </div>
  )
}
