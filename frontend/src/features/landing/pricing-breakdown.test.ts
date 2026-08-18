import { describe, expect, it } from 'vitest'
import { buildIncludedLines, perUnitHint } from './pricing-breakdown'
import type { PricingComboView, PricingServiceView } from '@/services/pricing.service'

const services: PricingServiceView[] = [
  {
    id: 's1',
    name: 'Wash & Fold',
    code: 'WASH_FOLD',
    description: '',
    unit: 'kg',
    price: 99.5,
    currency: 'INR',
    tax_rate: 5,
    estimated_days: 2,
    is_active: true,
    icon: '',
  },
  {
    id: 's2',
    name: 'Dry Cleaning',
    code: 'DRY',
    description: '',
    unit: 'per_item',
    price: 149,
    currency: 'INR',
    tax_rate: 0,
    estimated_days: 2,
    is_active: true,
    icon: '👔',
  },
]

const combo: PricingComboView = {
  id: 'c1',
  name: 'Weekly Pass',
  code: 'WEEKLY',
  description: '',
  unit: 'per_week',
  price: 499,
  currency: 'INR',
  is_active: true,
  is_featured: true,
  validity_days: 7,
  renewal_type: 'auto',
  validity_text: 'Valid for 1 week · Renews automatically',
  regular_price: 398,
  savings: 0,
  items: [{ service_type_id: 's1', quantity: 4 }],
}

describe('buildIncludedLines', () => {
  it('prefers the backend item snapshot when present', () => {
    const lines = buildIncludedLines(
      {
        ...combo,
        items: [
          {
            service_type_id: 's1',
            quantity: 4,
            service_name: 'Wash & Fold',
            unit: 'kg',
            price: 99.5,
            currency: 'INR',
          },
        ],
      },
      services,
    )
    expect(lines).toHaveLength(1)
    expect(lines[0].label).toBe('Wash & Fold')
    expect(lines[0].unit).toBe('kg')
    expect(lines[0].lineValue).toBe(398)
    expect(lines[0].currency).toBe('INR')
  })

  it('falls back to the services catalog join when the snapshot is missing', () => {
    const lines = buildIncludedLines(combo, services)
    expect(lines[0].label).toBe('Wash & Fold')
    expect(lines[0].unit).toBe('kg')
    expect(lines[0].lineValue).toBe(398)
  })

  it('treats empty snapshot fields as missing (stale cache) and falls back to services', () => {
    // A catalog snapshot cached by an older server version: the keys exist
    // but are empty/zero. The card must not render blank labels or ₹0.
    const lines = buildIncludedLines(
      {
        ...combo,
        items: [
          {
            service_type_id: 's1',
            quantity: 4,
            service_name: '',
            unit: '',
            price: 0,
            currency: '',
          },
        ],
      },
      services,
    )
    expect(lines[0].label).toBe('Wash & Fold')
    expect(lines[0].unit).toBe('kg')
    expect(lines[0].lineValue).toBe(398)
    expect(lines[0].currency).toBe('INR')
  })

  it('omits lineValue when no price is known anywhere', () => {
    const lines = buildIncludedLines({ ...combo, items: [{ service_type_id: 's1', quantity: 2 }] }, [])
    expect(lines[0].label).toBe('Service')
    expect(lines[0].lineValue).toBeUndefined()
    expect(lines[0].currency).toBe('INR') // combo currency fallback
  })

describe('perUnitHint', () => {
  it('computes the per-load price for a wash pass (499 / 4 ≈ 125)', () => {
    const hint = perUnitHint({
      ...combo,
      items: [{ service_type_id: 's1', quantity: 4, unit: 'kg' }],
    })
    expect(hint).toEqual({ label: 'load', value: 125 })
  })

  it('labels homogeneous per-item bundles as items', () => {
    const hint = perUnitHint({
      ...combo,
      price: 300,
      items: [
        { service_type_id: 's1', quantity: 1, unit: 'per_item' },
        { service_type_id: 's2', quantity: 1, unit: 'per_item' },
      ],
    })
    expect(hint).toEqual({ label: 'item', value: 150 })
  })

  it('is undefined for mixed units (never mixes loads and items)', () => {
    const hint = perUnitHint({
      ...combo,
      items: [
        { service_type_id: 's1', quantity: 12, unit: 'kg' },
        { service_type_id: 's2', quantity: 1, unit: 'per_item' },
      ],
    })
    expect(hint).toBeUndefined()
  })

  it('is undefined for empty items, missing units or zero price', () => {
    expect(perUnitHint({ ...combo, items: [] })).toBeUndefined()
    expect(perUnitHint({ ...combo, items: [{ service_type_id: 's1', quantity: 4 }] })).toBeUndefined()
    expect(perUnitHint({ ...combo, price: 0, items: [{ service_type_id: 's1', quantity: 4, unit: 'kg' }] })).toBeUndefined()
  })
})

  it('handles multiple items and empty item lists', () => {
    const lines = buildIncludedLines(
      {
        ...combo,
        items: [
          { service_type_id: 's1', quantity: 12 },
          { service_type_id: 's2', quantity: 1 },
        ],
      },
      services,
    )
    expect(lines).toHaveLength(2)
    expect(lines[0].lineValue).toBe(1194) // 12 × 99.5
    expect(lines[1].lineValue).toBe(149) // 1 × 149
    expect(buildIncludedLines({ ...combo, items: [] }, services)).toEqual([])
  })
})
