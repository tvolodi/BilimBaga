import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import '@/i18n'
import { AuditLogTable } from './AuditLogTable'
import type { AuditEntry } from '@/api/audit'

function makeEntry(n: number): AuditEntry {
  return {
    id: `entry-${n}-aaaaaaaa`,
    created_at: '2026-10-01T10:00:00Z',
    actor_id: n % 2 ? `actor-${n}` : null,
    actor_name: n % 2 ? `Actor ${n}` : null,
    action: `user.action_${n}`,
    entity_type: 'user',
    entity_id: `entity-${n}-bbbbbbbb`,
    ip_address: '127.0.0.1',
    metadata: { n },
  }
}

function setup(entries: AuditEntry[]) {
  return render(
    <MemoryRouter>
      <AuditLogTable entries={entries} />
    </MemoryRouter>,
  )
}

describe('AuditLogTable', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders many entries, including expanded rows, without React key warnings', () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const entries = [1, 2, 3, 4].map(makeEntry)
    setup(entries)

    // Expand two rows so both the main row and meta row render in the fragment.
    fireEvent.click(screen.getByText('user.action_1'))
    fireEvent.click(screen.getByText('user.action_3'))

    expect(screen.getAllByText('user.action_2')).toHaveLength(1)
    expect(document.querySelectorAll('pre')).toHaveLength(2)

    const keyWarnings = errorSpy.mock.calls.filter((args) =>
      args.some((a) => typeof a === 'string' && a.includes('unique "key" prop')),
    )
    expect(keyWarnings).toHaveLength(0)
    expect(errorSpy).not.toHaveBeenCalled()
  })

  it('renders the empty state with no entries', () => {
    setup([])
    expect(screen.getByText('—')).toBeInTheDocument()
  })
})
