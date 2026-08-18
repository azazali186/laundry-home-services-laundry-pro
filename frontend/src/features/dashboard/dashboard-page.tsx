import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Bike,
  Boxes,
  CalendarCheck,
  CircleDollarSign,
  Package,
  Receipt,
  Shirt,
  ShoppingBag,
  Truck,
} from 'lucide-react'
import { fetchDashboard, type DashboardData } from '@/services/analytics.service'
import { useShopCurrency } from '@/hooks/useShopCurrency'
import { formatCurrency, formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'
import { Card, CardBody, CardHeader } from '@/components/ui/card'
import { StatCard } from '@/components/ui/stat-card'
import { ErrorState, Skeleton } from '@/components/ui/feedback'
import { RevenueChart } from '@/components/charts/revenue-chart'
import { StatusDonut } from '@/components/charts/status-donut'
import { TopServices } from '@/components/charts/top-services'

const RANGES = [
  { key: '7d', label: '7D', days: 7 },
  { key: '30d', label: '30D', days: 30 },
  { key: '90d', label: '90D', days: 90 },
  { key: '12m', label: '12M', days: 365 },
] as const

type RangeKey = (typeof RANGES)[number]['key']

function rangeDates(days: number): { from: string; to: string } {
  const to = new Date()
  const from = new Date()
  from.setDate(to.getDate() - days)
  return { from: from.toISOString().slice(0, 10), to: to.toISOString().slice(0, 10) }
}

export function DashboardPage() {
  const [range, setRange] = useState<RangeKey>('30d')

  const { from, to } = useMemo(() => {
    const r = RANGES.find((x) => x.key === range)!
    return rangeDates(r.days)
  }, [range])

  const query = useQuery({
    queryKey: ['dashboard', from, to],
    queryFn: () => fetchDashboard({ from, to }),
  })

  const data = query.data

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-surface-900">Overview</h1>
          <p className="mt-0.5 text-sm text-surface-500">
            Live snapshot of your operation for the selected period.
          </p>
        </div>
        <div className="inline-flex items-center gap-1 rounded-lg border border-surface-200 bg-white p-1 shadow-sm">
          {RANGES.map((r) => (
            <button
              key={r.key}
              onClick={() => setRange(r.key)}
              className={cn(
                'rounded-md px-3 py-1.5 text-xs font-semibold transition-all duration-150',
                range === r.key
                  ? 'bg-surface-900 text-white shadow-sm'
                  : 'text-surface-500 hover:text-surface-800',
              )}
            >
              {r.label}
            </button>
          ))}
        </div>
      </div>

      {query.isLoading ? (
        <DashboardSkeleton />
      ) : query.isError ? (
        <Card>
          <ErrorState message="Couldn't load the dashboard" detail="Check your connection and try again." onRetry={() => void query.refetch()} />
        </Card>
      ) : data ? (
        <DashboardContent data={data} />
      ) : null}
    </div>
  )
}

function DashboardContent({ data }: { data: DashboardData }) {
  const { period, revenue_trend, orders_by_status, top_services, operational, bookings_by_month } = data
  const currency = useShopCurrency()
  const bookingsTotal = bookings_by_month.reduce((s, b) => s + b.count, 0)
  const pipeline = [
    { label: 'Pickups pending', value: operational.pickups_pending, icon: Truck, accent: 'amber' as const },
    { label: 'Deliveries out', value: operational.deliveries_out, icon: Bike, accent: 'blue' as const },
    { label: 'Garments in queue', value: operational.garments_in_queue, icon: Shirt, accent: 'violet' as const },
    { label: 'Unread invoices', value: operational.unread_invoices, icon: Receipt, accent: 'red' as const },
    { label: 'Low stock items', value: operational.low_stock_items, icon: Boxes, accent: 'amber' as const },
  ]

  return (
    <div className="space-y-5 animate-fade-in">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label="Revenue"
          value={formatCurrency(period.revenue, currency)}
          sub={`${period.from.slice(0, 10)} → ${period.to.slice(0, 10)}`}
          icon={<CircleDollarSign className="size-5" />}
          accent="brand"
        />
        <StatCard label="Orders" value={formatNumber(period.orders)} sub={`${formatNumber(period.bookings)} bookings`} icon={<ShoppingBag className="size-5" />} accent="blue" />
        <StatCard label="Avg order value" value={formatCurrency(period.avg_order_value, currency)} icon={<Package className="size-5" />} accent="violet" />
        <StatCard label="Bookings" value={formatNumber(bookingsTotal)} sub={`across ${bookings_by_month.length} months`} icon={<CalendarCheck className="size-5" />} accent="emerald" />
      </div>

      <div className="grid grid-cols-1 gap-5 xl:grid-cols-3">
        <Card className="xl:col-span-2">
          <CardHeader title="Revenue trend" description="Daily revenue for the selected period" />
          <CardBody>
            <RevenueChart data={revenue_trend} currency={currency} />
          </CardBody>
        </Card>

        <Card>
          <CardHeader title="Orders by status" description="Distribution of order states" />
          <CardBody>
            <StatusDonut data={orders_by_status} />
          </CardBody>
        </Card>
      </div>

      <div className="grid grid-cols-1 gap-5 xl:grid-cols-3">
        <Card className="xl:col-span-2">
          <CardHeader title="Top services" description="Highest revenue service types" />
          <CardBody>
            <TopServices data={top_services} currency={currency} />
          </CardBody>
        </Card>

        <Card>
          <CardHeader title="Live pipeline" description="Current operational counts" />
          <CardBody className="space-y-3">
            {pipeline.map((p) => (
              <div key={p.label} className="flex items-center justify-between rounded-xl border border-surface-100 px-3.5 py-2.5">
                <span className="flex items-center gap-2.5 text-xs font-medium text-surface-600">
                  <span className="flex size-7 items-center justify-center rounded-lg bg-surface-100 text-surface-500">
                    <p.icon className="size-3.5" />
                  </span>
                  {p.label}
                </span>
                <span className="text-sm font-bold tabular-nums text-surface-900">{formatNumber(p.value)}</span>
              </div>
            ))}
          </CardBody>
        </Card>
      </div>
    </div>
  )
}

function DashboardSkeleton() {
  return (
    <div className="space-y-5">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i} className="p-5">
            <Skeleton className="h-3 w-24" />
            <Skeleton className="mt-3 h-7 w-32" />
            <Skeleton className="mt-2 h-3 w-20" />
          </Card>
        ))}
      </div>
      <div className="grid grid-cols-1 gap-5 xl:grid-cols-3">
        <Card className="xl:col-span-2 p-5">
          <Skeleton className="h-4 w-36" />
          <Skeleton className="mt-5 h-52 w-full" />
        </Card>
        <Card className="p-5">
          <Skeleton className="h-4 w-36" />
          <Skeleton className="mt-5 h-52 w-full rounded-full" />
        </Card>
      </div>
    </div>
  )
}
