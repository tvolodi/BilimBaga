import { describe, it, expect, vi } from 'vitest'
import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from './sheet'

function SheetHarness({ withDescription = true }: { withDescription?: boolean }) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <button onClick={() => setOpen(true)}>opener</button>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent>
          <SheetHeader>
            <SheetTitle>Create user</SheetTitle>
            {withDescription && <SheetDescription>Fill in the details below.</SheetDescription>}
          </SheetHeader>
          <input aria-label="Full name" />
          <SheetFooter>
            <button>Cancel</button>
            <button>Save</button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </>
  )
}

function overlay(): HTMLElement {
  const el = document.querySelector<HTMLElement>('div.fixed[aria-hidden="true"]')
  if (!el) throw new Error('sheet overlay not rendered')
  return el
}

describe('Sheet accessibility (FR-BB320 AC-4)', () => {
  it('renders a modal dialog labelled by SheetTitle and described by SheetDescription', async () => {
    render(<SheetHarness />)
    await userEvent.click(screen.getByText('opener'))
    const dialog = screen.getByRole('dialog', { name: 'Create user' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAccessibleDescription('Fill in the details below.')
    expect(dialog).toHaveAttribute('aria-labelledby', screen.getByRole('heading', { name: 'Create user' }).id)
    expect(dialog).toHaveAttribute('aria-describedby', screen.getByText('Fill in the details below.').id)
  })

  it('omits aria-describedby when there is no SheetDescription', async () => {
    render(<SheetHarness withDescription={false} />)
    await userEvent.click(screen.getByText('opener'))
    expect(screen.getByRole('dialog', { name: 'Create user' })).not.toHaveAttribute('aria-describedby')
  })

  it('moves initial focus into the panel on open', async () => {
    render(<SheetHarness />)
    await userEvent.click(screen.getByText('opener'))
    expect(screen.getByLabelText('Full name')).toHaveFocus()
  })

  it('focuses the panel itself when it has no focusable element', async () => {
    function Empty() {
      const [open, setOpen] = useState(false)
      return (
        <>
          <button onClick={() => setOpen(true)}>opener</button>
          <Sheet open={open} onOpenChange={setOpen}>
            <SheetContent>
              <SheetTitle>Details</SheetTitle>
            </SheetContent>
          </Sheet>
        </>
      )
    }
    render(<Empty />)
    await userEvent.click(screen.getByText('opener'))
    expect(screen.getByRole('dialog', { name: 'Details' })).toHaveFocus()
  })

  it('traps Tab inside the panel: Tab from the last control wraps to the first', async () => {
    render(<SheetHarness />)
    await userEvent.click(screen.getByText('opener'))
    expect(screen.getByLabelText('Full name')).toHaveFocus()
    await userEvent.tab()
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus()
    await userEvent.tab()
    expect(screen.getByRole('button', { name: 'Save' })).toHaveFocus()
    await userEvent.tab()
    expect(screen.getByLabelText('Full name')).toHaveFocus()
  })

  it('wraps Shift+Tab from the first control back to the last', async () => {
    render(<SheetHarness />)
    await userEvent.click(screen.getByText('opener'))
    await userEvent.tab({ shift: true })
    expect(screen.getByRole('button', { name: 'Save' })).toHaveFocus()
  })

  it('closes on Escape and returns focus to the opener', async () => {
    render(<SheetHarness />)
    const opener = screen.getByText('opener')
    await userEvent.click(opener)
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(opener).toHaveFocus()
  })

  it('calls onOpenChange(false) on Escape', async () => {
    const onOpenChange = vi.fn()
    render(
      <Sheet open onOpenChange={onOpenChange}>
        <SheetContent>
          <SheetTitle>Details</SheetTitle>
          <button>Close</button>
        </SheetContent>
      </Sheet>,
    )
    await userEvent.keyboard('{Escape}')
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('closes on overlay click and returns focus to the opener', async () => {
    render(<SheetHarness />)
    const opener = screen.getByText('opener')
    await userEvent.click(opener)
    await userEvent.click(overlay())
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(opener).toHaveFocus()
  })

  it('does not close when a click lands inside the panel', async () => {
    render(<SheetHarness />)
    await userEvent.click(screen.getByText('opener'))
    await userEvent.click(screen.getByLabelText('Full name'))
    expect(screen.getByRole('dialog', { name: 'Create user' })).toBeInTheDocument()
  })
})
