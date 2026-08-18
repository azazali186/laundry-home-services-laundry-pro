import { Outlet } from 'react-router-dom'
import { useAppSelector } from '@/store'
import { Header } from './header'
import { Sidebar } from './sidebar'

export function AppLayout() {
  const collapsed = useAppSelector((s) => s.ui.sidebarCollapsed)

  return (
    <div className="flex h-dvh overflow-hidden">
      <Sidebar collapsed={collapsed} />
      <div className="flex min-w-0 flex-1 flex-col">
        <Header />
        <main className="flex-1 overflow-y-auto">
          <div className="mx-auto w-full max-w-7xl animate-fade-in p-4 sm:p-6 lg:p-8">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}
