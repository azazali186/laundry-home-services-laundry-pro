import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { formatCurrency } from '@/lib/format'

export interface RevenuePoint {
  day: string
  revenue: number
  orders: number
}

/** Revenue-per-day area chart. Tolerant of empty data. */
export function RevenueChart({ data, currency }: { data: RevenuePoint[]; currency?: string }) {
  if (!data.length) {
    return (
      <div className="flex h-56 items-center justify-center text-xs text-surface-400">
        No revenue data for this period
      </div>
    )
  }

  const chartData = data.map((p) => ({
    ...p,
    day: new Date(p.day + (p.day.length === 10 ? 'T00:00:00' : '')).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
    }),
  }))

  return (
    <div className="h-56 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={chartData} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
          <defs>
            <linearGradient id="rev" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#5850e8" stopOpacity={0.28} />
              <stop offset="100%" stopColor="#5850e8" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" vertical={false} />
          <XAxis
            dataKey="day"
            tick={{ fontSize: 11, fill: '#64748b' }}
            tickLine={false}
            axisLine={false}
            minTickGap={24}
          />
          <YAxis
            tick={{ fontSize: 11, fill: '#64748b' }}
            tickLine={false}
            axisLine={false}
            width={56}
            tickFormatter={(v: number) => formatCurrency(v, currency).replace(/\.00$/, '')}
          />
          <Tooltip
            formatter={(value, name) => [
              formatCurrency(Number(value), currency),
              String(name),
            ]}
            contentStyle={{
              borderRadius: 12,
              border: '1px solid #e2e8f0',
              boxShadow: '0 12px 24px -8px rgb(15 23 42 / 0.15)',
              fontSize: 12,
            }}
          />
          <Area
            type="monotone"
            dataKey="revenue"
            name="Revenue"
            stroke="#5850e8"
            strokeWidth={2}
            fill="url(#rev)"
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
