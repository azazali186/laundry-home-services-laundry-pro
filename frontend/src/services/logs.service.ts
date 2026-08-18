import { listRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'

export interface LoginLogView {
  id: string
  email: string
  ip: string
  user_agent: string
  status: string
  reason: string
  created_at: string
}

export interface AuditLogView {
  id: string
  user_id: string
  entity: string
  entity_id: string
  action: string
  method: string
  path: string
  ip: string
  before: unknown
  after: unknown
  created_at: string
}

export function listLoginLogs(req: PageRequest) {
  return listRequest<LoginLogView, null>('/login-logs/list', req)
}

export function listAuditLogs(req: PageRequest) {
  return listRequest<AuditLogView, null>('/audit-logs/list', req)
}
