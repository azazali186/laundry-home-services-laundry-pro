import { patchRequest, postRequest } from '@/lib/api'

/** Shop-wide configuration as returned by POST /settings/view. */
export interface ShopSettings {
  id: string
  business_name: string
  business_phone: string
  business_email: string
  address: string
  city: string
  currency: string
  tax_rate: number
  tax_id: string
  invoice_prefix: string
  receipt_footer: string
  timezone: string
  created_at: string
  updated_at: string
}

/** Partial patch — every field optional, mirroring the backend pointers. */
export type ShopSettingsUpdate = Partial<
  Omit<ShopSettings, 'id' | 'created_at' | 'updated_at'>
>

export async function fetchShopSettings(): Promise<ShopSettings> {
  return postRequest<object, ShopSettings>('/settings/view', {})
}

export async function updateShopSettings(req: ShopSettingsUpdate): Promise<ShopSettings> {
  return patchRequest<ShopSettingsUpdate, ShopSettings>('/settings', req)
}

/** Common ISO-4217 currencies for the shop configuration select. */
export const CURRENCY_OPTIONS = [
  { value: 'INR', label: 'INR — Indian Rupee (₹)' },
  { value: 'USD', label: 'USD — US Dollar ($)' },
  { value: 'EUR', label: 'EUR — Euro (€)' },
  { value: 'GBP', label: 'GBP — British Pound (£)' },
  { value: 'AED', label: 'AED — UAE Dirham (د.إ)' },
  { value: 'SGD', label: 'SGD — Singapore Dollar (S$)' },
  { value: 'CAD', label: 'CAD — Canadian Dollar (C$)' },
  { value: 'AUD', label: 'AUD — Australian Dollar (A$)' },
  { value: 'JPY', label: 'JPY — Japanese Yen (¥)' },
]

/** Common timezones for the shop configuration select. */
export const TIMEZONE_OPTIONS = [
  { value: 'Asia/Kolkata', label: 'Asia/Kolkata (IST, UTC+5:30)' },
  { value: 'Asia/Dubai', label: 'Asia/Dubai (GST, UTC+4)' },
  { value: 'Asia/Singapore', label: 'Asia/Singapore (SGT, UTC+8)' },
  { value: 'Asia/Kuala_Lumpur', label: 'Asia/Kuala_Lumpur (UTC+8)' },
  { value: 'Europe/London', label: 'Europe/London (GMT/BST)' },
  { value: 'Europe/Berlin', label: 'Europe/Berlin (CET/CEST)' },
  { value: 'America/New_York', label: 'America/New_York (ET)' },
  { value: 'America/Chicago', label: 'America/Chicago (CT)' },
  { value: 'America/Los_Angeles', label: 'America/Los_Angeles (PT)' },
  { value: 'UTC', label: 'UTC' },
]
