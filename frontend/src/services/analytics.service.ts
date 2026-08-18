import { postRequest } from '@/lib/api'

export interface DashboardRequest {
  from?: string
  to?: string
}

export interface DashboardData {
  period: {
    from: string
    to: string
    revenue: number
    orders: number
    bookings: number
    avg_order_value: number
  }
  revenue_trend: Array<{ day: string; revenue: number; orders: number }>
  orders_by_status: Array<{ status: string; count: number }>
  bookings_by_month: Array<{ status: string; count: number }>
  top_services: Array<{ service_type_id: string; orders: number; revenue: number }>
  operational: {
    pickups_pending: number
    deliveries_out: number
    garments_in_queue: number
    unread_invoices: number
    low_stock_items: number
  }
}

export async function fetchDashboard(req: DashboardRequest): Promise<DashboardData> {
  const data = await postRequest<DashboardRequest, DashboardData>('/analytics/dashboard', req)
  // The backend serializes empty aggregates as null; normalize to arrays so
  // consumers never have to guard against missing fields.
  return {
    ...data,
    period: data.period ?? {
      from: '',
      to: '',
      revenue: 0,
      orders: 0,
      bookings: 0,
      avg_order_value: 0,
    },
    revenue_trend: data.revenue_trend ?? [],
    orders_by_status: data.orders_by_status ?? [],
    bookings_by_month: data.bookings_by_month ?? [],
    top_services: data.top_services ?? [],
    operational: data.operational ?? {
      pickups_pending: 0,
      deliveries_out: 0,
      garments_in_queue: 0,
      unread_invoices: 0,
      low_stock_items: 0,
    },
  }
}
