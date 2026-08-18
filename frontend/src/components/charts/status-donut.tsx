import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'
import { humanize } from '@/lib/utils'

const COLORS = ['#5850e8', '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#06b6d4', '#64748b']

export interface StatusSlice {
  status: string
  count: number
}

/** Donut of status distribution with a center total. */
export function StatusDonut({ data }: { data: StatusSlice[] }) {
  const total = data.reduce((sum, d) => sum + d.count, 0)
  if (!data.length || total === 0) {
    return <div className="flex h-52 items-center justify-center text-xs text-surface-400">No data</div>
  }

  return (
    <div className="relative h-52 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={data}
            dataKey="count"
            nameKey="status"
            innerRadius={56}
            outerRadius={80}
            paddingAngle={2}
            strokeWidth={0}
          >
            {data.map((entry, i) => (
              <Cell key={entry.status} fill={COLORS[i % COLORS.length]} />
            ))}
          </Pie>
          <Tooltip
            formatter={(value, name) => [String(value), humanize(String(name))]}
            contentStyle={{
              borderRadius: 12,
              border: '1px solid #e2e8f0',
              boxShadow: '0 12px 24px -8px rgb(15 23 42 / 0.15)',
              fontSize: 12,
            }}
          />
        </PieChart>
      </ResponsiveContainer>
      <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
        <span className="text-2xl font-bold tabular-nums text-surface-900">{total}</span>
        <span className="text-[10px] font-medium uppercase tracking-wider text-surface-400">Total</span>
      </div>
    </div>
  )
}
