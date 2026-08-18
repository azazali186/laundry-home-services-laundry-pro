import { ChevronLeft, ChevronRight } from 'lucide-react'
import { showingSummary } from '@/lib/format'
import { Button } from '@/components/ui/button'
import type { ListQueryResult } from '@/hooks/useListQuery'

interface PaginationBarProps<T> {
  list: ListQueryResult<T, unknown>
}

export function PaginationBar<T>({ list }: PaginationBarProps<T>) {
  const { total, pageIndex, page, hasMore, isFetching, nextPage, prevPage } = list

  return (
    <div className="flex flex-col gap-2 border-t border-surface-100 px-5 py-3.5 sm:flex-row sm:items-center sm:justify-between">
      <p className="text-xs font-medium text-surface-500">
        {showingSummary(pageIndex, page.limit, total)}
        {isFetching && <span className="ml-2 text-surface-400">refreshing…</span>}
      </p>
      <div className="flex items-center gap-1.5">
        <Button
          variant="outline"
          size="sm"
          disabled={pageIndex === 0}
          onClick={prevPage}
          aria-label="Previous page"
        >
          <ChevronLeft className="size-3.5" />
          <span className="hidden sm:inline">Previous</span>
        </Button>
        <span className="px-2 text-xs font-semibold text-surface-600">
          Page {pageIndex + 1}
        </span>
        <Button
          variant="outline"
          size="sm"
          disabled={!hasMore}
          onClick={nextPage}
          aria-label="Next page"
        >
          <span className="hidden sm:inline">Next</span>
          <ChevronRight className="size-3.5" />
        </Button>
      </div>
    </div>
  )
}
