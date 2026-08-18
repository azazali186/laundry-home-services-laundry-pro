import { postRequest } from '@/lib/api'

/** Role view — code drives UI-level role gating. */
export interface RoleView {
  id: string
  name: string
  code: string
  description: string
  is_system: boolean
  permissions?: PermissionView[]
}

export interface PermissionView {
  id: string
  name: string
  route: string
  path: string
  method: string
  service: string
  guard: string
}

/** Public user representation. */
export interface UserView {
  id: string
  name: string
  email: string
  phone: string
  is_active: boolean
  last_login_at?: string | null
  created_at: string
  updated_at: string
  roles?: RoleView[]
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  access_token: string
  expires_in: number
  token_type: string
  user: UserView
}

export interface ChangePasswordRequest {
  current_password: string
  new_password: string
}

export async function login(req: LoginRequest): Promise<LoginResponse> {
  return postRequest<LoginRequest, LoginResponse>('/auth/login', req)
}

export async function fetchMe(): Promise<UserView> {
  return postRequest<object, UserView>('/auth/me', {})
}

export async function logout(): Promise<void> {
  await postRequest<object>('/auth/logout', {})
}

export async function changePassword(req: ChangePasswordRequest): Promise<void> {
  await postRequest<ChangePasswordRequest>('/auth/change-password', req)
}

/** Role codes treated as global administrators for UI gating. */
export const ADMIN_ROLES = new Set(['admin', 'super_admin', 'super-admin'])
