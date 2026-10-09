import { describe, it, expect } from 'vitest'
import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Dialog, DialogContent } from './dialog'

function Harness() {
  const [open, setOpen] = useState(false)
  return (
    <>
      <button onClick={() => setOpen(true)}>opener</button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <input aria-label="first" ref={(el) => el?.focus()} />
          <button>last</button>
        </DialogContent>
      </Dialog>
    </>
  )
}

describe('Dialog focus handling', () => {
  it('traps Tab and returns focus to the opener even with an autoFocus child', async () => {
    render(<Harness />)
    const opener = screen.getByText('opener')
    await userEvent.click(opener)
    expect(screen.getByLabelText('first')).toHaveFocus()
    await userEvent.tab()
    expect(screen.getByText('last')).toHaveFocus()
    await userEvent.tab()
    expect(screen.getByLabelText('first')).toHaveFocus()
    await userEvent.tab({ shift: true })
    expect(screen.getByText('last')).toHaveFocus()
    await userEvent.keyboard('{Escape}')
    expect(opener).toHaveFocus()
  })
})
