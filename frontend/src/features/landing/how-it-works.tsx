import { STEPS } from './data'
import { SectionHeading, Stagger, StaggerItem } from './reveal'

export function HowItWorks() {
  return (
    <section id="how-it-works" className="relative py-20 sm:py-28">
      <div className="mx-auto max-w-6xl px-4 sm:px-6">
        <SectionHeading
          eyebrow="How it works"
          title={
            <>
              From your wardrobe to your door —{' '}
              <em className="bg-gradient-to-r from-lp-teal to-lp-indigo bg-clip-text font-medium italic text-transparent">
                four calm steps.
              </em>
            </>
          }
          sub="A real sequence, not a black box. You always know which step your laundry is on — the same pipeline our operations team sees."
          center
        />

        <Stagger className="relative mt-14 grid gap-10 sm:grid-cols-2 lg:grid-cols-4 lg:gap-6">
          {/* Connector line (desktop) */}
          <div
            aria-hidden
            className="absolute left-[12.5%] right-[12.5%] top-7 hidden h-px bg-gradient-to-r from-lp-aqua/40 via-lp-cyan/40 to-lp-indigo/40 lg:block"
          />

          {STEPS.map((step, i) => {
            const Icon = step.icon
            return (
              <StaggerItem key={step.title} className="relative">
                <div className="group">
                  <div className="relative z-10 flex size-14 items-center justify-center rounded-2xl border border-white/70 bg-white text-lp-teal shadow-pop transition-all duration-200 group-hover:-translate-y-1 group-hover:shadow-[0_16px_32px_-12px_rgb(15_118_110/0.35)]">
                    <Icon className="size-6" aria-hidden />
                    <span className="absolute -right-2 -top-2 flex size-6 items-center justify-center rounded-full bg-gradient-to-br from-lp-teal to-lp-indigo font-mono text-[10px] font-bold text-white">
                      {String(i + 1).padStart(2, '0')}
                    </span>
                  </div>
                  <h3 className="mt-5 text-lg font-semibold tracking-tight text-lp-ink">
                    {step.title}
                  </h3>
                  <p className="mt-2 text-sm leading-relaxed text-lp-slate">{step.copy}</p>
                  <p className="mt-3 inline-flex items-center gap-1.5 rounded-full bg-lp-mint/80 px-2.5 py-1 text-[11px] font-semibold text-lp-teal">
                    {step.chip}
                  </p>
                </div>
              </StaggerItem>
            )
          })}
        </Stagger>
      </div>
    </section>
  )
}
