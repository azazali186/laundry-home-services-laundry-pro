import { lazy, Suspense } from 'react'
import { createBrowserRouter } from 'react-router-dom'
import { AppLayout } from '@/layouts/app-layout'
import { PublicOnly, RequireAuth, RouteLoader } from '@/layouts/route-guard'

const LoginPage = lazy(() =>
  import('@/features/auth/login-page').then((m) => ({ default: m.LoginPage })),
)
const LandingPage = lazy(() =>
  import('@/features/landing/landing-page').then((m) => ({ default: m.LandingPage })),
)
const DashboardPage = lazy(() =>
  import('@/features/dashboard/dashboard-page').then((m) => ({ default: m.DashboardPage })),
)
const UsersPage = lazy(() =>
  import('@/features/users/users-page').then((m) => ({ default: m.UsersPage })),
)
const RolesPage = lazy(() =>
  import('@/features/roles/roles-page').then((m) => ({ default: m.RolesPage })),
)
const BookingsPage = lazy(() =>
  import('@/features/bookings/bookings-page').then((m) => ({ default: m.BookingsPage })),
)
const OrdersPage = lazy(() =>
  import('@/features/orders/orders-page').then((m) => ({ default: m.OrdersPage })),
)
const PickupsPage = lazy(() =>
  import('@/features/pickups/pickups-page').then((m) => ({ default: m.PickupsPage })),
)
const DeliveriesPage = lazy(() =>
  import('@/features/deliveries/deliveries-page').then((m) => ({ default: m.DeliveriesPage })),
)
const PaymentsPage = lazy(() =>
  import('@/features/payments/payments-page').then((m) => ({ default: m.PaymentsPage })),
)
const InvoicesPage = lazy(() =>
  import('@/features/invoices/invoices-page').then((m) => ({ default: m.InvoicesPage })),
)
const ServiceTypesPage = lazy(() =>
  import('@/features/service-types/service-types-page').then((m) => ({ default: m.ServiceTypesPage })),
)
const CombosPage = lazy(() =>
  import('@/features/combos/combos-page').then((m) => ({ default: m.CombosPage })),
)
const InventoryPage = lazy(() =>
  import('@/features/inventory/inventory-page').then((m) => ({ default: m.InventoryPage })),
)
const GarmentsPage = lazy(() =>
  import('@/features/garments/garments-page').then((m) => ({ default: m.GarmentsPage })),
)
const RatingsPage = lazy(() =>
  import('@/features/ratings/ratings-page').then((m) => ({ default: m.RatingsPage })),
)
const NotificationsPage = lazy(() =>
  import('@/features/notifications/notifications-page').then((m) => ({ default: m.NotificationsPage })),
)
const LoginLogsPage = lazy(() =>
  import('@/features/logs/login-logs-page').then((m) => ({ default: m.LoginLogsPage })),
)
const AuditLogsPage = lazy(() =>
  import('@/features/logs/audit-logs-page').then((m) => ({ default: m.AuditLogsPage })),
)
const SettingsPage = lazy(() =>
  import('@/features/settings/settings-page').then((m) => ({ default: m.SettingsPage })),
)
const NotFoundPage = lazy(() =>
  import('@/features/not-found/not-found-page').then((m) => ({ default: m.NotFoundPage })),
)

function withSuspense(node: React.ReactNode) {
  return <Suspense fallback={<RouteLoader />}>{node}</Suspense>
}

export const router = createBrowserRouter([
  {
    path: '/',
    element: withSuspense(<LandingPage />),
  },
  {
    path: '/login',
    element: withSuspense(
      <PublicOnly>
        <LoginPage />
      </PublicOnly>,
    ),
  },
  {
    path: '/app',
    element: (
      <RequireAuth>
        <AppLayout />
      </RequireAuth>
    ),
    children: [
      { index: true, element: withSuspense(<DashboardPage />) },
      { path: 'bookings', element: withSuspense(<BookingsPage />) },
      { path: 'pickups', element: withSuspense(<PickupsPage />) },
      { path: 'orders', element: withSuspense(<OrdersPage />) },
      { path: 'deliveries', element: withSuspense(<DeliveriesPage />) },
      { path: 'garments', element: withSuspense(<GarmentsPage />) },
      { path: 'service-types', element: withSuspense(<ServiceTypesPage />) },
      { path: 'combos', element: withSuspense(<CombosPage />) },
      { path: 'payments', element: withSuspense(<PaymentsPage />) },
      { path: 'invoices', element: withSuspense(<InvoicesPage />) },
      { path: 'inventory', element: withSuspense(<InventoryPage />) },
      { path: 'ratings', element: withSuspense(<RatingsPage />) },
      { path: 'notifications', element: withSuspense(<NotificationsPage />) },
      { path: 'users', element: withSuspense(<UsersPage />) },
      { path: 'roles', element: withSuspense(<RolesPage />) },
      { path: 'logs/login', element: withSuspense(<LoginLogsPage />) },
      { path: 'logs/audit', element: withSuspense(<AuditLogsPage />) },
      { path: 'settings', element: withSuspense(<SettingsPage />) },
      { path: '*', element: withSuspense(<NotFoundPage />) },
    ],
  },
  // Root-level catch-all so unknown top-level URLs never render a blank page.
  {
    path: '*',
    element: withSuspense(<NotFoundPage />),
  },
])
