import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface RatingView {
  id: string
  order_id: string
  customer_id: string
  score: number
  comment: string
  is_public: boolean
  created_at: string
}

export interface RatingSummary {
  total: number
  avg_score: number
  five_star: number
  four_star: number
  three_star: number
  two_star: number
  one_star: number
}

export interface CreateRatingRequest {
  order_id: string
  customer_id: string
  score: number
  comment?: string
  is_public?: boolean
}

export interface UpdateRatingRequest {
  id: string
  score?: number
  comment?: string
  is_public?: boolean
}

export function listRatings(req: PageRequest) {
  return listRequest<RatingView, RatingSummary>('/ratings/list', req)
}

export function createRating(req: CreateRatingRequest): Promise<RatingView> {
  return postRequest<CreateRatingRequest, RatingView>('/ratings', req)
}

export function updateRating(req: UpdateRatingRequest): Promise<RatingView> {
  return patchRequest<UpdateRatingRequest, RatingView>('/ratings', req)
}

export function deleteRating(id: string): Promise<void> {
  return deleteRequest('/ratings', { id })
}
