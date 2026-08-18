import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import { errorMessage } from '@/lib/api'
import type { ListQueryResult } from '@/hooks/useListQuery'
import { DropdownMenu } from '@/components/ui/dropdown'
import { EmptyState, ErrorState, SkeletonRows } from '@/components/ui/feedback'
import { DataToolbar } from './toolbar'
import { PaginationBar } from './pagination-bar'
import type { Column, RowAction } from './columns'

export interface DataTableProps<T> {
  columns: Column<T>[]
  list: ListQueryResult<T, unknown>
  rowKey: (row: T) => string | number
  searchPlaceholder?: string
  enableSearch?: boolean
  enableDateRange?: boolean
  dateRangeColumn?: string
  toolbarExtra?: React.ReactNode
  rowActions?: (row: T) => RowAction<T>[]
  onRowClick?: (row: T) => void
  emptyTitle?: string
  emptyMessage?: string
  /** Optional summary chips rendered above the table. */
  summary?: React.ReactNode
}

export function DataTable<T>({
  columns,
  list,
  rowKey,
  searchPlaceholder,
  enableSearch,
  enableDateRange,
  dateRangeColumn,
  toolbarExtra,
  rowActions,
  onRowClick,
  emptyTitle = 'No records found',
  emptyMessage,
  summary,
}: DataTableProps<T>) {
  const { items, isLoading, isError, error, page, toggleSort, refetch } = list
  const skeletonCols = Math.min(columns.length + 1, 6)

  return (
    <div className="overflow-hidden rounded-2xl border border-surface-200/80 bg-white shadow-card">
      <DataToolbar
        columns={columns}
        list={list}
        searchPlaceholder={searchPlaceholder}
        enableSearch={enableSearch}
        enableDateRange={enableDateRange}
        dateRangeColumn={dateRangeColumn}
        toolbarExtra={toolbarExtra}
      />
      {summary}

      <div className="overflow-x-auto">
        <table className="w-full min-w-[640px] border-collapse text-left">
          <thead>
            <tr className="border-b border-surface-100">
              {columns.map((col) => {
                const active = page.sort_by === col.key
                const dir = page.sort_order
                return (
                  <th
                    key={col.key}
                    className={cn(
                      'whitespace-nowrap px-5 py-2.5 text-xs font-semibold text-surface-500',
                      col.align === 'right' && 'text-right',
                      col.hideOnMobile && 'hidden md:table-cell',
                      col.headerClassName,
                    )}
                  >
                    {col.sortable ? (
                      <button
                        onClick={() => toggleSort(col.key)}
                        className={cn(
                          'inline-flex items-center gap-1 transition-colors hover:text-surface-800',
                          active && 'text-brand-600',
                        )}
                      >
                        {col.header}
                        {active ? (
                          dir === 'asc' ? (
                            <ArrowUp className="size-3" />
                          ) : (
                            <ArrowDown className="size-3" />
                          )
                        ) : (
                          <ArrowUpDown className="size-3 opacity-40" />
                        )}
                      </button>
                    ) : (
                      col.header
                    )}
                  </th>
                )
              })}
              {rowActions && <th className="w-10 px-2 py-2.5" aria-label="Actions" />}
            </tr>
          </thead>
          <tbody className="divide-y divide-surface-100">
            {isLoading ? (
              <tr>
                <td colSpan={columns.length + (rowActions ? 1 : 0)} className="p-0">
                  <SkeletonRows rows={7} cols={skeletonCols} />
                </td>
              </tr>
            ) : isError ? (
              <tr>
                <td colSpan={columns.length + (rowActions ? 1 : 0)}>
                  <ErrorState message="Couldn't load this data" detail={errorMessage(error)} onRetry={refetch} />
                </td>
              </tr>
            ) : items.length === 0 ? (
              <tr>
                <td colSpan={columns.length + (rowActions ? 1 : 0)}>
                  <EmptyState title={emptyTitle} message={emptyMessage} />
                </td>
              </tr>
            ) : (
              items.map((row) => (
                <tr
                  key={String(rowKey(row))}
                  onClick={onRowClick ? () => onRowClick(row) : undefined}
                  className={cn(
                    'group transition-colors duration-150 hover:bg-surface-50/80',
                    onRowClick && 'cursor-pointer',
                  )}
                >
                  {columns.map((col) => (
                    <td
                      key={col.key}
                      className={cn(
                        'px-5 py-3 text-sm text-surface-700',
                        col.align === 'right' ? 'text-right' : 'text-left',
                        col.hideOnMobile && 'hidden md:table-cell',
                        col.className,
                      )}
                    >
                      {col.render ? col.render(row) : (row as Record<string, unknown>)[col.key] != null ? String((row as Record<string, unknown>)[col.key]) : '—'}
                    </td>
                  ))}
                  {rowActions && (
                    <td className="px-2 py-3 text-right">
                      <div className="transition-opacity duration-150 sm:opacity-0 sm:group-hover:opacity-100 sm:group-focus-within:opacity-100">
                        <DropdownMenu
                          items={rowActions(row).map((a) => ({
                            ...a,
                            onClick: () => a.onClick(row),
                          }))}
                        />
                      </div>
                    </td>
                  )}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <PaginationBar list={list} />
    </div>
  )
}

export type { Column, RowAction } from './columns'
