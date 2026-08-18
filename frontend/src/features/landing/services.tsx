import { Footprints, Home, RefreshCw, Shirt, Sparkles, Sun, type LucideIcon } from 'lucide-react'
import type { PricingServiceView } from '@/services/pricing.service'
import { formatLandingPrice, unitSuffix } from './price-format'
import { SectionHeading, Stagger, StaggerItem } from './reveal'

const ICON_BY_CODE: Record<string, LucideIcon> = {
  WASH: Shirt,
  DRY: Sparkles,
  IRON: Sun,
  LINEN: Home,
  SHOES: Footprints,
}

const TINTS = [
  'from-teal-500 to-cyan-500',
  'from-indigo-500 to-violet-500',
  'from-amber-500 to-orange-500',
  'from-emerald-500 to-teal-500',
  'from-sky-500 to-cyan-500',
  'from-fuchsia-500 to-indigo-500',
]

interface ServicesProps {
  services: PricingServiceView[] | undefined
  isLoading: boolean
  isError: boolean
  onRetry: () => void
}

export function Services({ services, isLoading, isError, onRetry }: ServicesProps) {
  const rows = services ?? []
  const showSkeleton = isLoading && !isError && rows.length === 0

  return (
    <section id="services" className="relative bg-white py-20 sm:py-28">
      <div aria-hidden className="pointer-events-none absolute inset-0">
        <div className="absolute inset-0 bg-[radial-gradient(50%_40%_at_85%_10%,rgb(215_240_234/0.5),transparent_70%)]" />
      </div>

      <div className="relative mx-auto max-w-6xl px-4 sm:px-6">
        <SectionHeading
          eyebrow="Services"
          title={
            <>
              Everything a wardrobe needs,{' '}
              <em className="bg-gradient-to-r from-lp-teal to-lp-indigo bg-clip-text font-medium italic text-transparent">
                handled.
              </em>
            </>
          }
          sub="From everyday washes to heirloom silk — each service has its own line, its own finish, and its own care standard."
          center
        />

        {showSkeleton ? (
          <div className="mt-14 grid gap-5 sm:grid-cols-2 lg:grid-cols-3" aria-busy="true">
            {Array.from({ length: 6 }).map((_, i) => (
              <div
                key={i}
                className="h-52 animate-pulse rounded-3xl border border-surface-200/70 bg-surface-100/60"
              />
            ))}
          </div>
        ) : isError || rows.length === 0 ? (
          <div className="mt-14 flex flex-col items-center rounded-3xl border border-dashed border-surface-300/80 bg-white/60 px-6 py-14 text-center">
            <p className="text-sm font-semibold text-lp-ink">Services are temporarily unavailable</p>
            <p className="mt-1.5 max-w-md text-sm text-lp-slate">
              We couldn’t load the live service catalogue. Please try again — your laundry is
              already on its way.
            </p>
            <button
              onClick={onRetry}
              className="mt-6 inline-flex items-center gap-2 rounded-xl bg-lp-teal px-5 py-2.5 text-sm font-semibold text-white shadow-sm transition-all duration-200 hover:bg-lp-teal-dark active:scale-[0.98]"
            >
              <RefreshCw className="size-4" aria-hidden /> Try again
            </button>
          </div>
        ) : (
          <Stagger className="mt-14 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
            {rows.map((s, i) => {
              const Icon = ICON_BY_CODE[s.code] ?? Shirt
              const tint = TINTS[i % TINTS.length]
              return (
                <StaggerItem key={s.id}>
                  <article className="group relative h-full overflow-hidden rounded-3xl border border-surface-200/70 bg-white p-6 shadow-card transition-all duration-300 hover:-translate-y-1 hover:border-lp-aqua/40 hover:shadow-pop">
                    <div
                      aria-hidden
                      className="absolute inset-0 bg-gradient-to-br from-lp-mint/60 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100"
                    />
                    <div className="relative">
                      <div
                        className={`flex size-12 items-center justify-center rounded-2xl bg-gradient-to-br ${tint} text-white shadow-sm transition-transform duration-300 group-hover:scale-110 group-hover:-rotate-3`}
                      >
                        <Icon className="size-5.5" aria-hidden />
                      </div>
                      <div className="mt-5 flex items-start justify-between gap-3">
                        <h3 className="text-lg font-semibold tracking-tight text-lp-ink">{s.name}</h3>
                        <span className="shrink-0 rounded-full bg-surface-100 px-2.5 py-1 text-[11px] font-bold text-lp-teal">
                          From {formatLandingPrice(s.price, s.currency)}
                          {unitSuffix(s.unit)}
                        </span>
                      </div>
                      <p className="mt-2 text-sm leading-relaxed text-lp-slate">{s.description}</p>
                    </div>
                  </article>
                </StaggerItem>
              )
            })}
          </Stagger>
        )}
      </div>
    </section>
  )
}
