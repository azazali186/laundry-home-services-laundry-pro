import { useCallback, useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { PageRequest, PageState, ListResult } from '@/lib/types'
import { useDebounce } from './useDebounce'

export interface ListQueryOptions<T, S = unknown> {
  /** Base query key (excluding pagination state). */
  queryKey: readonly unknown[]
  /** List fetcher returning normalized items + pagination + summary. */
  fetcher: (req: PageRequest) => Promise<ListResult<T, S>>
  defaultLimit?: number
  defaultSortBy?: string
  defaultSortOrder?: 'asc' | 'desc'
  /** Re-run the query when any of these change (e.g. a selected filter id). */
  extra?: unknown
}

export interface ListQueryResult<T, S = unknown> {
  items: T[]
  summary: S | null
  total: number
  hasMore: boolean
  isLoading: boolean
  isFetching: boolean
  isError: boolean
  error: unknown
  refetch: () => void
  page: PageState
  pageIndex: number
  setPageIndex: (i: number) => void
  setSearch: (v: string) => void
  setSort: (by: string) => void
  toggleSort: (by: string) => void
  setFilter: (key: string, value: string | null) => void
  setDateRange: (from?: string, to?: string, column?: string) => void
  setLimit: (n: number) => void
  reset: () => void
  nextPage: () => void
  prevPage: () => void
}

/**
 * Offset-paginated list with server-side search, sorting, filters.
 * Each page is fetched by computing offset = pageIndex * limit.
 */
export function useListQuery<T, S = unknown>({
  queryKey,
  fetcher,
  defaultLimit = 20,
  defaultSortBy = 'created_at',
  defaultSortOrder = 'desc',
  extra,
}: ListQueryOptions<T, S>): ListQueryResult<T, S> {
  const [pageIndex, setPageIndex] = useState(0)
  const [limit, setLimitState] = useState(defaultLimit)
  const [sortBy, setSortBy] = useState(defaultSortBy)
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>(defaultSortOrder)
  const [search, setSearchState] = useState('')
  const [filters, setFilters] = useState<Record<string, string>>({})

  const debouncedSearch = useDebounce(search, 400)

  // Any parameter change resets navigation back to page zero.
  useEffect(() => {
    setPageIndex(0)
  }, [debouncedSearch, sortBy, sortOrder, limit, filters])

  const request = useMemo<PageRequest>(
    () => ({
      offset: pageIndex * limit,
      limit,
      sort_by: sortBy,
      sort_order: sortOrder,
      search: debouncedSearch || undefined,
      filters: Object.keys(filters).length ? filters : undefined,
    }),
    [pageIndex, limit, sortBy, sortOrder, debouncedSearch, filters],
  )

  const query = useQuery({
    queryKey: [...queryKey, { ...request, extra }],
    queryFn: () => fetcher(request),
    placeholderData: (prev) => prev,
  })

  const items = query.data?.items ?? []
  const summary = (query.data?.summary ?? null) as S | null
  const total = query.data?.pagination?.total ?? 0
  const hasMore = query.data?.pagination?.has_more ?? false

  const setSearch = useCallback((v: string) => setSearchState(v), [])
  const setLimit = useCallback((n: number) => {
    setLimitState(n)
    setPageIndex(0) // reset to first page on page-size change
  }, [])

  const toggleSort = useCallback(
    (by: string) => {
      if (by === sortBy) {
        setSortOrder((o) => (o === 'asc' ? 'desc' : 'asc'))
      } else {
        setSortBy(by)
        setSortOrder('desc')
      }
    },
    [sortBy],
  )

  const setSort = useCallback((by: string) => {
    setSortBy(by)
    setSortOrder('desc')
  }, [])

  const setFilter = useCallback((key: string, value: string | null) => {
    setFilters((prev) => {
      const next = { ...prev }
      if (value === null || value === '') delete next[key]
      else next[key] = value
      return next
    })
  }, [])

  const setDateRange = useCallback(
    (from?: string, to?: string, column = 'created_at') => {
      setFilters((prev) => {
        const next = { ...prev }
        const fromKey = `${column}_from`
        const toKey = `${column}_to`
        if (from) next[fromKey] = from
        else delete next[fromKey]
        if (to) next[toKey] = to
        else delete next[toKey]
        return next
      })
    },
    [],
  )

  const reset = useCallback(() => {
    setSearchState('')
    setFilters({})
    setSortBy(defaultSortBy)
    setSortOrder(defaultSortOrder)
    setLimitState(defaultLimit)
    setPageIndex(0)
  }, [defaultLimit, defaultSortBy, defaultSortOrder])

  const nextPage = useCallback(() => {
    setPageIndex((i) => (hasMore ? i + 1 : i))
  }, [hasMore])

  const prevPage = useCallback(() => {
    setPageIndex((i) => Math.max(0, i - 1))
  }, [])

  const page: PageState = useMemo(
    () => ({ limit, sort_by: sortBy, sort_order: sortOrder, search, filters }),
    [limit, sortBy, sortOrder, search, filters],
  )

  return {
    items,
    summary,
    total,
    hasMore,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
    page,
    pageIndex,
    setPageIndex,
    setSearch,
    setSort,
    toggleSort,
    setFilter,
    setDateRange,
    setLimit,
    reset,
    nextPage,
    prevPage,
  }
}
