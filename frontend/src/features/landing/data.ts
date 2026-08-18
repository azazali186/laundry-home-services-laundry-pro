import { CalendarCheck, PackageCheck, Sparkles, Truck, type LucideIcon } from 'lucide-react'

/* ---------------------------------------------------------------------------
   LaundryPro landing content. Copy is part of the design — kept specific,
   plain and from the customer's side of the counter.
--------------------------------------------------------------------------- */

export interface Step {
  title: string
  copy: string
  chip: string
  icon: LucideIcon
}

export const STEPS: Step[] = [
  {
    title: 'Book in two minutes',
    copy: 'Choose a service, pick a pickup window, pay online. No calls, no forms, no waiting on hold.',
    chip: 'Takes ~2 min',
    icon: CalendarCheck,
  },
  {
    title: 'We pick it up',
    copy: 'Your driver collects the bag at the scheduled slot and confirms with a photo — track them live.',
    chip: 'Live tracking',
    icon: Truck,
  },
  {
    title: 'We clean & inspect',
    copy: 'Washed, dried, ironed and hand-checked for stains or damage before anything leaves the plant.',
    chip: 'Quality gate',
    icon: Sparkles,
  },
  {
    title: 'Delivered, folded',
    copy: 'Back at your door within 24 hours — folded, tagged and photographed on arrival.',
    chip: '24h turnaround',
    icon: PackageCheck,
  },
]

// Note: service prices and combo plans intentionally live on the backend
// (public /pricing endpoint). The landing page renders them from there — no
// hardcoded pricing in this codebase.

export interface Testimonial {
  quote: string
  name: string
  role: string
  initials: string
  tint: string
}

export const TESTIMONIALS: Testimonial[] = [
  {
    quote:
      "I've used laundry apps for years — nothing comes close. The photo confirmation before delivery is genius. My favourite shirt finally survives a wash cycle.",
    name: 'Ananya Iyer',
    role: 'Product designer · Bengaluru',
    initials: 'AI',
    tint: 'from-teal-500 to-cyan-500',
  },
  {
    quote:
      'We moved our whole team to the corporate plan. Pickup at the office, delivery to the office, one monthly bill. It just works, every single week.',
    name: 'Rohan Mehta',
    role: 'Founder, 40-person startup',
    initials: 'RM',
    tint: 'from-indigo-500 to-violet-500',
  },
  {
    quote:
      'With two kids, laundry used to eat my whole weekend. Now it is someone else’s problem — and the app tells me exactly when it is back at my door.',
    name: 'Sara Fernandes',
    role: 'Teacher & mother of two',
    initials: 'SF',
    tint: 'from-emerald-500 to-teal-500',
  },
]

export interface Faq {
  q: string
  a: string
}

export const FAQS: Faq[] = [
  {
    q: 'How fast is the turnaround?',
    a: 'Orders booked before 9 AM can come back the same evening in most cities. Standard turnaround is 24 hours; members on the Weekly pass get priority service on top of that.',
  },
  {
    q: 'What if I’m not home at pickup or delivery?',
    a: 'Leave the bag with your concierge, reception or a door mat — your driver confirms collection with a photo. Everything is tagged and tracked, so it always finds its way back to you.',
  },
  {
    q: 'How do you handle delicates and stains?',
    a: 'Every garment is tagged with your care instructions. Silk, wool and structured items go to our certified dry-cleaning line, and stubborn stains are pre-treated by hand before the main wash.',
  },
  {
    q: 'Is my payment information secure?',
    a: 'Payments are tokenized and processed through PCI-compliant gateways — your card number never touches our servers. You can pay per order, or store a card with one-click checkout.',
  },
  {
    q: 'What if something gets damaged?',
    a: 'We photograph every bag on arrival and before dispatch. If anything is ever damaged, our 100% care guarantee covers repair or fair compensation — no fine print, no argument.',
  },
  {
    q: 'Can I set up recurring pickups?',
    a: 'Yes. Weekly, fortnightly or monthly schedules with a fixed slot and driver. Change or pause any cycle from the app — no calls, no cancellation fees.',
  },
]

export const PARTNERS = [
  'UrbanNest Residences',
  'Skyline Suites',
  'Maison Hotels',
  'Campus & Co',
  'Pearl Residences',
  'Bloom Group',
]

export const STATS = [
  { value: 120, suffix: 'K+', label: 'loads delivered' },
  { value: 4.9, suffix: '★', label: 'average rating', decimals: 1 },
  { value: 24, suffix: 'h', label: 'average turnaround' },
  { value: 98.7, suffix: '%', label: 'on-time deliveries', decimals: 1 },
]

export const NAV_LINKS = [
  { href: '#services', label: 'Services' },
  { href: '#how-it-works', label: 'How it works' },
  { href: '#pricing', label: 'Pricing' },
  { href: '#faq', label: 'FAQ' },
]
