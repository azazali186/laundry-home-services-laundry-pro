import { CalendarRange, Search, X } from 'lucide-react'
import { humanize } from '@/lib/utils'
import { Input, Select } from '@/components/ui/form'
import { Button } from '@/components/ui/button'
import type { Column } from './columns'
import type { ListQueryResult } from '@/hooks/useListQuery'

interface ToolbarProps<T> {
  columns: Column<T>[]
  list: ListQueryResult<T, unknown>
  searchPlaceholder?: string
  enableSearch?: boolean
  enableDateRange?: boolean
  dateRangeColumn?: string
  toolbarExtra?: React.ReactNode
}

export function DataToolbar<T>({
  columns,
  list,
  searchPlaceholder = 'Search…',
  enableSearch = true,
  enableDateRange = true,
  dateRangeColumn = 'created_at',
  toolbarExtra,
}: ToolbarProps<T>) {
  const { page, setSearch, setFilter, setDateRange, setLimit } = list
  const filterColumns = columns.filter((c) => c.filterOptions)

  const hasActiveFilters =
    Object.keys(page.filters).length > 0 || page.search !== '' || page.limit !== 20

  return (
    <div className="flex flex-col gap-3 border-b border-surface-100 px-5 py-3.5">
      <div className="flex flex-wrap items-center gap-2">
        {enableSearch && (
          <div className="relative min-w-52 flex-1 sm:max-w-xs">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-400" />
            <Input
              value={page.search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={searchPlaceholder}
              className="pl-9"
              aria-label="Search"
            />
            {page.search && (
              <button
                onClick={() => setSearch('')}
                aria-label="Clear search"
                className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-surface-400 hover:text-surface-600"
              >
                <X className="size-3.5" />
              </button>
            )}
          </div>
        )}

        {filterColumns.map((col) => (
          <Select
            key={col.key}
            aria-label={`Filter by ${col.header}`}
            value={page.filters[col.key]?.replace(/^eq:/, '') ?? ''}
            onChange={(e) => {
              const v = e.target.value
              setFilter(col.key, v ? `eq:${v}` : null)
            }}
            className="h-9.5 w-40"
          >
            <option value="">{humanize(col.header)}: All</option>
            {col.filterOptions?.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </Select>
        ))}

        <div className="flex-1" />

        <div className="flex items-center gap-2">
          <Select
            aria-label="Page size"
            value={String(page.limit)}
            onChange={(e) => setLimit(Number(e.target.value))}
            className="h-9.5 w-auto pr-8"
          >
            {[10, 20, 50, 100].map((n) => (
              <option key={n} value={n}>
                {n} / page
              </option>
            ))}
          </Select>
          {toolbarExtra}
        </div>
      </div>

      {enableDateRange && (
        <div className="flex flex-wrap items-center gap-2">
          <span className="inline-flex items-center gap-1.5 text-xs font-medium text-surface-500">
            <CalendarRange className="size-3.5" /> Created
          </span>
          <input
            type="date"
            aria-label="From date"
            value={page.filters[`${dateRangeColumn}_from`] ?? ''}
            onChange={(e) => setDateRange(e.target.value || undefined, page.filters[`${dateRangeColumn}_to`], dateRangeColumn)}
            className="h-9.5 rounded-lg border border-surface-300 bg-white px-2.5 text-sm text-surface-700 shadow-sm transition-colors focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/25"
          />
          <span className="text-xs text-surface-400">to</span>
          <input
            type="date"
            aria-label="To date"
            value={page.filters[`${dateRangeColumn}_to`] ?? ''}
            onChange={(e) => setDateRange(page.filters[`${dateRangeColumn}_from`], e.target.value || undefined, dateRangeColumn)}
            className="h-9.5 rounded-lg border border-surface-300 bg-white px-2.5 text-sm text-surface-700 shadow-sm transition-colors focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/25"
          />
          {hasActiveFilters && (
            <Button variant="ghost" size="sm" onClick={() => list.reset()}>
              <X className="size-3.5" /> Reset
            </Button>
          )}
        </div>
      )}
    </div>
  )
}
