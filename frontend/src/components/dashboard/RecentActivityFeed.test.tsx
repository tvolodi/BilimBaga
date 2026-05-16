import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import '@/i18n'
import { RecentActivityFeed } from './RecentActivityFeed'
import type { RecentActivity } from '@/api/dashboard'

const passedActivity: RecentActivity = {
  session_id: 's-1',
  employee_name: 'Alice Smith',
  exam_title: 'Safety Exam',
  score_pct: 92.0,
  passed: true,
  submitted_at: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
}

const failedActivity: RecentActivity = {
  session_id: 's-2',
  employee_name: 'Bob Jones',
  exam_title: 'Loyalty Exam',
  score_pct: 45.0,
  passed: false,
  submitted_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
}

const pendingActivity: RecentActivity = {
  session_id: 's-3',
  employee_name: 'Carol White',
  exam_title: 'Security Exam',
  score_pct: null,
  passed: null,
  submitted_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
}

describe('RecentActivityFeed', () => {
  it('shows empty state when no activities', () => {
    render(<RecentActivityFeed activities={[]} />)
    expect(screen.getByText('No recent activity.')).toBeInTheDocument()
  })

  it('renders employee name and exam title', () => {
    render(<RecentActivityFeed activities={[passedActivity]} />)
    expect(screen.getByText('Alice Smith')).toBeInTheDocument()
    expect(screen.getByText('Safety Exam')).toBeInTheDocument()
  })

  it('renders score badge with percentage for passed activity', () => {
    render(<RecentActivityFeed activities={[passedActivity]} />)
    expect(screen.getByText('92%')).toBeInTheDocument()
  })

  it('renders score badge for failed activity', () => {
    render(<RecentActivityFeed activities={[failedActivity]} />)
    expect(screen.getByText('45%')).toBeInTheDocument()
  })

  it('renders dash badge for grading_pending activity', () => {
    render(<RecentActivityFeed activities={[pendingActivity]} />)
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('limits to 20 items', () => {
    const many = Array.from({ length: 25 }, (_, i) => ({
      ...passedActivity,
      session_id: `s-${i}`,
      employee_name: `User ${i}`,
    }))
    render(<RecentActivityFeed activities={many} />)
    expect(screen.getAllByText(/^User \d+$/).length).toBe(20)
  })
})
