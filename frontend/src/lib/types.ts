/**
 * Backend contract types — mirror backend/internal/pkg/response and
 * backend/internal/pkg/pagination exactly (snake_case throughout).
 */

/** Standard response envelope: { code, message, data, pagination, summary, request_id } */
export interface Envelope<T = unknown, S = unknown> {
  code: number
  message: string
  data?: T
  pagination?: Pagination
  summary?: S
  request_id?: string
}

/** Offset-based pagination summary returned on list endpoints. */
export interface Pagination {
  limit: number
  total: number
  has_more: boolean
  offset: number
}

/** Page request body sent to every list endpoint (POST /<resource>/list). */
export interface PageRequest {
  offset?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  search?: string
  filters?: Record<string, string>
}

/** Client-side pagination state managed by useListQuery. */
export interface PageState {
  limit: number
  sort_by: string
  sort_order: 'asc' | 'desc'
  search: string
  filters: Record<string, string>
}

/** Normalized result of a list call. */
export interface ListResult<T, S = unknown> {
  items: T[]
  pagination: Pagination | null
  summary: S | null
}

/** Typed error surfaced to the UI. */
export class ApiError extends Error {
  code: number
  status: number
  request_id?: string

  constructor(status: number, code: number, message: string, requestId?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.request_id = requestId
  }
}

/** Common body type for delete / id-only actions. */
export interface IDRequest {
  id: string
}
