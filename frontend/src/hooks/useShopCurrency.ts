import { useQuery } from '@tanstack/react-query'
import { fetchShopSettings } from '@/services/settings.service'

/**
 * The shop's configured currency, loaded from the backend /settings/view
 * endpoint. Shares the ['settings'] cache entry with the shop configuration
 * page, so the settings are fetched at most once per session.
 *
 * All money display in the admin app should format through this hook —
 * never hardcode a currency code.
 */
export function useShopCurrency(): string {
  const query = useQuery({
    queryKey: ['settings'],
    queryFn: fetchShopSettings,
    staleTime: 60_000,
  })
  return query.data?.currency ?? 'INR'
}
