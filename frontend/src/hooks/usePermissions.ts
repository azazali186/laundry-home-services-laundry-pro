import { useMemo } from 'react'
import { useAppSelector } from '@/store'
import { ADMIN_ROLES } from '@/services/auth.service'

export interface PermissionSet {
  /** Raw permission routes ("POST /bookings") the user holds. */
  routes: Set<string>
  /** Role codes the user holds. */
  roles: Set<string>
  /** True when the user holds a global admin role. */
  isAdmin: boolean
  /** Gate by permission route; admins (and empty permission sets) pass. */
  can: (route: string) => boolean
  /** Gate by role code. */
  hasRole: (code: string) => boolean
}

/**
 * UI-level RBAC. The backend is the real enforcement point; this only
 * hides/disables actions. When the API returns no permissions for a role
 * (it currently returns role codes only), role-level gates still apply.
 */
export function usePermissions(): PermissionSet {
  const user = useAppSelector((s) => s.auth.user)

  return useMemo<PermissionSet>(() => {
    const roles = new Set<string>()
    const routes = new Set<string>()
    for (const role of user?.roles ?? []) {
      if (role.code) roles.add(role.code.toLowerCase())
      for (const p of role.permissions ?? []) {
        if (p.route) routes.add(p.route.toUpperCase())
      }
    }
    const isAdmin = [...roles].some((r) => ADMIN_ROLES.has(r))

    return {
      routes,
      roles,
      isAdmin,
      can: (route: string) => isAdmin || routes.size === 0 || routes.has(route.toUpperCase()),
      hasRole: (code: string) => isAdmin || roles.has(code.toLowerCase()),
    }
  }, [user])
}
