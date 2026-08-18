import { useEffect, useState } from 'react'
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { CalendarCheck, Camera, PackageCheck, Sparkles, Truck } from 'lucide-react'
import { formatCurrency } from '@/lib/format'
import { cn } from '@/lib/utils'
import { usePublicPricing } from './use-public-pricing'
import { EASE } from './reveal'

/* ---------------------------------------------------------------------------
   The signature element: a living laundry order card that cycles through
   the real order lifecycle — Booked → Picked up → Cleaning → Delivered —
   the same pipeline the LaundryPro operations console tracks.
--------------------------------------------------------------------------- */

const STAGES = [
  {
    key: 'booked',
    label: 'Booked',
    icon: CalendarCheck,
    line: 'Pickup scheduled · Tue, 4:30–6:00 PM',
  },
  {
    key: 'picked',
    label: 'Picked up',
    icon: Truck,
    line: 'Ramesh is 5 minutes away · bag confirmed ✓',
  },
  {
    key: 'cleaning',
    label: 'Cleaning',
    icon: Sparkles,
    line: 'In the wash · fragrance: Sea Breeze',
  },
  {
    key: 'delivered',
    label: 'Delivered',
    icon: PackageCheck,
    line: 'Delivered & folded · photo confirmed ✓',
  },
] as const

const CYCLE_MS = 3200

