import { createContext, useCallback, useContext, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { CheckCircle2, Info, TriangleAlert, X, XCircle } from 'lucide-react'
import { cn } from '@/lib/utils'

type ToastVariant = 'success' | 'error' | 'info' | 'warning'

interface ToastItem {
  id: number
  title: string
  description?: string
  variant: ToastVariant
}

interface ToastInput {
  title: string
  description?: string
  variant?: ToastVariant
  duration?: number
}

interface ToastContextValue {
  toast: (input: ToastInput) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

const variantStyles: Record<ToastVariant, { icon: React.ReactNode; ring: string; iconBg: string }> = {
  success: {
    icon: <CheckCircle2 className="size-4.5 text-emerald-600" />,
    ring: 'ring-emerald-200',
    iconBg: 'bg-emerald-50',
  },
  error: {
    icon: <XCircle className="size-4.5 text-red-600" />,
    ring: 'ring-red-200',
    iconBg: 'bg-red-50',
  },
  info: {
    icon: <Info className="size-4.5 text-blue-600" />,
    ring: 'ring-blue-200',
    iconBg: 'bg-blue-50',
  },
  warning: {
    icon: <TriangleAlert className="size-4.5 text-amber-600" />,
    ring: 'ring-amber-200',
    iconBg: 'bg-amber-50',
  },
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([])
  const nextId = useRef(1)

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const toast = useCallback(
    ({ title, description, variant = 'info', duration = 4200 }: ToastInput) => {
      const id = nextId.current++
      setToasts((prev) => [...prev.slice(-4), { id, title, description, variant }])
      window.setTimeout(() => dismiss(id), duration)
    },
    [dismiss],
  )

  const value = useMemo(() => ({ toast }), [toast])

  return (
    <ToastContext.Provider value={value}>
      {children}
      {createPortal(
        <div className="pointer-events-none fixed inset-x-0 top-0 z-[60] flex flex-col items-center gap-2 p-4 sm:items-end sm:p-6">
          {toasts.map((t) => {
            const s = variantStyles[t.variant]
            return (
              <div
                key={t.id}
                role={t.variant === 'error' || t.variant === 'warning' ? 'alert' : 'status'}
                className={cn(
                  'pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-xl bg-white p-3.5 shadow-pop ring-1 animate-toast-in',
                  s.ring,
                )}
              >
                <span className={cn('mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-lg', s.iconBg)}>
                  {s.icon}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-semibold text-surface-900">{t.title}</p>
                  {t.description && <p className="mt-0.5 text-xs leading-relaxed text-surface-500">{t.description}</p>}
                </div>
                <button
                  onClick={() => dismiss(t.id)}
                  aria-label="Dismiss"
                  className="rounded-md p-1 text-surface-400 transition-colors hover:bg-surface-100 hover:text-surface-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500/40"
                >
                  <X className="size-3.5" />
                </button>
              </div>
            )
          })}
        </div>,
        document.body,
      )}
    </ToastContext.Provider>
  )
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within <ToastProvider>')
  return ctx
}
