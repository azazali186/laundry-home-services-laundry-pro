import { useQuery } from '@tanstack/react-query'
import { fetchPublicPricing } from '@/services/pricing.service'

/**
 * Public pricing catalog for the landing page. Every price shown on the
 * marketing site comes from the backend's /pricing endpoint — there are no
 * hardcoded rates in the frontend.
 */
export function usePublicPricing() {
  return useQuery({
    queryKey: ['public-pricing'],
    queryFn: fetchPublicPricing,
    staleTime: 5 * 60_000,
    retry: 2,
    refetchOnWindowFocus: false,
  })
}
