import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Users } from 'lucide-react'
import { KpiCard } from './KpiCard'

describe('KpiCard', () => {
  it('renders label and value', () => {
    render(<KpiCard icon={<Users />} label="Total Employees" value={42} />)
    expect(screen.getByText('Total Employees')).toBeInTheDocument()
    expect(screen.getByText('42')).toBeInTheDocument()
  })

  it('renders description when provided', () => {
    render(
      <KpiCard
        icon={<Users />}
        label="Completion Rate"
        value="78.5%"
        description="Last 30 days"
      />,
    )
    expect(screen.getByText('Last 30 days')).toBeInTheDocument()
  })

  it('does not render description element when omitted', () => {
    render(<KpiCard icon={<Users />} label="Pass Rate" value="60.0%" />)
    expect(screen.queryByText('Last 30 days')).not.toBeInTheDocument()
  })
})
