import { describe, it, expect } from 'vitest'
import { formatDistanceToNow, intervalLabel, formatDuration } from '../utils/time'

describe('formatDistanceToNow', () => {
  it('returns seconds for recent times', () => {
    const now = new Date(Date.now() - 10_000).toISOString()
    expect(formatDistanceToNow(now)).toMatch(/\d+s ago/)
  })

  it('returns minutes for minute-range', () => {
    const recent = new Date(Date.now() - 90_000).toISOString()
    expect(formatDistanceToNow(recent)).toBe('1m ago')
  })

  it('returns hours for hour-range', () => {
    const hourAgo = new Date(Date.now() - 3_700_000).toISOString()
    expect(formatDistanceToNow(hourAgo)).toBe('1h ago')
  })

  it('returns days for day-range', () => {
    const dayAgo = new Date(Date.now() - 86_500_000).toISOString()
    expect(formatDistanceToNow(dayAgo)).toBe('1d ago')
  })
})

describe('intervalLabel', () => {
  it('returns seconds for sub-minute', () => {
    expect(intervalLabel(30)).toBe('30s')
  })

  it('returns minutes for minute range', () => {
    expect(intervalLabel(300)).toBe('5m')
  })

  it('returns hours for hour range', () => {
    expect(intervalLabel(3600)).toBe('1h')
  })
})

describe('formatDuration', () => {
  it('returns ms for sub-second', () => {
    expect(formatDuration(200)).toBe('200ms')
  })

  it('returns seconds for >= 1000ms', () => {
    expect(formatDuration(1500)).toBe('1.5s')
  })
})
