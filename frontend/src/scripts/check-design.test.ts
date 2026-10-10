import { describe, expect, it } from 'vitest'
import { baselineFrom, checkDesign, type Baseline, type SourceFile } from '../../scripts/check-design'

// FR-BB320 AC-8: the check's rules on in-memory files. `npm run check:design` runs the same rules on the repository.

const file = (path: string, ...lines: string[]): SourceFile => ({ path, text: lines.join('\n') })
const none: Baseline = {}

describe('check:design colour rules (FR-BB320 AC-6)', () => {
  it('fails a hex colour literal, and a baseline cannot excuse it', () => {
    const result = checkDesign([file('pages/A.tsx', "const c = '#ff0000'")], none)
    expect(result.ok).toBe(false)
    expect(result.failures.join('\n')).toMatch(/pages\/A\.tsx:1 hex colour literal '#ff0000'/)
  })

  it('fails an arbitrary Tailwind colour class', () => {
    const result = checkDesign([file('pages/A.tsx', '<div className="bg-[#123456]" />')], none)
    expect(result.ok).toBe(false)
    expect(result.failures.join('\n')).toMatch(/pages\/A\.tsx:1 arbitrary colour class/)
  })

  it('counts rgb( and hsl( against the baseline, not as a hard failure', () => {
    const source = file('pages/A.tsx', 'const c = rgb(0 0 0)')
    expect(checkDesign([source], none).ok).toBe(false)
    expect(checkDesign([source], { 'rgb-hsl': { 'pages/A.tsx': 1 } }).ok).toBe(true)
  })

  it('counts a default-palette class per occurrence', () => {
    const source = file('pages/A.tsx', '<p className="text-gray-500 bg-white" />')
    const result = checkDesign([source], { palette: { 'pages/A.tsx': 2 } })
    expect(result.ok).toBe(true)
    expect(result.counts.palette).toEqual({ 'pages/A.tsx': 2 })
  })

  it('applies the palette and colour rules inside components/ui, but not the raw-element rule', () => {
    expect(checkDesign([file('components/ui/x.tsx', '<button>x</button>')], none).ok).toBe(true)
    const palette = checkDesign([file('components/ui/x.tsx', 'className="bg-white"')], none)
    expect(palette.ok).toBe(false)
    expect(palette.failures.join('\n')).toMatch(/components\/ui\/x\.tsx: 1 palette/)
  })
})

describe('check:design raw elements and scope (FR-BB320 AC-6)', () => {
  it('counts raw button, input, select and textarea elements, including a tag that breaks across lines', () => {
    const source = file('pages/A.tsx', '<button>x</button>', '<input', '  value="" />', '<select><textarea /></select>')
    expect(checkDesign([source], none).counts['raw-element']).toEqual({ 'pages/A.tsx': 4 })
  })

  it('does not count element names that only start with those words', () => {
    const source = file('pages/A.tsx', '<Button>x</Button>', '<selectable />', '<buttonGroup />')
    expect(checkDesign([source], none).ok).toBe(true)
  })

  it('does not scan comments', () => {
    const source = file('pages/A.tsx', '// use <button> here', '{/* the inner <input> is the control */}', '/* <select> */')
    expect(checkDesign([source], none).ok).toBe(true)
  })

  it('skips test files, the design gallery and index.css', () => {
    const result = checkDesign(
      [
        file('pages/A.test.tsx', "<button /> const c = '#fff'"),
        file('pages/DesignGallery/Page.tsx', '<button /> className="bg-white"'),
        file('index.css', '--color-x: #fff;'),
      ],
      none,
    )
    expect(result.ok).toBe(true)
  })
})

