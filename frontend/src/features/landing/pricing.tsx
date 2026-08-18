import { Check, Clock, RefreshCw, Sparkles } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { PricingComboView, PricingServiceView } from '@/services/pricing.service'
import { buildIncludedLines, perUnitHint } from './pricing-breakdown'
import { formatLandingPrice, humanizeUnit } from './price-format'
import { SectionHeading, Stagger, StaggerItem } from './reveal'

interface PricingProps {
  combos: PricingComboView[] | undefined
  services: PricingServiceView[] | undefined
  isLoading: boolean
  isError: boolean
  onRetry: () => void
}

export function Pricing({ combos, services, isLoading, isError, onRetry }: PricingProps) {
  const rows = combos ?? []
  const showSkeleton = isLoading && !isError && rows.length === 0

  return (
    <section id="pricing" className="relative bg-lp-mist/70 py-20 sm:py-28">
      <div className="mx-auto max-w-6xl px-4 sm:px-6">
        <SectionHeading
          eyebrow="Pricing"
          title={
            <>
              Simple plans,{' '}
              <em className="bg-gradient-to-r from-lp-teal to-lp-indigo bg-clip-text font-medium italic text-transparent">
                no fine print.
              </em>
            </>
          }
          sub="Start with a single load. Move to a pass when the laundry mountain starts to feel routine."
          center
        />

        {showSkeleton ? (
          <div className="mt-14 grid items-stretch gap-6 lg:grid-cols-3" aria-busy="true">
            {Array.from({ length: 3 }).map((_, i) => (
              <div
                key={i}
                className="h-96 animate-pulse rounded-3xl border border-surface-200/70 bg-surface-100/60"
              />
            ))}
          </div>
        ) : isError || rows.length === 0 ? (
          <div className="mt-14 flex flex-col items-center rounded-3xl border border-dashed border-surface-300/80 bg-white/60 px-6 py-14 text-center">
            <p className="text-sm font-semibold text-lp-ink">Plans are temporarily unavailable</p>
            <p className="mt-1.5 max-w-md text-sm text-lp-slate">
              We couldn’t load the live plans. Please try again — pricing is always sourced from
              the shop in real time.
            </p>
            <button
              onClick={onRetry}
              className="mt-6 inline-flex items-center gap-2 rounded-xl bg-lp-teal px-5 py-2.5 text-sm font-semibold text-white shadow-sm transition-all duration-200 hover:bg-lp-teal-dark active:scale-[0.98]"
            >
              <RefreshCw className="size-4" aria-hidden /> Try again
            </button>
          </div>
        ) : (
          <Stagger className="mt-14 grid items-stretch gap-6 lg:grid-cols-3">
            {rows.map((plan) => {
              const featured = plan.is_featured
              const included = buildIncludedLines(plan, services)
              const hint = perUnitHint(plan)
              const hasValue = (plan.regular_price ?? 0) > 0
              const savings = plan.savings ?? 0
              return (
                <StaggerItem key={plan.id} className="h-full">
                  <article
                    className={cn(
                      'relative flex h-full flex-col rounded-3xl border p-7 transition-all duration-300',
                      featured
                        ? 'border-transparent bg-gradient-to-br from-lp-teal to-lp-indigo text-white shadow-[0_24px_60px_-20px_rgb(15_118_110/0.55)] lg:-my-4 lg:py-11'
                        : 'border-surface-200/70 bg-white shadow-card hover:-translate-y-1 hover:shadow-pop',
                    )}
                  >
                    {featured && (
                      <span className="absolute -top-3.5 left-1/2 -translate-x-1/2 rounded-full bg-white px-3.5 py-1 text-[11px] font-bold uppercase tracking-wider text-lp-teal shadow-sm">
                        Most popular
                      </span>
                    )}

                    <h3
                      className={cn(
                        'text-sm font-bold uppercase tracking-[0.14em]',
                        featured ? 'text-white/85' : 'text-lp-teal',
                      )}
                    >
                      {plan.name}
                    </h3>
                    <p
                      className={cn(
                        'mt-2 text-sm leading-relaxed',
                        featured ? 'text-white/75' : 'text-lp-slate',
                      )}
                    >
                      {plan.description}
                    </p>

                    <div className="mt-5 flex items-baseline gap-1.5">
                      <span
                        className={cn(
                          'font-display text-5xl font-semibold tracking-tight',
                          featured ? 'text-white' : 'text-lp-ink',
                        )}
                      >
                        {formatLandingPrice(plan.price, plan.currency)}
                      </span>
                      <span
                        className={cn(
                          'text-sm font-medium',
                          featured ? 'text-white/70' : 'text-lp-slate',
                        )}
                      >
                        {humanizeUnit(plan.unit)}
                      </span>
                    </div>

                    {hint && (
                      <p
                        className={cn(
                          'mt-2 text-sm font-medium',
                          featured ? 'text-white/70' : 'text-lp-slate',
                        )}
                      >
                        ≈ {formatLandingPrice(hint.value, plan.currency)}/{hint.label}
                      </p>
                    )}

                    {plan.validity_text && (
                      <div
                        className={cn(
                          'mt-3 flex items-center gap-1.5 text-xs',
                          featured ? 'text-white/65' : 'text-lp-slate',
                        )}
                      >
                        <Clock className="size-3.5 shrink-0" aria-hidden />
                        <span>{plan.validity_text}</span>
                      </div>
                    )}

                    <p
                      className={cn(
                        'mt-6 text-[11px] font-bold uppercase tracking-[0.14em]',
                        featured ? 'text-white/60' : 'text-lp-slate',
                      )}
                    >
                      What’s included
                    </p>
                    <ul className="mt-3 flex-1 space-y-3">
                      {included.map((line, i) => (
                        <li
                          key={`${line.label}-${i}`}
                          className="flex items-start justify-between gap-3 text-sm"
                        >
                          <span className="flex min-w-0 items-start gap-2.5">
                            <span
                              className={cn(
                                'mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full',
                                featured ? 'bg-white/20 text-white' : 'bg-lp-mint text-lp-teal',
                              )}
                            >
                              <Check className="size-3" aria-hidden />
                            </span>
                            <span className="min-w-0">
                              <span
                                className={cn(
                                  'font-medium',
                                  featured ? 'text-white/90' : 'text-lp-ink',
                                )}
                              >
                                {line.quantity} × {line.label}
                              </span>
                              {line.unit && (
                                <span
                                  className={cn(
                                    'mt-0.5 block text-xs',
                                    featured ? 'text-white/55' : 'text-lp-slate',
                                  )}
                                >
                                  {humanizeUnit(line.unit)}
                                </span>
                              )}
                            </span>
                          </span>
                          {line.lineValue !== undefined && (
                            <span
                              className={cn(
                                'shrink-0 font-medium',
                                featured ? 'text-white/85' : 'text-lp-ink',
                              )}
                            >
                              {formatLandingPrice(line.lineValue, line.currency)}
                            </span>
                          )}
                        </li>
                      ))}
                    </ul>

                    {hasValue && (
                      <div
                        className={cn(
                          'mt-5 border-t pt-4',
                          featured ? 'border-white/15' : 'border-surface-200/80',
                        )}
                      >
                        <div className="flex items-center justify-between text-sm">
                          <span className={featured ? 'text-white/65' : 'text-lp-slate'}>
                            À la carte value
                          </span>
                          <span
                            className={cn(
                              'font-medium line-through decoration-1',
                              featured ? 'text-white/60' : 'text-lp-slate',
                            )}
                          >
                            {formatLandingPrice(plan.regular_price, plan.currency)}
                          </span>
                        </div>
                        {savings > 0 && (
                          <span
                            className={cn(
                              'mt-3 inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-semibold',
                              featured ? 'bg-white/20 text-white' : 'bg-lp-mint text-lp-teal',
                            )}
                          >
                            <Sparkles className="size-3.5" aria-hidden />
                            You save {formatLandingPrice(savings, plan.currency)}
                          </span>
                        )}
                      </div>
                    )}

                    <a
                      href="#book"
                      className={cn(
                        'mt-8 inline-flex items-center justify-center rounded-xl px-5 py-3 text-sm font-semibold transition-all duration-200 active:scale-[0.98]',
                        featured
                          ? 'bg-white text-lp-teal shadow-sm hover:bg-lp-mist'
                          : 'border border-surface-300/80 bg-white text-lp-ink hover:border-lp-aqua/60 hover:bg-lp-mist/50',
                      )}
                    >
                      Choose plan
                    </a>
                  </article>
                </StaggerItem>
              )
            })}
          </Stagger>
        )}

        <p className="mt-10 text-center text-sm text-lp-slate">
          Prices vary by city &amp; fabric type. GST included. Cancel any pass with one tap.
        </p>
      </div>
    </section>
  )
}
