/** Formatting helpers — Intl-based, safe on missing values. */

const currencyCache = new Map<string, Intl.NumberFormat>()
function currencyFmt(currency?: string): Intl.NumberFormat {
  const key = currency || 'USD'
  let fmt = currencyCache.get(key)
  if (!fmt) {
    fmt = new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: key,
      maximumFractionDigits: 2,
    })
    currencyCache.set(key, fmt)
  }
  return fmt
}

export function formatCurrency(value?: number | null, currency?: string): string {
  if (value === undefined || value === null || Number.isNaN(value)) return '—'
  return currencyFmt(currency).format(value)
}

export function formatNumber(value?: number | null): string {
  if (value === undefined || value === null || Number.isNaN(value)) return '—'
  return new Intl.NumberFormat('en-US').format(value)
}

const dateFmt = new Intl.DateTimeFormat('en-US', {
  month: 'short',
  day: 'numeric',
  year: 'numeric',
})

const dateTimeFmt = new Intl.DateTimeFormat('en-US', {
  month: 'short',
  day: 'numeric',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
})

export function formatDate(value?: string | null): string {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  return dateFmt.format(d)
}

export function formatDateTime(value?: string | null): string {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  return dateTimeFmt.format(d)
}

export function formatRelative(value?: string | null): string {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  const diff = Date.now() - d.getTime()
  const mins = Math.round(diff / 60_000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.round(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.round(hours / 24)
  if (days < 30) return `${days}d ago`
  return dateFmt.format(d)
}

/** Short id: 8-char prefix. */
export function shortId(id?: string | null): string {
  if (!id) return '—'
  return id.slice(0, 8)
}

/** Title-case a snake_case or dashed status code. */
export function humanize(slug?: string | null): string {
  if (!slug) return '—'
  return slug
    .replace(/[_-]+/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase())
}

/** "Showing 1–20 of 145" summary line. */
export function showingSummary(pageIndex: number, limit: number, total: number): string {
  if (total === 0) return 'Showing 0 results'
  const start = pageIndex * limit + 1
  const end = Math.min((pageIndex + 1) * limit, total)
  return `Showing ${start}–${end} of ${total}`
}

/** Format a float score (e.g. 4.6) with one decimal. */
export function formatScore(value?: number | null): string {
  if (value === undefined || value === null || Number.isNaN(value)) return '—'
  return value.toFixed(1)
}
