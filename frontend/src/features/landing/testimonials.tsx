import { Star } from 'lucide-react'
import { cn } from '@/lib/utils'
import { TESTIMONIALS } from './data'
import { SectionHeading, Stagger, StaggerItem } from './reveal'

export function Testimonials() {
  const [featured, ...rest] = TESTIMONIALS

  return (
    <section className="relative bg-white py-20 sm:py-28">
      <div className="mx-auto max-w-6xl px-4 sm:px-6">
        <SectionHeading
          eyebrow="Loved by customers"
          title={
            <>
              Real households,{' '}
              <em className="bg-gradient-to-r from-lp-teal to-lp-indigo bg-clip-text font-medium italic text-transparent">
                real weekends back.
              </em>
            </>
          }
          sub="12,000+ five-star reviews across app stores. Here's what a few of them sound like."
          center
        />

        <Stagger className="mt-14 grid gap-5 lg:grid-cols-2">
          {/* Featured quote spans two rows on desktop; the rest stack beside it */}
          <StaggerItem className="lg:row-span-2">
            <figure className="flex h-full flex-col justify-between rounded-3xl bg-gradient-to-br from-lp-teal to-lp-indigo p-7 text-white shadow-[0_24px_60px_-24px_rgb(15_118_110/0.6)] sm:p-9">
              <div>
                <Stars className="mb-5 text-white/90" />
                <blockquote className="font-display text-xl font-medium leading-snug tracking-tight sm:text-2xl">
                  “{featured.quote}”
                </blockquote>
              </div>
              <figcaption className="mt-8 flex items-center gap-3">
                <Avatar initials={featured.initials} tint={featured.tint} />
                <div>
                  <p className="text-sm font-bold">{featured.name}</p>
                  <p className="text-xs text-white/70">{featured.role}</p>
                </div>
              </figcaption>
            </figure>
          </StaggerItem>

          {rest.map((t) => (
            <StaggerItem key={t.name}>
              <figure className="flex h-full flex-col justify-between rounded-3xl border border-surface-200/70 bg-white p-7 shadow-card transition-all duration-300 hover:-translate-y-1 hover:shadow-pop">
                <div>
                  <Stars />
                  <blockquote className="mt-4 text-[15px] leading-relaxed text-lp-ink">
                    “{t.quote}”
                  </blockquote>
                </div>
                <figcaption className="mt-6 flex items-center gap-3">
                  <Avatar initials={t.initials} tint={t.tint} />
                  <div>
                    <p className="text-sm font-bold text-lp-ink">{t.name}</p>
                    <p className="text-xs text-lp-slate">{t.role}</p>
                  </div>
                </figcaption>
              </figure>
            </StaggerItem>
          ))}
        </Stagger>
      </div>
    </section>
  )
}

function Stars({ className }: { className?: string }) {
  return (
    <div className={cn('flex gap-0.5 text-amber-400', className)} aria-label="5 out of 5 stars">
      {Array.from({ length: 5 }).map((_, i) => (
        <Star key={i} className="size-4 fill-current" aria-hidden />
      ))}
    </div>
  )
}

function Avatar({ initials, tint }: { initials: string; tint: string }) {
  return (
    <span
      className={cn(
        'flex size-11 shrink-0 items-center justify-center rounded-full bg-gradient-to-br text-sm font-bold text-white shadow-sm',
        tint,
      )}
    >
      {initials}
    </span>
  )
}
