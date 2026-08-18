import { postRequest } from '@/lib/api'

/** Public service-type row served by the unauthenticated /pricing endpoint. */
export interface PricingServiceView {
  id: string
  name: string
  code: string
  description: string
  unit: string
  price: number
  currency: string
  tax_rate: number
  estimated_days: number
  is_active: boolean
  icon: string
}

export interface PricingComboItem {
  service_type_id: string
  quantity: number
  /** Service snapshot — what the bundle includes, straight from the backend. */
  service_name?: string
  unit?: string
  price?: number
  currency?: string
  icon?: string
}

/** Public combo (priced bundle) row. */
export interface PricingComboView {
  id: string
  name: string
  code: string
  description: string
  unit: string
  price: number
  currency: string
  is_active: boolean
  is_featured: boolean
  validity_days: number
  renewal_type: string
  validity_text: string
  items: PricingComboItem[]
  /** À-la-carte value of the included items; savings = regular_price − price. */
  regular_price?: number
  savings?: number
}

/** The full public pricing catalog. */
export interface PublicPricingView {
  business_name: string
  currency: string
  tax_rate: number
  services: PricingServiceView[]
  combos: PricingComboView[]
}

/** Fetch the public catalog — all pricing on the landing page comes from here. */
export function fetchPublicPricing(): Promise<PublicPricingView> {
  return postRequest<Record<string, never>, PublicPricingView>('/pricing', {})
}
