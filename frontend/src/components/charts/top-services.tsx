import { formatCurrency } from '@/lib/format'
import { shortId } from '@/lib/format'

export interface TopService {
  service_type_id: string
  orders: number
  revenue: number
}

/** Top services by revenue — simple accessible bar list (no chart lib). */
export function TopServices({ data, currency }: { data: TopService[]; currency?: string }) {
  if (!data.length) {
    return <div className="flex h-40 items-center justify-center text-xs text-surface-400">No services yet</div>
  }
  const max = Math.max(...data.map((d) => d.revenue), 1)

  return (
    <ul className="space-y-3">
      {data.map((s, i) => (
        <li key={s.service_type_id}>
          <div className="mb-1 flex items-baseline justify-between gap-2 text-xs">
            <span className="font-medium text-surface-700">
              <span className="mr-1.5 text-surface-300">#{i + 1}</span>
              {shortId(s.service_type_id)}
            </span>
            <span className="tabular-nums text-surface-600">
              {formatCurrency(s.revenue, currency)}
              <span className="ml-1.5 text-surface-400">· {s.orders} orders</span>
            </span>
          </div>
          <div className="h-1.5 overflow-hidden rounded-full bg-surface-100">
            <div
              className="h-full rounded-full bg-gradient-to-r from-brand-500 to-brand-400 transition-all duration-500"
              style={{ width: `${Math.max((s.revenue / max) * 100, 4)}%` }}
            />
          </div>
        </li>
      ))}
    </ul>
  )
}
