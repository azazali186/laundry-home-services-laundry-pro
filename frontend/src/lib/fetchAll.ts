import type { ListResult, PageRequest } from './types'

/** Fetch every page of an offset-paginated list endpoint (up to `maxPages`). */
export interface FetchAllOptions {
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  maxPages?: number
}

export async function fetchAllPages<T, S = unknown>(
  fetcher: (req: PageRequest) => Promise<ListResult<T, S>>,
  { limit = 100, sort_by = 'name', sort_order = 'asc', maxPages = 5 }: FetchAllOptions = {},
): Promise<T[]> {
  const out: T[] = []
  for (let i = 0; i < maxPages; i += 1) {
    const offset = i * limit
    const res = await fetcher({ offset, limit, sort_by, sort_order })
    out.push(...res.items)
    if (!res.pagination?.has_more) break
  }
  return out
}