describe('check:design exceptions (FR-BB320 AC-6)', () => {
  it('lets a design-ok line marker exempt its hit and reports the reason', () => {
    const result = checkDesign([file('pages/B.tsx', "const c = '#000000' // design-ok: fallback for an unparseable draft")], none)
    expect(result.ok).toBe(true)
    expect(result.exceptions).toEqual([{ path: 'pages/B.tsx', line: 1, reason: 'fallback for an unparseable draft' }])
  })

  it('exempts a baselined kind on a marked line too, so the count does not grow', () => {
    const source = file('pages/B.tsx', '<button /> // design-ok: legacy control, removed with FR-BB999')
    const result = checkDesign([source], none)
    expect(result.ok).toBe(true)
    expect(result.counts['raw-element']).toEqual({})
  })

  it('fails a design-ok marker with no reason', () => {
    const result = checkDesign([file('pages/B.tsx', "const c = '#000000' // design-ok:")], none)
    expect(result.ok).toBe(false)
    expect(result.failures.join('\n')).toMatch(/pages\/B\.tsx:1 design-ok needs a reason/)
  })

  it('lets a first-line design-ok-file marker exempt every hit in the file', () => {
    const result = checkDesign(
      [file('pages/C.tsx', '/* design-ok-file: tenant colour editor */', '<button />', "const c = '#fff'")],
      none,
    )
    expect(result.ok).toBe(true)
    expect(result.exceptions).toEqual([{ path: 'pages/C.tsx', line: null, reason: 'tenant colour editor' }])
  })

  it('fails a design-ok marker that exempts nothing, so exceptions cannot go stale', () => {
    const result = checkDesign([file('pages/S.tsx', 'const n = 1 // design-ok: nothing here')], none)
    expect(result.ok).toBe(false)
    expect(result.failures.join('\n')).toMatch(/pages\/S\.tsx:1 design-ok marker exempts nothing/)
  })
})

describe('check:design baseline ratchet (FR-BB320 AC-7)', () => {
  const two = file('pages/D.tsx', '<button />', '<button />')

  it('passes when the count equals the baseline', () => {
    expect(checkDesign([two], { 'raw-element': { 'pages/D.tsx': 2 } }).ok).toBe(true)
  })

  it('fails when the count exceeds the baseline', () => {
    const result = checkDesign([two], { 'raw-element': { 'pages/D.tsx': 1 } })
    expect(result.ok).toBe(false)
    expect(result.failures.join('\n')).toMatch(/pages\/D\.tsx: 2 raw-element \(baseline allows 1\): new violation/)
  })

  it('fails a new file with a violation that has no baseline entry', () => {
    const result = checkDesign([two], { 'raw-element': { 'pages/Other.tsx': 1 } })
    expect(result.ok).toBe(false)
    expect(result.failures.join('\n')).toMatch(/pages\/D\.tsx: 2 raw-element \(baseline allows 0\): new violation/)
  })

  it('fails when the baseline is larger than the count, so a fixed violation lowers the baseline', () => {
    const result = checkDesign([two], { 'raw-element': { 'pages/D.tsx': 3 } })
    expect(result.ok).toBe(false)
    expect(result.failures.join('\n')).toMatch(/pages\/D\.tsx: 2 raw-element \(baseline 3\): baseline can be lowered to 2/)
  })

  it('fails a baseline entry for a file that no longer has the violation', () => {
    const result = checkDesign([file('pages/D.tsx', 'const n = 1')], { 'raw-element': { 'pages/D.tsx': 1 } })
    expect(result.ok).toBe(false)
    expect(result.failures.join('\n')).toMatch(/pages\/D\.tsx: 0 raw-element \(baseline 1\): baseline can be lowered to 0/)
  })

  it('reports the baseline total across kinds', () => {
    const source = file('pages/D.tsx', '<button />', 'rgb(0 0 0)', 'text-gray-500')
    const result = checkDesign([source], { 'raw-element': { 'pages/D.tsx': 1 }, 'rgb-hsl': { 'pages/D.tsx': 1 }, palette: { 'pages/D.tsx': 1 } })
    expect(result.ok).toBe(true)
    expect(result.baselineTotal).toBe(3)
  })

  it('baselineFrom writes exactly the current counts, so the ratchet passes on them', () => {
    const files = [two, file('pages/E.tsx', 'text-gray-500', "const c = '#fff'")]
    const baseline = baselineFrom(files)
    expect(baseline).toEqual({ 'raw-element': { 'pages/D.tsx': 2 }, palette: { 'pages/E.tsx': 1 } })
    expect(checkDesign(files, baseline).failures).toEqual(["pages/E.tsx:2 hex colour literal '#fff': use a token class or mark // design-ok: <reason>"])
  })
})
