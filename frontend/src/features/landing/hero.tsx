import { motion, useReducedMotion } from 'motion/react'
import { Link } from 'react-router-dom'
import { ArrowRight, Leaf, ShieldCheck, Star, Timer } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import { OrderCard } from './order-card'
import { EASE } from './reveal'

const TRUST_CHIPS = [
  { icon: Timer, label: '24-hour turnaround' },
  { icon: ShieldCheck, label: '100% care guarantee' },
  { icon: Leaf, label: 'Carbon-neutral fleet' },
]

export function Hero() {
  const { isAuthenticated } = useAuth()
  const reduce = useReducedMotion()

  return (
    <section id="top" className="relative overflow-hidden pt-32 pb-20 sm:pt-40 sm:pb-28">
      {/* Aqueous background */}
      <div aria-hidden className="pointer-events-none absolute inset-0">
        <div className="absolute inset-0 bg-[radial-gradient(60%_50%_at_50%_0%,rgb(215_240_234/0.9),rgb(242_248_247)_70%)]" />
        <div className="absolute -left-32 top-16 size-[28rem] animate-float-slow rounded-full bg-gradient-to-br from-lp-aqua/20 to-lp-cyan/10 blur-3xl" />
        <div className="absolute -right-24 top-40 size-96 animate-float-slow rounded-full bg-gradient-to-br from-lp-indigo/15 to-lp-cyan/10 blur-3xl [animation-delay:-4s]" />
      </div>

      <div className="relative mx-auto grid max-w-6xl items-center gap-14 px-4 sm:px-6 lg:grid-cols-[1.05fr_0.95fr] lg:gap-10">
        {/* Copy */}
        <div>
          <motion.p
            initial={reduce ? false : { opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, ease: EASE }}
            className="inline-flex items-center gap-2 rounded-full border border-lp-mint bg-white/80 px-3.5 py-1.5 text-xs font-semibold text-lp-teal shadow-sm backdrop-blur"
          >
            <span className="size-1.5 animate-pulse-dot rounded-full bg-lp-aqua" />
            Same-day pickup in 23 cities
          </motion.p>

          <motion.h1
            initial={reduce ? false : { opacity: 0, y: 24 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, ease: EASE, delay: 0.08 }}
            className="mt-5 font-display text-[2.75rem] font-semibold leading-[1.05] tracking-tight text-lp-ink sm:text-6xl lg:text-[4.25rem]"
          >
            Fresh laundry,{' '}
            <em className="bg-gradient-to-r from-lp-teal via-lp-cyan to-lp-indigo bg-clip-text font-medium italic text-transparent">
              folded &amp; at your door.
            </em>
          </motion.h1>

          <motion.p
            initial={reduce ? false : { opacity: 0, y: 24 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, ease: EASE, delay: 0.16 }}
            className="mt-5 max-w-xl text-base leading-relaxed text-lp-slate sm:text-lg"
          >
            LaundryPro picks up, cleans and delivers your clothes within 24 hours — tracked at
            every step, from our truck to your doorstep. No more coin machines, lost socks or
            weekend chores.
          </motion.p>

          <motion.div
            initial={reduce ? false : { opacity: 0, y: 24 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, ease: EASE, delay: 0.24 }}
            className="mt-8 flex flex-wrap items-center gap-3"
          >
            <a
              href="#book"
              className="group inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-lp-teal to-lp-indigo px-6 py-3.5 text-sm font-semibold text-white shadow-[0_10px_30px_-10px_rgb(15_118_110/0.55)] transition-all duration-200 hover:shadow-[0_16px_40px_-12px_rgb(15_118_110/0.7)] active:scale-[0.98]"
            >
              Book your first pickup
              <ArrowRight
                className="size-4 transition-transform duration-200 group-hover:translate-x-0.5"
                aria-hidden
              />
            </a>
            <a
              href="#pricing"
              className="inline-flex items-center rounded-2xl border border-surface-300/80 bg-white/80 px-6 py-3.5 text-sm font-semibold text-lp-ink shadow-sm backdrop-blur transition-all duration-150 hover:border-lp-aqua/60 hover:bg-white active:scale-[0.98]"
            >
              See pricing
            </a>
            {isAuthenticated && (
              <Link
                to="/app"
                className="text-sm font-semibold text-lp-teal underline-offset-4 transition-colors hover:text-lp-indigo hover:underline"
              >
                Open the operations console →
              </Link>
            )}
          </motion.div>

          {/* Trust row */}
          <motion.div
            initial={reduce ? false : { opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, ease: EASE, delay: 0.34 }}
            className="mt-9 flex flex-wrap items-center gap-x-7 gap-y-4"
          >
            <div className="flex items-center gap-2">
              <div className="flex text-amber-400" aria-hidden>
                {Array.from({ length: 5 }).map((_, i) => (
                  <Star key={i} className="size-3.5 fill-current" />
                ))}
              </div>
              <p className="text-sm font-semibold text-lp-ink">
                4.9 <span className="font-normal text-lp-slate">· 12,000+ reviews</span>
              </p>
            </div>
            <div className="hidden h-5 w-px bg-surface-300/70 sm:block" aria-hidden />
            <div className="flex flex-wrap gap-2">
              {TRUST_CHIPS.map((c) => (
                <span
                  key={c.label}
                  className="inline-flex items-center gap-1.5 rounded-full bg-white/70 px-3 py-1.5 text-xs font-medium text-lp-slate shadow-sm backdrop-blur"
                >
                  <c.icon className="size-3.5 text-lp-teal" aria-hidden />
                  {c.label}
                </span>
              ))}
            </div>
          </motion.div>
        </div>

        {/* Signature visual */}
        <div className="flex justify-center lg:justify-end">
          <OrderCard />
        </div>
      </div>
    </section>
  )
}
