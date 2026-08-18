import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { AnimatePresence, motion } from 'motion/react'
import { ArrowRight, Menu, Shirt, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useAuth } from '@/hooks/useAuth'
import { NAV_LINKS } from './data'
import { EASE } from './reveal'

export function Navbar() {
  const { isAuthenticated } = useAuth()
  const [scrolled, setScrolled] = useState(false)
  const [open, setOpen] = useState(false)

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 12)
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  // Lock body scroll while the mobile menu is open.
  useEffect(() => {
    document.body.style.overflow = open ? 'hidden' : ''
    return () => {
      document.body.style.overflow = ''
    }
  }, [open])

  const ctaHref = isAuthenticated ? '/app' : '/login'
  const ctaLabel = isAuthenticated ? 'Open console' : 'Sign in'

  return (
    <header
      className={cn(
        'fixed inset-x-0 top-0 z-50 transition-all duration-300',
        scrolled
          ? 'border-b border-white/50 bg-white/75 shadow-[0_1px_12px_rgb(13_28_40/0.06)] backdrop-blur-xl'
          : 'bg-transparent',
      )}
    >
      <nav className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
        {/* Brand */}
        <a href="#top" className="group flex items-center gap-2.5">
          <span className="flex size-9 items-center justify-center rounded-xl bg-gradient-to-br from-lp-teal to-lp-indigo text-white shadow-sm transition-transform duration-200 group-hover:-rotate-6">
            <Shirt className="size-4.5" aria-hidden />
          </span>
          <span className="font-display text-lg font-semibold tracking-tight text-lp-ink">
            Laundry<span className="text-lp-teal">Pro</span>
          </span>
        </a>

        {/* Desktop links */}
        <div className="hidden items-center gap-1 md:flex">
          {NAV_LINKS.map((l) => (
            <a
              key={l.href}
              href={l.href}
              className="rounded-lg px-3.5 py-2 text-sm font-medium text-lp-slate transition-colors duration-150 hover:bg-white/70 hover:text-lp-ink"
            >
              {l.label}
            </a>
          ))}
        </div>

        {/* Desktop CTAs */}
        <div className="hidden items-center gap-2 md:flex">
          <Link
            to={ctaHref}
            className="inline-flex items-center gap-1.5 rounded-xl px-4 py-2 text-sm font-semibold text-lp-teal transition-colors duration-150 hover:text-lp-ink"
          >
            {ctaLabel}
            <ArrowRight className="size-3.5" aria-hidden />
          </Link>
          <a
            href="#book"
            className="inline-flex items-center rounded-xl bg-gradient-to-r from-lp-teal to-lp-indigo px-4.5 py-2.5 text-sm font-semibold text-white shadow-sm transition-all duration-200 hover:shadow-[0_8px_24px_-8px_rgb(15_118_110/0.6)] active:scale-[0.98]"
          >
            Book a pickup
          </a>
        </div>

        {/* Mobile toggle */}
        <button
          onClick={() => setOpen((v) => !v)}
          aria-label={open ? 'Close menu' : 'Open menu'}
          aria-expanded={open}
          className="rounded-lg p-2 text-lp-ink transition-colors hover:bg-white/70 md:hidden"
        >
          {open ? <X className="size-5" /> : <Menu className="size-5" />}
        </button>
      </nav>

      {/* Mobile drawer */}
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.28, ease: EASE }}
            className="overflow-hidden border-b border-white/60 bg-white/95 backdrop-blur-xl md:hidden"
          >
            <div className="space-y-1 px-4 pb-5 pt-2">
              {NAV_LINKS.map((l, i) => (
                <motion.a
                  key={l.href}
                  href={l.href}
                  initial={{ opacity: 0, x: -12 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: 0.05 + i * 0.05, duration: 0.25 }}
                  onClick={() => setOpen(false)}
                  className="block rounded-xl px-3 py-3 text-base font-medium text-lp-ink transition-colors hover:bg-lp-mist"
                >
                  {l.label}
                </motion.a>
              ))}
              <div className="flex gap-3 pt-3">
                <Link
                  to={ctaHref}
                  onClick={() => setOpen(false)}
                  className="flex-1 rounded-xl border border-surface-300 bg-white px-4 py-3 text-center text-sm font-semibold text-lp-ink transition-colors hover:bg-surface-100"
                >
                  {ctaLabel}
                </Link>
                <a
                  href="#book"
                  onClick={() => setOpen(false)}
                  className="flex-1 rounded-xl bg-gradient-to-r from-lp-teal to-lp-indigo px-4 py-3 text-center text-sm font-semibold text-white"
                >
                  Book a pickup
                </a>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </header>
  )
}
