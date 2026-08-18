import type { PricingComboView, PricingServiceView } from '@/services/pricing.service'

/** One rendered line in a plan's "what's included" list. */
export interface IncludedLine {
  label: string
  unit?: string
  quantity: number
  currency?: string
  /** quantity × unit price — the à-la-carte cost of this line. */
  lineValue?: number
}

/**
 * Normalize a combo's items into display lines. Prefers the service snapshot
 * the backend attaches to each item (name, unit, unit price); when those are
 * absent (e.g. a stale catalog cache served right after a deploy), falls back
 * to a join against the live services catalog. Every price still originates
 * from the backend.
 */
export interface PerUnitHint {
  label: string
  value: number
}

/**
 * Effective per-unit price of a bundle ("≈ ₹125/load") — the combo price
 * divided by its total included quantity. Only returned when every included
 * service shares the same unit, so the hint never mixes loads and items;
 * missing snapshot units (stale cache) also suppress it. Purely derived from
 * backend-sourced prices.
 */
export function perUnitHint(combo: PricingComboView): PerUnitHint | undefined {
  const items = combo.items ?? []
  if (items.length === 0 || combo.price <= 0) return undefined
  const totalQty = items.reduce((sum, it) => sum + (it.quantity || 0), 0)
  if (totalQty <= 0) return undefined
  const units = new Set(items.map((it) => it.unit || '').filter(Boolean))
  if (units.size !== 1) return undefined
  const unit = [...units][0]
  // Wash & fold passes are marketed per "load"; everything else keeps its
  // plain unit label (per_item -> item).
  const label = unit === 'kg' ? 'load' : unit.replace(/^per_/, '')
  return { label, value: Math.round(combo.price / totalQty) }
}

export function buildIncludedLines(
  combo: PricingComboView,
  services?: PricingServiceView[],
): IncludedLine[] {
  const byID = new Map((services ?? []).map((s) => [s.id, s]))
  return (combo.items ?? []).map((item) => {
    const svc = byID.get(item.service_type_id)
    // Empty-string / zero snapshot values mean the backend didn't populate
    // them (e.g. a catalog snapshot cached by an older server version right
    // after a deploy) — fall back to the live services catalog then.
    const label = item.service_name || svc?.name || 'Service'
    const unit = item.unit || svc?.unit
    const price = item.price || svc?.price
    const lineValue = price === undefined ? undefined : item.quantity * price
    return {
      label,
      unit,
      quantity: item.quantity,
      currency: item.currency || svc?.currency || combo.currency,
      lineValue,
    }
  })
}
