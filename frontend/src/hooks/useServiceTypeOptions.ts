import { useQuery } from '@tanstack/react-query'
import { listServiceTypes } from '@/services/service-types.service'

/** Active service-type options for pickers. */
export function useServiceTypeOptions() {
  return useQuery({
    queryKey: ['service-types', 'options'],
    queryFn: async () => {
      const res = await listServiceTypes({ limit: 100, sort_by: 'name', sort_order: 'asc' })
      return res.items.filter((s) => s.is_active)
    },
    staleTime: 10 * 60_000,
  })
}
