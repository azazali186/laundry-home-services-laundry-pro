import {
  Bell,
  Boxes,
  CalendarCheck,
  CreditCard,
  FileText,
  Layers,
  LayoutDashboard,
  LogIn,
  Package,
  ScrollText,
  Shield,
  Shirt,
  Sparkles,
  Star,
  Truck,
  Users,
  type LucideIcon,
} from 'lucide-react'

export interface NavItem {
  to: string
  label: string
  icon: LucideIcon
  /** Backend permission route that gates this item (best-effort UI gating). */
  permission?: string
}

export interface NavSection {
  title?: string
  items: NavItem[]
}

export const NAV_SECTIONS: NavSection[] = [
  {
    items: [{ to: '/app', label: 'Overview', icon: LayoutDashboard, permission: 'POST /analytics/dashboard' }],
  },
  {
    title: 'Operations',
    items: [
      { to: '/app/bookings', label: 'Bookings', icon: CalendarCheck, permission: 'POST /bookings/list' },
      { to: '/app/pickups', label: 'Pickups', icon: Truck, permission: 'POST /pickups/list' },
      { to: '/app/orders', label: 'Orders', icon: Package, permission: 'POST /orders/list' },
      { to: '/app/deliveries', label: 'Deliveries', icon: Truck, permission: 'POST /deliveries/list' },
      { to: '/app/garments', label: 'Garments', icon: Shirt, permission: 'POST /garments/list' },
    ],
  },
  {
    title: 'Commerce',
    items: [
      { to: '/app/service-types', label: 'Service Types', icon: Sparkles, permission: 'POST /service-types/list' },
      { to: '/app/combos', label: 'Combos', icon: Layers, permission: 'POST /combos/list' },
      { to: '/app/payments', label: 'Payments', icon: CreditCard, permission: 'POST /payments/list' },
      { to: '/app/invoices', label: 'Invoices', icon: FileText, permission: 'POST /invoices/list' },
      { to: '/app/inventory', label: 'Inventory', icon: Boxes, permission: 'POST /inventory/list' },
    ],
  },
  {
    title: 'Engagement',
    items: [
      { to: '/app/ratings', label: 'Ratings', icon: Star, permission: 'POST /ratings/list' },
      { to: '/app/notifications', label: 'Notifications', icon: Bell, permission: 'POST /notifications/list' },
    ],
  },
  {
    title: 'Access',
    items: [
      { to: '/app/users', label: 'Users', icon: Users, permission: 'POST /users/list' },
      { to: '/app/roles', label: 'Roles & Permissions', icon: Shield, permission: 'POST /roles/list' },
    ],
  },
  {
    title: 'Audit',
    items: [
      { to: '/app/logs/login', label: 'Login Logs', icon: LogIn, permission: 'POST /login-logs/list' },
      { to: '/app/logs/audit', label: 'Audit Trail', icon: ScrollText, permission: 'POST /audit-logs/list' },
    ],
  },
]
