import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'
import type { PermissionView, RoleView } from './auth.service'

export interface RoleSummary {
  total_system: number
  total_custom: number
  total_users: number
}

export interface CreateRoleRequest {
  name: string
  code: string
  description: string
  permission_ids: string[]
}

export interface UpdateRoleRequest {
  id: string
  name?: string
  code?: string
  description?: string
  permission_ids?: string[]
}

export interface AssignPermissionsRequest {
  role_id: string
  permission_ids: string[]
}

export function listRoles(req: PageRequest) {
  return listRequest<RoleView, RoleSummary>('/roles/list', req)
}

export function listPermissions(req: PageRequest) {
  return listRequest<PermissionView, null>('/permissions/list', req)
}

export function createRole(req: CreateRoleRequest): Promise<RoleView> {
  return postRequest<CreateRoleRequest, RoleView>('/roles', req)
}

export function updateRole(req: UpdateRoleRequest): Promise<RoleView> {
  return patchRequest<UpdateRoleRequest, RoleView>('/roles', req)
}

export function deleteRole(id: string): Promise<void> {
  return deleteRequest('/roles', { id })
}

export function assignPermissions(req: AssignPermissionsRequest): Promise<RoleView> {
  return postRequest<AssignPermissionsRequest, RoleView>('/roles/assign-permissions', req)
}
