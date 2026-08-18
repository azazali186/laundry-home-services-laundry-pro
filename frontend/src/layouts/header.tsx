import { useMemo } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { ChevronDown, LogOut, Menu, Settings } from 'lucide-react'
import { useAppDispatch } from '@/store'
import { mobileSidebarToggled, sidebarToggled } from '@/store/ui.slice'
import { useAuth } from '@/hooks/useAuth'
import { NAV_SECTIONS } from './nav'
import { DropdownMenu } from '@/components/ui/dropdown'

export function Header() {
  const { pathname } = useLocation()
  const navigate = useNavigate()
  const dispatch = useAppDispatch()
  const { user, logout } = useAuth()

  const section = useMemo(() => {
    for (const s of NAV_SECTIONS) {
      const hit = s.items.find((i) => i.to === pathname || (i.to !== '/' && pathname.startsWith(i.to)))
      if (hit) return hit.label
    }
    return 'Overview'
  }, [pathname])

  const initials = useMemo(() => {
    const name = user?.name ?? '?'
    return name
      .split(/\s+/)
      .slice(0, 2)
      .map((p) => p[0]?.toUpperCase())
      .join('')
  }, [user])

  const roleLabel = user?.roles?.map((r) => r.name).join(', ') || 'Staff'

  return (
    <header className="sticky top-0 z-40 flex h-16 items-center gap-3 border-b border-surface-200/80 bg-white/80 px-4 backdrop-blur-md sm:px-6">
      <button
        onClick={() => dispatch(mobileSidebarToggled())}
        aria-label="Toggle navigation"
        className="rounded-lg p-2 text-surface-500 transition-colors hover:bg-surface-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500/40 lg:hidden"
      >
        <Menu className="size-5" />
      </button>
      <button
        onClick={() => dispatch(sidebarToggled())}
        aria-label="Toggle sidebar"
        className="hidden rounded-lg p-2 text-surface-500 transition-colors hover:bg-surface-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500/40 lg:inline-flex"
      >
        <Menu className="size-5" />
      </button>

      <div className="min-w-0 flex-1">
        <h2 className="truncate text-sm font-semibold text-surface-900">{section}</h2>
        <p className="hidden text-xs text-surface-400 sm:block">Operations console</p>
      </div>

      <DropdownMenu
        trigger={
          <div className="flex items-center gap-2.5 rounded-full py-1 pl-1 pr-2 transition-colors hover:bg-surface-100">
            <span className="flex size-8 items-center justify-center rounded-full bg-gradient-to-br from-brand-500 to-brand-700 text-xs font-bold text-white">
              {initials || 'U'}
            </span>
            <span className="hidden text-left sm:block">
              <span className="block max-w-36 truncate text-xs font-semibold text-surface-900">
                {user?.name ?? 'User'}
              </span>
              <span className="block max-w-36 truncate text-[10px] text-surface-400">{roleLabel}</span>
            </span>
            <ChevronDown className="hidden size-3.5 text-surface-400 sm:block" />
          </div>
        }
        items={[
          { key: 'settings', label: 'Account settings', icon: <Settings className="size-3.5" />, onClick: () => navigate('/settings') },
          { key: 'logout', label: 'Sign out', icon: <LogOut className="size-3.5" />, danger: true, onClick: () => void logout() },
        ]}
      />
    </header>
  )
}
