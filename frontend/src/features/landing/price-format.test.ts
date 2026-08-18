import { describe, expect, it } from 'vitest'
import { formatLandingPrice, humanizeUnit, unitSuffix } from './price-format'

describe('formatLandingPrice', () => {
  it('omits decimals for whole numbers', () => {
    expect(formatLandingPrice(99, 'INR')).toBe('₹99')
    expect(formatLandingPrice(1299, 'INR')).toBe('₹1,299')
  })

  it('keeps two decimals for fractional prices', () => {
    expect(formatLandingPrice(99.5, 'INR')).toBe('₹99.50')
  })

  it('falls back to a dash for missing values', () => {
    expect(formatLandingPrice(null, 'INR')).toBe('—')
    expect(formatLandingPrice(undefined, 'INR')).toBe('—')
  })
})

describe('unitSuffix', () => {
  it('maps known units to short suffixes', () => {
    expect(unitSuffix('kg')).toBe('/kg')
    expect(unitSuffix('per_item')).toBe('/item')
    expect(unitSuffix('per_week')).toBe('/week')
    expect(unitSuffix('flat')).toBe('')
  })

  it('falls back for unknown units', () => {
    expect(unitSuffix('per_load')).toBe('/load')
  })
})

describe('humanizeUnit', () => {
  it('maps known units to readable labels', () => {
    expect(humanizeUnit('per_week')).toBe('per week')
    expect(humanizeUnit('per_month')).toBe('per month')
    expect(humanizeUnit('kg')).toBe('per kg')
  })

  it('falls back for missing values', () => {
    expect(humanizeUnit('')).toBe('')
  })
})
