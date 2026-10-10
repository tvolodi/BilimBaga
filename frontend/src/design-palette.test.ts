import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, sep } from 'node:path'
import { describe, expect, it } from 'vitest'

// FR-BB321 AC-13 guard: components use the semantic tokens (text-danger, bg-bg-success, text-muted-foreground, ...).
// Tailwind's default palette is light-only, so a palette class on a dark ground can be unreadable (for example
// text-red-600 is 2.86:1 on the dark grounds). Test files and the allow-list below are exempt.

const SRC = join(process.cwd(), 'src')

const COLOURS = 'gray|slate|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose'
const SHADES = '50|100|200|300|400|500|600|700|800|900|950'
const PALETTE_CLASS = new RegExp(
  `(?<![\\w-])((?:[a-z-]+:)*(?:bg|text|border|ring|from|to|via|fill|stroke|outline|divide|placeholder|decoration|accent|caret|shadow|ring-offset)(?:-[xytblrse])?-(?:${COLOURS})-(?:${SHADES})(?:\\/\\d+)?)(?![\\w-])` +
    `|(?<![\\w-])((?:[a-z-]+:)*(?:bg|text|border|ring)(?:-[xytblrse])?-(?:white|black)(?:\\/\\d+)?)(?![\\w-])`,
  'g',
)

// Each entry is a class that is allowed in one file, with the reason it is not a status colour.
const ALLOWED: Array<{ file: string; token: string; reason: string }> = [
  { file: 'components/TenantLogo.tsx', token: 'bg-white', reason: 'AC-9: the white plate behind a logo in dark theme' },
  { file: 'components/ui/dialog.tsx', token: 'bg-black/50', reason: 'overlay backdrop' },
  { file: 'components/ui/sheet.tsx', token: 'bg-black/50', reason: 'overlay backdrop' },
  { file: 'pages/ExamTaking/ExamLayout.tsx', token: 'bg-black/50', reason: 'overlay backdrop' },
]

function componentFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) return componentFiles(full)
    if (!entry.name.endsWith('.tsx') || entry.name.endsWith('.test.tsx') || full.includes(`${sep}__tests__${sep}`)) return []
    return [full]
  })
}

function paletteHits(): Array<{ file: string; line: number; token: string }> {
  return componentFiles(SRC).flatMap((full) => {
    const file = relative(SRC, full).split(sep).join('/')
    return readFileSync(full, 'utf8').split(/\r?\n/).flatMap((text, index) =>
      [...text.matchAll(PALETTE_CLASS)].map((match) => ({
        file,
        line: index + 1,
        token: (match[1] ?? match[2]).replace(/^(?:[a-z-]+:)+/, ''),
      })),
    )
  })
}

describe('design-system palette guard (FR-BB321 AC-13)', () => {
  it('no non-test component uses a default-palette colour class outside the allow-list', () => {
    const offenders = paletteHits()
      .filter((hit) => !ALLOWED.some((entry) => entry.file === hit.file && entry.token === hit.token))
      .map((hit) => `${hit.file}:${hit.line} ${hit.token}`)
    expect(offenders, 'use the status or surface tokens instead (see docs/design-system/README.md)').toEqual([])
  })

  it('every allow-list entry still matches a class, so the list cannot go stale', () => {
    const hits = paletteHits()
    const stale = ALLOWED.filter((entry) => !hits.some((hit) => hit.file === entry.file && hit.token === entry.token))
    expect(stale.map((entry) => `${entry.file} ${entry.token} (${entry.reason})`)).toEqual([])
  })
})
