export interface Column<T> {
  /** Field key — used for server-side sorting and filter selects. */
  key: string
  header: string
  sortable?: boolean
  /** Enum-like filter options; renders a select in the toolbar. */
  filterOptions?: Array<{ value: string; label: string }>
  align?: 'left' | 'right' | 'center'
  render?: (row: T) => React.ReactNode
  /** Hide the column below md breakpoint. */
  hideOnMobile?: boolean
  className?: string
  headerClassName?: string
}

export interface RowAction<T> {
  key: string
  label: string
  icon?: React.ReactNode
  danger?: boolean
  disabled?: boolean
  onClick: (row: T) => void
}
