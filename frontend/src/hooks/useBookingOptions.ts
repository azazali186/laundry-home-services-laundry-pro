import { useQuery } from '@tanstack/react-query'
import { fetchAllPages } from '@/lib/fetchAll'
import { listBookings, type BookingView } from '@/services/bookings.service'

/** Recent bookings for form selects (pickups). */
export function useBookingOptions() {
  return useQuery({
    queryKey: ['bookings', 'options'],
    queryFn: () => fetchAllPages<BookingView>(listBookings, { limit: 100, sort_by: 'created_at', sort_order: 'desc' }),
    staleTime: 5 * 60_000,
  })
}
