import { cn } from '@/lib/utils'
import { Card } from './card'

export interface StatCardProps {
  label: string
  value: React.ReactNode
  sub?: React.ReactNode
  icon?: React.ReactNode
  accent?: 'brand' | 'emerald' | 'amber' | 'red' | 'blue' | 'violet'
}

const accentClasses = {
  brand: 'bg-brand-50 text-brand-600',
  emerald: 'bg-emerald-50 text-emerald-600',
  amber: 'bg-amber-50 text-amber-600',
  red: 'bg-red-50 text-red-600',
  blue: 'bg-blue-50 text-blue-600',
  violet: 'bg-violet-50 text-violet-600',
}

export function StatCard({ label, value, sub, icon, accent = 'brand' }: StatCardProps) {
  return (
    <Card className="relative overflow-hidden p-5 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-pop">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-xs font-medium text-surface-500">{label}</p>
          <p className="mt-1.5 text-2xl font-bold tracking-tight text-surface-900">{value}</p>
          {sub && <p className="mt-1 text-xs text-surface-500">{sub}</p>}
        </div>
        {icon && (
          <div
            className={cn(
              'flex size-10 shrink-0 items-center justify-center rounded-xl',
              accentClasses[accent],
            )}
          >
            {icon}
          </div>
        )}
      </div>
    </Card>
  )
}
