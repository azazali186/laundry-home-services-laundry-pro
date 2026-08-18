import { useState } from 'react'
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import { FAQS } from './data'
import { EASE, SectionHeading, Stagger, StaggerItem } from './reveal'

export function Faq() {
  const reduce = useReducedMotion()
  const [open, setOpen] = useState<number | null>(0)

  return (
    <section id="faq" className="relative py-20 sm:py-28">
      <div className="mx-auto grid max-w-6xl gap-12 px-4 sm:px-6 lg:grid-cols-[0.85fr_1.15fr] lg:gap-16">
        <div className="lg:sticky lg:top-28 lg:self-start">
          <SectionHeading
            eyebrow="FAQ"
            title={
              <>
                Questions we hear{' '}
                <em className="bg-gradient-to-r from-lp-teal to-lp-indigo bg-clip-text font-medium italic text-transparent">
                  every day.
                </em>
              </>
            }
            sub="If yours isn't here, write to hello@laundrypro.example — a human replies within the hour, 7 days a week."
          />
        </div>

        <Stagger className="space-y-3">
          {FAQS.map((f, i) => {
            const isOpen = open === i
            return (
              <StaggerItem key={f.q}>
                <div
                  className={cn(
                    'overflow-hidden rounded-2xl border transition-colors duration-200',
                    isOpen
                      ? 'border-lp-aqua/40 bg-white shadow-pop'
                      : 'border-surface-200/70 bg-white/70 hover:border-lp-aqua/30 hover:bg-white',
                  )}
                >
                  <button
                    onClick={() => setOpen(isOpen ? null : i)}
                    aria-expanded={isOpen}
                    className="flex w-full items-center justify-between gap-4 px-5 py-4.5 text-left"
                  >
                    <span className="text-[15px] font-semibold text-lp-ink">{f.q}</span>
                    <motion.span
                      animate={{ rotate: isOpen ? 180 : 0 }}
                      transition={{ duration: 0.25, ease: EASE }}
                      className={cn(
                        'flex size-7 shrink-0 items-center justify-center rounded-full transition-colors duration-200',
                        isOpen ? 'bg-lp-mint text-lp-teal' : 'bg-surface-100 text-lp-slate',
                      )}
                    >
                      <ChevronDown className="size-4" aria-hidden />
                    </motion.span>
                  </button>
                  <AnimatePresence initial={false}>
                    {isOpen && (
                      <motion.div
                        key="answer"
                        initial={reduce ? false : { height: 0, opacity: 0 }}
                        animate={{ height: 'auto', opacity: 1 }}
                        exit={reduce ? undefined : { height: 0, opacity: 0 }}
                        transition={{ duration: 0.32, ease: EASE }}
                        className="overflow-hidden"
                      >
                        <p className="px-5 pb-5 text-sm leading-relaxed text-lp-slate">{f.a}</p>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>
              </StaggerItem>
            )
          })}
        </Stagger>
      </div>
    </section>
  )
}
