import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, sep } from 'node:path'
import { describe, expect, it } from 'vitest'

// #530: the sweep found no open-work markers in frontend source. Unfinished work is an issue, not a comment, so this test
// keeps the count at zero. The words are split in the pattern so this file does not match itself.
const OPEN_WORK_MARKER = new RegExp(`\\b(T${'ODO'}|FIX${'ME'})\\b`)

const SRC = join(process.cwd(), 'src')

function sourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) return sourceFiles(full)
    return /\.(ts|tsx)$/.test(entry.name) ? [full] : []
  })
}

describe('open-work marker guard (#530)', () => {
  it('no frontend source file carries an open-work marker', () => {
    const files = sourceFiles(SRC)
    expect(files.length, 'the guard must scan real sources').toBeGreaterThan(0)
    const hits = files.flatMap((full) =>
      readFileSync(full, 'utf8')
        .split(/\r?\n/)
        .flatMap((text, index) =>
          OPEN_WORK_MARKER.test(text) ? [`${relative(SRC, full).split(sep).join('/')}:${index + 1}`] : [],
        ),
    )
    expect(hits, 'remove the marker and file an issue for the work it names').toEqual([])
  })
})
