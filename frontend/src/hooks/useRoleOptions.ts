import { useQuery } from '@tanstack/react-query'
import { listRoles } from '@/services/roles.service'

/** Role options for pickers — fetches the first page at max page size. */
export function useRoleOptions() {
  return useQuery({
    queryKey: ['roles', 'options'],
    queryFn: async () => {
      const res = await listRoles({ limit: 100, sort_by: 'name', sort_order: 'asc' })
      return res.items
    },
    staleTime: 10 * 60_000,
  })
}
