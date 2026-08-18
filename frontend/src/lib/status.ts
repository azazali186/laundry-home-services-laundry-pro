import type { BadgeTone } from '@/components/ui/badge'

/** Map domain status strings to badge tones (unknown → slate). */
const TONE_BY_STATUS: Record<string, BadgeTone> = {
  // bookings
  pending: 'amber',
  confirmed: 'blue',
  cancelled: 'red',
  completed: 'emerald',
  // orders
  processing: 'blue',
  cleaned: 'cyan',
  ironed: 'cyan',
  ready: 'emerald',
  delivered: 'emerald',
  // pickups / deliveries
  scheduled: 'amber',
  in_transit: 'blue',
  active: 'blue',
  // payments / invoices
  paid: 'emerald',
  draft: 'slate',
  issued: 'blue',
  void: 'red',
  failed: 'red',
  refunded: 'violet',
  // garments
  received: 'amber',
  damaged: 'red',
  // notifications
  sent: 'emerald',
  unread: 'blue',
  // users / auth
  success: 'emerald',
  login_failed: 'red',
  // generic booleans
  true: 'emerald',
  false: 'slate',
  yes: 'emerald',
  no: 'slate',
}

export function statusTone(status: string): BadgeTone {
  return TONE_BY_STATUS[status.toLowerCase()] ?? 'slate'
}
