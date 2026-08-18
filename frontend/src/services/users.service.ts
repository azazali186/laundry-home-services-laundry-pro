import { deleteRequest, listRequest, patchRequest, postRequest } from '@/lib/api'
import type { PageRequest } from '@/lib/types'
import type { UserView } from './auth.service'

export interface UserSummary {
  total_active: number
  total_inactive: number
  total_admins: number
  new_this_month: number
}

export interface CreateUserRequest {
  name: string
  email: string
  phone: string
  password: string
  role_ids: string[]
}

export interface UpdateUserRequest {
  id: string
  name?: string
  phone?: string
  is_active?: boolean
  password?: string
  role_ids?: string[]
}

export function listUsers(req: PageRequest) {
  return listRequest<UserView, UserSummary>('/users/list', req)
}

export function createUser(req: CreateUserRequest): Promise<UserView> {
  return postRequest<CreateUserRequest, UserView>('/users', req)
}

export function updateUser(req: UpdateUserRequest): Promise<UserView> {
  return patchRequest<UpdateUserRequest, UserView>('/users', req)
}

export function deleteUser(id: string): Promise<void> {
  return deleteRequest('/users', { id })
}
