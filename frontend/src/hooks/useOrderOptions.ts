import { useQuery } from '@tanstack/react-query'
import { fetchAllPages } from '@/lib/fetchAll'
import { listOrders, type OrderView } from '@/services/orders.service'

/** Recent orders for form selects (deliveries, garments, ratings). */
export function useOrderOptions() {
  return useQuery({
    queryKey: ['orders', 'options'],
    queryFn: () => fetchAllPages<OrderView>(listOrders, { limit: 100, sort_by: 'created_at', sort_order: 'desc' }),
    staleTime: 5 * 60_000,
  })
}
