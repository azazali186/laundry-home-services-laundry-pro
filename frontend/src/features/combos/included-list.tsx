import type { ComboItemView } from '@/services/combos.service'

const MAX_LINES = 3

/** Compact "what's included" list for the combos table. Every item line is
 * sourced from the backend's combo item snapshot (service_name, quantity). */
export function IncludedList({ items }: { items: ComboItemView[] }) {
  const all = items ?? []
  const lines = all.slice(0, MAX_LINES)
  const rest = all.length - lines.length
  return (
    <div className="space-y-0.5">
      {lines.map((item, i) => (
        <p key={`${item.service_type_id}-${i}`} className="truncate text-xs text-surface-500">
          <span className="font-medium text-surface-600">{item.quantity} ×</span>{' '}
          {item.service_name || 'Service'}
        </p>
      ))}
      {rest > 0 && <p className="text-[11px] text-surface-400">+{rest} more</p>}
    </div>
  )
}
