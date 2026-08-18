import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { Spinner } from '@/components/ui/feedback'

/** Redirects unauthenticated users to /login, preserving destination. */
export function RequireAuth({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, hydrated } = useAuth()
  const location = useLocation()

  if (!isAuthenticated && !hydrated) {
    return <Navigate to="/login" state={{ from: location.pathname }} replace />
  }
  return <>{children}</>
}

/** Redirects authenticated users away from public pages (e.g. login). */
export function PublicOnly({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth()
  if (isAuthenticated) return <Navigate to="/app" replace />
  return <>{children}</>
}

/** Full-screen loader used during route-level suspense fallbacks. */
export function RouteLoader() {
  return (
    <div className="flex h-dvh items-center justify-center">
      <Spinner className="size-7" />
    </div>
  )
}
