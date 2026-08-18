import { Link } from 'react-router-dom'
import { ArrowRight, CalendarCheck, CreditCard, Truck } from 'lucide-react'
import { Reveal } from './reveal'

export function Cta() {
  return (
    <section id="book" className="px-4 pb-20 sm:px-6 sm:pb-28">
      <Reveal className="mx-auto max-w-6xl">
        <div className="relative overflow-hidden rounded-[2.5rem] bg-gradient-to-br from-lp-teal via-[#155e75] to-lp-indigo px-6 py-16 text-center text-white shadow-[0_32px_80px_-32px_rgb(15_118_110/0.6)] sm:px-12 sm:py-20">
          {/* Decorative bubbles */}
          <div aria-hidden className="pointer-events-none absolute inset-0">
            <div className="absolute -left-16 -top-16 size-64 rounded-full bg-white/10 blur-2xl" />
            <div className="absolute -bottom-24 -right-10 size-80 rounded-full bg-white/10 blur-3xl" />
            <div className="absolute right-16 top-10 size-20 animate-float-slow rounded-full border border-white/20" />
            <div className="absolute bottom-12 left-16 size-12 animate-float-slow rounded-full border border-white/20 [animation-delay:-3s]" />
          </div>

          <div className="relative">
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-white/70">
              Your laundry is one tap away
            </p>
            <h2 className="mx-auto mt-4 max-w-2xl font-display text-4xl font-semibold leading-[1.08] tracking-tight sm:text-5xl">
              Fresh laundry, folded — <em className="italic">starting tonight.</em>
            </h2>
            <p className="mx-auto mt-4 max-w-xl text-sm leading-relaxed text-white/90 sm:text-base">
              Book before 9 PM and we'll be at your door tomorrow. Free pickup on your first order —
              no minimum, no commitment.
            </p>

            <div className="mt-9 flex flex-wrap items-center justify-center gap-3">
              <Link
                to="/login"
                className="group inline-flex items-center gap-2 rounded-2xl bg-white px-7 py-3.5 text-sm font-bold text-lp-teal shadow-[0_12px_32px_-12px_rgb(2_6_23/0.5)] transition-all duration-200 hover:scale-[1.03] hover:shadow-[0_16px_40px_-12px_rgb(2_6_23/0.6)] active:scale-[0.98]"
              >
                Book your first pickup
                <ArrowRight
                  className="size-4 transition-transform duration-200 group-hover:translate-x-0.5"
                  aria-hidden
                />
              </Link>
              <a
                href="#pricing"
                className="inline-flex items-center rounded-2xl border border-white/30 bg-white/10 px-7 py-3.5 text-sm font-semibold text-white backdrop-blur transition-colors duration-150 hover:bg-white/20"
              >
                Compare plans
              </a>
            </div>

            <div className="mt-8 flex flex-wrap items-center justify-center gap-x-7 gap-y-2 text-xs font-medium text-white/75">
              <span className="inline-flex items-center gap-1.5">
                <CalendarCheck className="size-3.5" aria-hidden /> Book in 2 minutes
              </span>
              <span className="inline-flex items-center gap-1.5">
                <Truck className="size-3.5" aria-hidden /> Free first pickup
              </span>
              <span className="inline-flex items-center gap-1.5">
                <CreditCard className="size-3.5" aria-hidden /> Pay online or on delivery
              </span>
            </div>
          </div>
        </div>
      </Reveal>
    </section>
  )
}
