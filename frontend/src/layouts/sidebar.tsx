import { NavLink } from 'react-router-dom'
import { Shirt, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useAppDispatch, useAppSelector } from '@/store'
import { mobileSidebarClosed } from '@/store/ui.slice'
import { usePermissions } from '@/hooks/usePermissions'
import { NAV_SECTIONS } from './nav'

export function Brand({ collapsed }: { collapsed: boolean }) {
  return (
    <div className={cn('flex items-center gap-2.5 px-4', collapsed && 'justify-center px-0')}>
      <div className="flex size-8 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-brand-500 to-brand-700 text-white shadow-sm">
        <Shirt className="size-4.5" aria-hidden />
      </div>
      {!collapsed && (
        <div className="min-w-0">
          <p className="truncate text-sm font-bold tracking-tight text-surface-900">LaundryPro</p>
          <p className="text-[10px] font-medium uppercase tracking-widest text-surface-400">
            Admin
          </p>
        </div>
      )}
    </div>
  )
}

export function SidebarContent({ collapsed }: { collapsed: boolean }) {
  const { can } = usePermissions()

  return (
    <nav className="flex-1 space-y-5 overflow-y-auto px-3 py-4">
      {NAV_SECTIONS.map((section, i) => {
        const items = section.items.filter((item) => can(item.permission ?? ''))
        if (items.length === 0) return null
        return (
          <div key={i}>
            {section.title && !collapsed && (
              <p className="mb-1.5 px-2 text-[10px] font-semibold uppercase tracking-widest text-surface-400">
                {section.title}
              </p>
            )}
            <ul className="space-y-0.5">
              {items.map((item) => (
                <li key={item.to}>
                  <NavLink
                    to={item.to}
                    end={item.to === '/app'}
                    className={({ isActive }) =>
                      cn(
                        'group relative flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-[13px] font-medium transition-all duration-150',
                        collapsed && 'justify-center px-0',
                        isActive
                          ? 'bg-brand-50 text-brand-700'
                          : 'text-surface-600 hover:bg-surface-100 hover:text-surface-900',
                      )
                    }
                  >
                    <item.icon className="size-4 shrink-0" aria-hidden />
                    {!collapsed && <span className="truncate">{item.label}</span>}
                    {collapsed && (
                      <span className="pointer-events-none absolute left-14 z-40 hidden whitespace-nowrap rounded-md bg-surface-900 px-2 py-1 text-xs font-medium text-white shadow-pop group-hover:block">
                        {item.label}
                      </span>
                    )}
                  </NavLink>
                </li>
              ))}
            </ul>
          </div>
        )
      })}
    </nav>
  )
}

export function Sidebar({ collapsed }: { collapsed: boolean }) {
  const dispatch = useAppDispatch()
  const mobileOpen = useAppSelector((s) => s.ui.mobileSidebarOpen)

  return (
    <>
      {/* Desktop */}
      <aside
        className={cn(
          'hidden h-full flex-col border-r border-surface-200/80 bg-white transition-[width] duration-200 lg:flex',
          collapsed ? 'w-[68px]' : 'w-60',
        )}
      >
        <div className="flex h-16 items-center border-b border-surface-100">
          <Brand collapsed={collapsed} />
        </div>
        <SidebarContent collapsed={collapsed} />
      </aside>

      {/* Mobile drawer */}
      <div
        className={cn(
          'fixed inset-0 z-50 lg:hidden',
          mobileOpen ? 'pointer-events-auto' : 'pointer-events-none',
        )}
        aria-hidden={!mobileOpen}
      >
        <div
          className={cn(
            'absolute inset-0 bg-surface-950/50 backdrop-blur-[2px] transition-opacity duration-200',
            mobileOpen ? 'opacity-100' : 'opacity-0',
          )}
          onClick={() => dispatch(mobileSidebarClosed())}
        />
        <div
          className={cn(
            'absolute inset-y-0 left-0 flex w-64 flex-col bg-white shadow-dialog transition-transform duration-200',
            mobileOpen ? 'translate-x-0' : '-translate-x-full',
          )}
        >
          <div className="flex h-16 items-center justify-between border-b border-surface-100 pr-3">
            <Brand collapsed={false} />
            <button
              onClick={() => dispatch(mobileSidebarClosed())}
              aria-label="Close navigation"
              className="rounded-lg p-1.5 text-surface-400 hover:bg-surface-100 hover:text-surface-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500/40"
            >
              <X className="size-4.5" />
            </button>
          </div>
          <SidebarContent collapsed={false} />
        </div>
      </div>
    </>
  )
}
