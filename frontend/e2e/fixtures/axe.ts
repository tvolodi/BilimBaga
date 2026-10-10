/**
 * FR-BB320 AC-5: axe helper for colour-contrast checks.
 *
 * Runs only the `color-contrast` rule on the current page and returns one readable line per
 * failing node, so a failing assertion names the element and its ratio instead of a raw result.
 */

import AxeBuilder from '@axe-core/playwright'
import type { Page } from '@playwright/test'

export async function colorContrastViolations(page: Page): Promise<string[]> {
  const results = await new AxeBuilder({ page }).withRules(['color-contrast']).analyze()
  return results.violations.flatMap((violation) =>
    violation.nodes.map(
      (node) => `${violation.id}: ${node.target.join(' ')} — ${node.failureSummary ?? violation.help}`,
    ),
  )
}
