import { useEffect, useRef, useState } from 'react'
import { motion, useInView, useReducedMotion } from 'motion/react'
import { STATS, PARTNERS } from './data'
import { EASE } from './reveal'

/** Animates a number from 0 to its target when it enters the viewport. */
function CountUp({
  value,
  decimals,
  suffix,
}: {
  value: number
  decimals?: number
  suffix: string
}) {
  const ref = useRef<HTMLSpanElement>(null)
  const inView = useInView(ref, { once: true, amount: 0.6 })
  const reduce = useReducedMotion()
  const [display, setDisplay] = useState(reduce ? value : 0)

  useEffect(() => {
    if (!inView) return
    if (reduce) {
      setDisplay(value)
      return
    }
    const duration = 1400
    const start = performance.now()
    let raf = 0
    const tick = (now: number) => {
      const t = Math.min((now - start) / duration, 1)
      const eased = 1 - Math.pow(1 - t, 3)
      setDisplay(value * eased)
      if (t < 1) raf = requestAnimationFrame(tick)
    }
    raf = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(raf)
  }, [inView, value, reduce])

  const formatted =
    decimals != null ? display.toFixed(decimals) : Math.round(display).toLocaleString('en-IN')

  return (
    <span ref={ref}>
      {formatted}
      <span className="text-lp-aqua">{suffix}</span>
    </span>
  )
}

export function StatsBand() {
  return (
    <section className="relative border-y border-white/60 bg-white/60 backdrop-blur-sm">
      <div className="mx-auto max-w-6xl px-4 py-14 sm:px-6">
        <motion.dl
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, amount: 0.4 }}
          variants={{
            hidden: {},
            visible: { transition: { staggerChildren: 0.1 } },
          }}
          className="grid grid-cols-2 gap-8 lg:grid-cols-4"
        >
          {STATS.map((s) => (
            <motion.div
              key={s.label}
              variants={{
                hidden: { opacity: 0, y: 18 },
                visible: { opacity: 1, y: 0, transition: { duration: 0.55, ease: EASE } },
              }}
              className="text-center"
            >
              <p className="font-display text-4xl font-semibold tracking-tight text-lp-ink sm:text-5xl">
                <CountUp value={s.value} decimals={s.decimals} suffix={s.suffix} />
              </p>
              <p className="mt-1.5 text-xs font-medium uppercase tracking-[0.14em] text-lp-slate">
                {s.label}
              </p>
            </motion.div>
          ))}
        </motion.dl>
      </div>

      {/* Partner marquee */}
      <div className="border-t border-white/60 py-5">
        <div
          className="group relative overflow-hidden"
          style={{
            maskImage: 'linear-gradient(90deg, transparent, black 12%, black 88%, transparent)',
            WebkitMaskImage: 'linear-gradient(90deg, transparent, black 12%, black 88%, transparent)',
          }}
        >
          <div className="flex w-max animate-marquee gap-14 group-hover:[animation-play-state:paused]">
            {[...PARTNERS, ...PARTNERS].map((p, i) => (
              <span
                key={`${p}-${i}`}
                className="whitespace-nowrap font-display text-lg font-medium text-surface-500 transition-colors duration-200 hover:text-lp-teal"
              >
                {p}
              </span>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
