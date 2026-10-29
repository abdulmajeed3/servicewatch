import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StatusBadge } from '../components/ui/StatusBadge'
import type { MonitorStatus } from '../types'

describe('StatusBadge', () => {
  const cases: Array<[MonitorStatus, string]> = [
    ['up', 'Up'],
    ['down', 'Down'],
    ['pending', 'Pending'],
    ['paused', 'Paused'],
  ]

  it.each(cases)('renders status=%s with label=%s', (status, label) => {
    render(<StatusBadge status={status} />)
    expect(screen.getByText(label)).toBeInTheDocument()
  })

  it('applies correct color class for up', () => {
    const { container } = render(<StatusBadge status="up" />)
    expect(container.firstChild).toHaveClass('bg-green-100')
  })

  it('applies correct color class for down', () => {
    const { container } = render(<StatusBadge status="down" />)
    expect(container.firstChild).toHaveClass('bg-red-100')
  })
})
