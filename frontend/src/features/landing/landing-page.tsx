import { Navbar } from './navbar'
import { Hero } from './hero'
import { StatsBand } from './stats'
import { HowItWorks } from './how-it-works'
import { Services } from './services'
import { Pricing } from './pricing'
import { Testimonials } from './testimonials'
import { Faq } from './faq'
import { Cta } from './cta'
import { Footer } from './footer'
import { usePublicPricing } from './use-public-pricing'

export function LandingPage() {
  // All pricing shown on the landing page is fetched from the backend's
  // public /pricing endpoint — nothing is hardcoded.
  const pricing = usePublicPricing()

  const catalogProps = {
    isLoading: pricing.isLoading,
    isError: pricing.isError,
    onRetry: () => void pricing.refetch(),
  }

  return (
    <div className="min-h-dvh bg-lp-mist font-sans text-lp-ink antialiased">
      <Navbar />
      <main>
        <Hero />
        <StatsBand />
        <HowItWorks />
        <Services services={pricing.data?.services} {...catalogProps} />
        <Pricing combos={pricing.data?.combos} services={pricing.data?.services} {...catalogProps} />
        <Testimonials />
        <Faq />
        <Cta />
      </main>
      <Footer />
    </div>
  )
}