export function OrderCard() {
  const reduce = useReducedMotion()
  const [stage, setStage] = useState(0)

  // The example order value is derived from the live pricing catalog
  // (per-kg wash rate × 3.2 kg), so the card never hardcodes a price.
  const pricing = usePublicPricing()
  const catalog = pricing.data
  const perKgServices = catalog?.services.filter((s) => s.unit === 'kg') ?? []
  const washService =
    perKgServices.find((s) => s.name.toLowerCase().includes('wash & fold')) ??
    perKgServices.find((s) => s.name.toLowerCase().includes('wash')) ??
    perKgServices[0]
  const currency = catalog?.currency
  const orderValue = washService != null && currency ? formatCurrency(washService.price * 3.2, currency) : null

  useEffect(() => {
    if (reduce) return
    const id = window.setInterval(() => {
      setStage((s) => (s + 1) % STAGES.length)
    }, CYCLE_MS)
    return () => window.clearInterval(id)
  }, [reduce])

  const done = stage === STAGES.length - 1
  const progress = reduce ? 100 : (((stage + 1) / STAGES.length) * 100).toFixed(0)
  const active = STAGES[stage]
  const ActiveIcon = active.icon

  return (
    <div className="relative">
      {/* Ambient glow behind the card */}
      <div
        aria-hidden
        className="absolute -inset-10 rounded-[3rem] bg-gradient-to-br from-lp-aqua/25 via-lp-cyan/15 to-lp-indigo/25 blur-3xl"
      />

      {/* Floating chip — top left */}
      <motion.div
        className="absolute -left-8 top-10 z-10 hidden items-center gap-2 rounded-full border border-white/70 bg-white/85 px-3.5 py-2 text-xs font-semibold text-lp-ink shadow-pop backdrop-blur md:flex"
        initial={{ opacity: 0, x: -16 }}
        animate={{ opacity: 1, x: 0 }}
        transition={{ delay: 0.7, duration: 0.6, ease: EASE }}
      >
        <span className="size-1.5 rounded-full bg-lp-aqua" />
        Fragrance: Sea Breeze
      </motion.div>

      {/* Floating chip — bottom right */}
      <motion.div
        className="absolute -right-10 bottom-12 z-10 hidden items-center gap-2 rounded-full border border-white/70 bg-white/85 px-3.5 py-2 text-xs font-semibold text-lp-ink shadow-pop backdrop-blur md:flex"
        initial={{ opacity: 0, x: 16 }}
        animate={{ opacity: 1, x: 0 }}
        transition={{ delay: 0.9, duration: 0.6, ease: EASE }}
      >
        <span className="size-1.5 rounded-full bg-emerald-500" />
        On time · 98.7%
      </motion.div>

      {/* The card */}
      <motion.div
        initial={{ opacity: 0, y: 34, scale: 0.97 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ duration: 0.9, ease: EASE, delay: 0.25 }}
        className="relative w-full max-w-sm rounded-3xl border border-white/80 bg-white/90 p-5 shadow-dialog backdrop-blur-xl sm:p-6"
      >
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <p className="font-mono text-[10px] font-medium uppercase tracking-widest text-lp-slate">
              Order #LP-2048
            </p>
            <p className="mt-0.5 text-sm font-semibold text-lp-ink">
              4 items · 3.2 kg{orderValue ? ` · ${orderValue}` : ''}
            </p>
          </div>
          <AnimatePresence mode="wait">
            <motion.span
              key={done ? 'done' : 'live'}
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.9 }}
              transition={{ duration: 0.25 }}
              className={cn(
                'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[10px] font-bold uppercase tracking-wider',
                done ? 'bg-emerald-100 text-emerald-700' : 'bg-lp-mint text-lp-teal',
              )}
            >
              {!done && <span className="size-1.5 animate-pulse-dot rounded-full bg-lp-aqua" />}
              {done ? 'Delivered' : 'Live'}
            </motion.span>
          </AnimatePresence>
        </div>

        {/* Stepper */}
        <div className="mt-6">
          <ol className="relative flex justify-between">
            {/* Connector rail */}
            <div
              aria-hidden
              className="absolute left-4 right-4 top-4 h-0.5 rounded-full bg-surface-200"
            >
              <motion.div
                className="h-full rounded-full bg-gradient-to-r from-lp-teal to-lp-indigo"
                animate={{ width: reduce ? '100%' : `${(stage / (STAGES.length - 1)) * 100}%` }}
                transition={{ duration: 0.6, ease: EASE }}
              />
            </div>

            {STAGES.map((s, i) => {
              const Icon = s.icon
              const reached = i <= stage
              const current = i === stage && !done
              return (
                <li key={s.key} className="relative z-10 flex flex-col items-center gap-2">
                  <motion.span
                    animate={current ? { scale: [1, 1.18, 1] } : { scale: 1 }}
                    transition={{ duration: 1.4, repeat: current ? Infinity : 0, ease: 'easeInOut' }}
                    className={cn(
                      'flex size-8 items-center justify-center rounded-full border-2 transition-colors duration-300',
                      reached
                        ? 'border-transparent bg-gradient-to-br from-lp-teal to-lp-indigo text-white'
                        : 'border-surface-300 bg-white text-surface-400',
                    )}
                  >
                    <Icon className="size-3.5" aria-hidden />
                  </motion.span>
                  <span
                    className={cn(
                      'text-[10px] font-semibold transition-colors duration-300',
                      reached ? 'text-lp-ink' : 'text-surface-400',
                    )}
                  >
                    {s.label}
                  </span>
                </li>
              )
            })}
          </ol>
        </div>

        {/* Live status line */}
        <div className="mt-5 h-6 overflow-hidden">
          <AnimatePresence mode="wait">
            <motion.p
              key={active.key}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.3, ease: EASE }}
              className="flex items-center gap-2 text-xs font-medium text-lp-slate"
            >
              <ActiveIcon className="size-3.5 text-lp-teal" aria-hidden />
              {active.line}
            </motion.p>
          </AnimatePresence>
        </div>

        {/* ETA bar */}
        <div className="mt-3">
          <div className="flex items-center justify-between text-[10px] font-medium text-surface-400">
            <span>ETA progress</span>
            <span className="font-mono">{progress}%</span>
          </div>
          <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-surface-100">
            <motion.div
              className="h-full rounded-full bg-gradient-to-r from-lp-aqua to-lp-indigo"
              animate={{ width: `${progress}%` }}
              transition={{ duration: reduce ? 0 : 1.1, ease: EASE }}
            />
          </div>
        </div>

        {/* Footer chips */}
        <div className="mt-5 flex items-center gap-2 border-t border-surface-100 pt-4">
          <span className="flex size-7 items-center justify-center rounded-full bg-gradient-to-br from-amber-400 to-orange-500 text-[10px] font-bold text-white">
            R
          </span>
          <div className="min-w-0 flex-1">
            <p className="truncate text-xs font-semibold text-lp-ink">Ramesh · Driver</p>
            <p className="text-[10px] text-surface-400">4.9★ · 3,100 rides</p>
          </div>
          <span className="inline-flex items-center gap-1 rounded-md bg-surface-100 px-2 py-1 text-[10px] font-semibold text-lp-slate">
            <Camera className="size-3" aria-hidden />
            Confirmed
          </span>
        </div>
      </motion.div>
    </div>
  )
}
