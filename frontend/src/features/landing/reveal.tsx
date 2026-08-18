import { motion, useReducedMotion } from 'motion/react'
import type { Variants } from 'motion/react'

/** Shared easing for the landing — one rhythm across every transition. */
export const EASE = [0.16, 1, 0.3, 1] as const

/** Fade-and-rise reveal that triggers once when scrolled into view. */
export function Reveal({
  children,
  className,
  delay = 0,
  y = 28,
  amount = 0.25,
}: {
  children: React.ReactNode
  className?: string
  delay?: number
  y?: number
  amount?: number
}) {
  const reduce = useReducedMotion()
  return (
    <motion.div
      className={className}
      initial={reduce ? false : { opacity: 0, y }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, amount }}
      transition={{ duration: 0.7, ease: EASE, delay }}
    >
      {children}
    </motion.div>
  )
}

export const staggerContainer: Variants = {
  hidden: {},
  visible: {
    transition: { staggerChildren: 0.09, delayChildren: 0.05 },
  },
}

export const staggerItem: Variants = {
  hidden: { opacity: 0, y: 22 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.6, ease: EASE },
  },
}

/** Staggered-grid wrapper; children use <StaggerItem> for orchestrated entry. */
export function Stagger({
  children,
  className,
  amount = 0.2,
}: {
  children: React.ReactNode
  className?: string
  amount?: number
}) {
  return (
    <motion.div
      className={className}
      variants={staggerContainer}
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, amount }}
    >
      {children}
    </motion.div>
  )
}

export function StaggerItem({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return (
    <motion.div className={className} variants={staggerItem}>
      {children}
    </motion.div>
  )
}

/** Small eyebrow label used above section headings. */
export function Eyebrow({ children }: { children: React.ReactNode }) {
  return (
    <p className="mb-3 text-xs font-semibold uppercase tracking-[0.18em] text-lp-teal">
      {children}
    </p>
  )
}

/** Section heading with the Fraunces display face. */
export function SectionHeading({
  eyebrow,
  title,
  sub,
  center,
}: {
  eyebrow: string
  title: React.ReactNode
  sub?: string
  center?: boolean
}) {
  return (
    <Reveal className={center ? 'mx-auto max-w-2xl text-center' : 'max-w-2xl'}>
      <Eyebrow>{eyebrow}</Eyebrow>
      <h2 className="font-display text-3xl font-semibold leading-[1.12] tracking-tight text-lp-ink sm:text-[2.6rem]">
        {title}
      </h2>
      {sub && <p className="mt-4 text-base leading-relaxed text-lp-slate sm:text-lg">{sub}</p>}
    </Reveal>
  )
}
