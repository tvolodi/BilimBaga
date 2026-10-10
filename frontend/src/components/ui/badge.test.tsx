import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Badge, badgeVariants } from './badge'

// FR-BB320 AC-3: status variants use the semantic pairs (text colour on its own background).
const STATUS_PAIRS = [
  ['success', ['text-success', 'bg-bg-success']],
  ['warning', ['text-warning', 'bg-bg-warning']],
  ['danger', ['text-danger', 'bg-bg-danger']],
  ['info', ['text-info', 'bg-bg-info']],
] as const

describe('Badge status variants (FR-BB320 AC-3)', () => {
  it.each(STATUS_PAIRS)('%s renders the semantic token pair', (variant, classes) => {
    render(<Badge variant={variant}>status</Badge>)
    const badge = screen.getByText('status')
    for (const cls of classes) {
      expect(badge).toHaveClass(cls)
    }
  })

  it.each(STATUS_PAIRS)('%s does not use Tailwind default-palette colours', (variant) => {
    const classes = badgeVariants({ variant })
    expect(classes).not.toMatch(/-(green|yellow|red|blue|amber|emerald|orange|gray|slate)-\d/)
  })

  it('keeps the default and destructive variants on the shadcn token aliases', () => {
    render(<Badge variant="destructive">gone</Badge>)
    expect(screen.getByText('gone')).toHaveClass('bg-destructive', 'text-destructive-foreground')
  })
})
