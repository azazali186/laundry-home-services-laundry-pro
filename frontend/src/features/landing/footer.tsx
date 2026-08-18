import { Link } from 'react-router-dom'
import { ArrowUpRight, Shirt } from 'lucide-react'

const COLUMNS: { title: string; links: { label: string; href: string }[] }[] = [
  {
    title: 'Services',
    links: [
      { label: 'Wash & Fold', href: '#services' },
      { label: 'Dry Cleaning', href: '#services' },
      { label: 'Ironing & Pressing', href: '#services' },
      { label: 'Corporate & Staff', href: '#services' },
    ],
  },
  {
    title: 'Company',
    links: [
      { label: 'How it works', href: '#how-it-works' },
      { label: 'Pricing', href: '#pricing' },
      { label: 'Careers', href: '#top' },
      { label: 'Press', href: '#top' },
    ],
  },
  {
    title: 'Support',
    links: [
      { label: 'FAQ', href: '#faq' },
      { label: 'Care guarantee', href: '#faq' },
      { label: 'Privacy policy', href: '#top' },
      { label: 'Terms of service', href: '#top' },
    ],
  },
]

export function Footer() {
  return (
    <footer className="bg-lp-ink text-white">
      <div className="mx-auto max-w-6xl px-4 py-16 sm:px-6">
        <div className="grid gap-12 lg:grid-cols-[1.2fr_2fr]">
          {/* Brand */}
          <div>
            <a href="#top" className="flex items-center gap-2.5">
              <span className="flex size-9 items-center justify-center rounded-xl bg-gradient-to-br from-lp-teal to-lp-indigo text-white">
                <Shirt className="size-4.5" aria-hidden />
              </span>
              <span className="font-display text-lg font-semibold tracking-tight">
                Laundry<span className="text-lp-aqua">Pro</span>
              </span>
            </a>
            <p className="mt-4 max-w-xs text-sm leading-relaxed text-white/60">
              Laundry picked up, cleaned and delivered within 24 hours — tracked at every step,
              from our truck to your doorstep.
            </p>
            <div className="mt-6 flex items-center gap-2 rounded-full border border-white/15 bg-white/5 px-3.5 py-1.5 text-xs font-medium text-white/70 w-fit">
              <span className="size-1.5 rounded-full bg-emerald-400" />
              All systems operational
            </div>
          </div>

          {/* Link columns */}
          <div className="grid grid-cols-2 gap-8 sm:grid-cols-3">
            {COLUMNS.map((col) => (
              <div key={col.title}>
                <h3 className="text-xs font-bold uppercase tracking-[0.16em] text-white/45">
                  {col.title}
                </h3>
                <ul className="mt-4 space-y-2.5">
                  {col.links.map((l) => (
                    <li key={l.label}>
                      <a
                        href={l.href}
                        className="text-sm text-white/70 transition-colors duration-150 hover:text-white"
                      >
                        {l.label}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>

        <div className="mt-14 flex flex-col gap-4 border-t border-white/10 pt-7 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-xs text-white/45">
            © {new Date().getFullYear()} Aeroxe Enterprises Private limited
          </p>
          <div className="flex items-center gap-5">
            <Link
              to="/login"
              className="inline-flex items-center gap-1 text-xs font-semibold text-white/70 transition-colors duration-150 hover:text-white"
            >
              Staff sign in <ArrowUpRight className="size-3" aria-hidden />
            </Link>
            <Link
              to="/app"
              className="inline-flex items-center gap-1 text-xs font-semibold text-white/70 transition-colors duration-150 hover:text-white"
            >
              Operations console <ArrowUpRight className="size-3" aria-hidden />
            </Link>
          </div>
        </div>
      </div>
    </footer>
  )
}
