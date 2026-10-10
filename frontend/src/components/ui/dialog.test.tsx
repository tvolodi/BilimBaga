import { describe, it, expect, vi } from 'vitest'
import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from './dialog'

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

function LabelledHarness({ withDescription = true }: { withDescription?: boolean }) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <button onClick={() => setOpen(true)}>opener</button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete user</DialogTitle>
            {withDescription && <DialogDescription>This cannot be undone.</DialogDescription>}
          </DialogHeader>
          <button>Cancel</button>
          <button>Confirm</button>
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

describe('Dialog accessibility (FR-BB320 AC-4)', () => {
  it('labels the dialog with its DialogTitle and describes it with its DialogDescription', async () => {
    render(<LabelledHarness />)
    await userEvent.click(screen.getByText('opener'))
    const dialog = screen.getByRole('dialog', { name: 'Delete user' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAccessibleDescription('This cannot be undone.')
    const titleId = screen.getByRole('heading', { name: 'Delete user' }).id
    const descriptionId = screen.getByText('This cannot be undone.').id
    expect(titleId).not.toBe('')
    expect(dialog).toHaveAttribute('aria-labelledby', titleId)
    expect(dialog).toHaveAttribute('aria-describedby', descriptionId)
  })

  it('omits aria-describedby when there is no DialogDescription', async () => {
    render(<LabelledHarness withDescription={false} />)
    await userEvent.click(screen.getByText('opener'))
    const dialog = screen.getByRole('dialog', { name: 'Delete user' })
    expect(dialog).not.toHaveAttribute('aria-describedby')
  })

  it('moves initial focus to the first focusable element on open', async () => {
    render(<LabelledHarness />)
    await userEvent.click(screen.getByText('opener'))
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus()
  })

  it('focuses the dialog container when it has no focusable element', async () => {
    function Empty() {
      const [open, setOpen] = useState(false)
      return (
        <>
          <button onClick={() => setOpen(true)}>opener</button>
          <Dialog open={open} onOpenChange={setOpen}>
            <DialogContent>
              <DialogTitle>Notice</DialogTitle>
            </DialogContent>
          </Dialog>
        </>
      )
    }
    render(<Empty />)
    await userEvent.click(screen.getByText('opener'))
    expect(screen.getByRole('dialog', { name: 'Notice' })).toHaveFocus()
  })

  it('wraps Tab from the last control back to the first', async () => {
    render(<LabelledHarness />)
    await userEvent.click(screen.getByText('opener'))
    await userEvent.tab()
    expect(screen.getByRole('button', { name: 'Confirm' })).toHaveFocus()
    await userEvent.tab()
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus()
  })

  it('closes on Escape and returns focus to the opener', async () => {
    render(<LabelledHarness />)
    const opener = screen.getByText('opener')
    await userEvent.click(opener)
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(opener).toHaveFocus()
  })

  it('calls onOpenChange(false) on Escape', async () => {
    const onOpenChange = vi.fn()
    render(
      <Dialog open onOpenChange={onOpenChange}>
        <DialogContent>
          <DialogTitle>Notice</DialogTitle>
          <button>Ok</button>
        </DialogContent>
      </Dialog>,
    )
    await userEvent.keyboard('{Escape}')
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('keeps an explicit aria-labelledby passed by the caller', () => {
    render(
      <Dialog open onOpenChange={() => {}}>
        <DialogContent aria-labelledby="custom-label">
          <span id="custom-label">Custom label</span>
          <DialogTitle>Title</DialogTitle>
          <button>Ok</button>
        </DialogContent>
      </Dialog>,
    )
    expect(screen.getByRole('dialog', { name: 'Custom label' })).toBeInTheDocument()
  })
})
