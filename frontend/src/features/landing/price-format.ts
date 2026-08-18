/** Landing-specific formatting for prices coming from the backend catalog. */

const UNIT_SUFFIX: Record<string, string> = {
  kg: '/kg',
  per_item: '/item',
  per_week: '/week',
  per_month: '/month',
  per_bundle: '/bundle',
  flat: '',
}

const UNIT_LABEL: Record<string, string> = {
  kg: 'per kg',
  per_item: 'per item',
  per_week: 'per week',
  per_month: 'per month',
  per_bundle: 'per bundle',
  flat: 'one time',
}

/** "₹99" / "₹1,299" — whole numbers without decimals. */
export function formatLandingPrice(price?: number | null, currency?: string): string {
  if (price === undefined || price === null || Number.isNaN(price)) return '—'
  const fmt = new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency: currency || 'INR',
    maximumFractionDigits: Number.isInteger(price) ? 0 : 2,
  })
  return fmt.format(price)
}

/** Short suffix for card chips, e.g. "/kg". */
export function unitSuffix(unit?: string | null): string {
  if (!unit) return ''
  return UNIT_SUFFIX[unit] ?? `/${unit.replace(/^per_/, '')}`
}

/** Full label for pricing cards, e.g. "per week". */
export function humanizeUnit(unit?: string | null): string {
  if (!unit) return ''
  return UNIT_LABEL[unit] ?? unit.replace(/[_-]/g, ' ')
}
