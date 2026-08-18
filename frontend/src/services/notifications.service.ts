import { listRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface NotificationView {
  id: string
  user_id: string
  title: string
  body: string
  type: string
  channel: string
  status: string
  sent_at?: string | null
  read_at?: string | null
  created_at: string
}

export interface NotificationSummary {
  total: number
  pending: number
  sent: number
  failed: number
  unread: number
}

export interface CreateNotificationRequest {
  user_id: string
  title: string
  body: string
  type: string
  channel: string
}

export function listNotifications(req: PageRequest) {
  return listRequest<NotificationView, NotificationSummary>('/notifications/list', req)
}

export function createNotification(req: CreateNotificationRequest): Promise<NotificationView> {
  return postRequest<CreateNotificationRequest, NotificationView>('/notifications', req)
}

export function markNotificationRead(id: string): Promise<NotificationView> {
  return postRequest<{ id: string }, NotificationView>('/notifications/mark-read', { id })
}

export const NOTIFICATION_TYPES = ['info', 'success', 'warning', 'error'] as const
export const NOTIFICATION_CHANNELS = ['in_app', 'email', 'sms'] as const
