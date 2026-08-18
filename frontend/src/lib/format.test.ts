import { describe, expect, it } from 'vitest'
import { formatCurrency, formatNumber, humanize, shortId, showingSummary } from './format'

describe('formatCurrency', () => {
  it('formats with the given currency', () => {
    expect(formatCurrency(99, 'INR')).toBe('₹99.00')
  })

  it('falls back to a dash for missing values', () => {
    expect(formatCurrency(undefined, 'USD')).toBe('—')
    expect(formatCurrency(null, 'USD')).toBe('—')
  })
})

describe('formatNumber', () => {
  it('adds thousands separators', () => {
    expect(formatNumber(245)).toBe('245')
    expect(formatNumber(120000)).toBe('120,000')
  })

  it('falls back to a dash for invalid input', () => {
    expect(formatNumber(Number.NaN)).toBe('—')
  })
})

describe('showingSummary', () => {
  it('describes the visible range', () => {
    expect(showingSummary(1, 10, 120)).toBe('Showing 11–20 of 120')
  })

  it('handles empty datasets', () => {
    expect(showingSummary(0, 10, 0)).toBe('Showing 0 results')
  })
})

describe('humanize', () => {
  it('title-cases snake_case codes', () => {
    expect(humanize('per_item')).toBe('Per Item')
    expect(humanize('pending_pickup')).toBe('Pending Pickup')
  })

  it('falls back for missing values', () => {
    expect(humanize(null)).toBe('—')
  })
})

describe('shortId', () => {
  it('returns an 8-char prefix', () => {
    expect(shortId('019fec99-1984-7cbc-b920-a669934e8c0b')).toBe('019fec99')
  })

  it('falls back for missing values', () => {
    expect(shortId('')).toBe('—')
  })
})
