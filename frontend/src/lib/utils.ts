import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export { humanize } from './format'

/** Merge Tailwind classes with conflict resolution. */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs))
}

/** Sort an array of {value,label} options alphabetically by label. */
export function sortOptions<T extends { label: string }>(opts: T[]): T[] {
  return [...opts].sort((a, b) => a.label.localeCompare(b.label))
}

/** Guard: value is a non-empty string. */
export function isNonEmpty(v: unknown): v is string {
  return typeof v === 'string' && v.trim().length > 0
}

/** Build a filter value with the backend operator prefix (default eq). */
export function eq(value: string): string {
  return `eq:${value}`
}

export function like(value: string): string {
  return `like:${value}`
}

/** Date range filter keys per the backend's `_from` / `_to` convention. */
export interface DateRange {
  from?: string
  to?: string
}

export function dateRangeFilters(range: DateRange, column = 'created_at'): Record<string, string> {
  const filters: Record<string, string> = {}
  if (range.from) filters[`${column}_from`] = range.from
  if (range.to) filters[`${column}_to`] = range.to
  return filters
}
