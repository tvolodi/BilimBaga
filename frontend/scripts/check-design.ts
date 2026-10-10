// FR-BB320 Part B (AC-6 to AC-8): design-system enforcement for frontend/src. Rules: docs/design-system/README.md.
//   npm run check:design                         check the repository against scripts/design-baseline.json
//   npm run check:design -- --print-baseline     print the current counts in baseline format
//
// Hard rules (never baselined): a hex colour literal in quotes, and an arbitrary colour class (bg-[#...]).
// Baselined per file, and a count may only go down: a raw <button>, <input>, <select> or <textarea> outside
// components/ui, rgb( and hsl( colours, and default-palette colour classes (bg-white, text-gray-500, ...).
// Exceptions: `// design-ok: <reason>` on the line, or `/* design-ok-file: <reason> */` on the first line of a file.
// The reason is mandatory and printed. A marker that exempts nothing fails, so exceptions cannot go stale.
// Excluded: *.test.ts(x), index.css, pages/DesignGallery/. Comments are not scanned.

import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join, relative, sep } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

export type BaselinedKind = 'raw-element' | 'palette' | 'rgb-hsl'
export type Kind = BaselinedKind | 'hex' | 'arbitrary-colour'
export type Baseline = Partial<Record<BaselinedKind, Record<string, number>>>
export interface SourceFile {
  /** Relative to frontend/src, with forward slashes, for example `components/ui/button.tsx`. */
  path: string
  text: string
}
export interface Exception {
  path: string
  /** null for a file-level marker. */
  line: number | null
  reason: string
}
export interface CheckResult {
  ok: boolean
  failures: string[]
  exceptions: Exception[]
  /** Non-exempt hits per baselined kind and file. */
  counts: Record<BaselinedKind, Record<string, number>>
  baselineTotal: number
}

const BASELINED: BaselinedKind[] = ['raw-element', 'palette', 'rgb-hsl']

const COLOURS = 'gray|slate|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose'
const SHADES = '50|100|200|300|400|500|600|700|800|900|950'
// The same class rules as src/design-palette.test.ts (FR-BB321 AC-13).
const PALETTE = new RegExp(
  `(?<![\\w-])(?:[a-z-]+:)*(?:bg|text|border|ring|from|to|via|fill|stroke|outline|divide|placeholder|decoration|accent|caret|shadow|ring-offset)(?:-[xytblrse])?-(?:${COLOURS})-(?:${SHADES})(?:\\/\\d+)?(?![\\w-])` +
    `|(?<![\\w-])(?:[a-z-]+:)*(?:bg|text|border|ring)(?:-[xytblrse])?-(?:white|black)(?:\\/\\d+)?(?![\\w-])`,
  'g',
)
const HEX = /(['"`])#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})\1/g
const ARBITRARY = /(?<![\w-])(?:[a-z-]+:)*(?:bg|text|border|ring|fill|stroke|outline|from|to|via|shadow|placeholder|decoration|accent|caret|divide)-\[(?:#|rgba?\(|hsla?\()[^\]\s"'`]*\]?/g
const RGB_HSL = /\b(?:rgb|hsl)a?\(/g
const RAW_ELEMENT = /<(?:button|input|select|textarea)(?=[\s/>]|$)/g

interface Hit {
  line: number
  kind: Kind
  value: string
}

interface Marker {
  line: number
  reason: string
}

function isExcludedFile(path: string): boolean {
  return /\.test\.tsx?$/.test(path) || path === 'index.css' || path.startsWith('pages/DesignGallery/')
}

function matchesOf(pattern: RegExp, text: string): string[] {
  return [...text.matchAll(pattern)].map((match) => match[0])
}

/** The hits, markers and first-line file marker of one file. A marker with no reason is a problem. */
function scan(file: SourceFile): { hits: Hit[]; markers: Marker[]; problems: string[]; fileReason: string | null } {
  const lines = file.text.split(/\r?\n/)
  const hits: Hit[] = []
  const markers: Marker[] = []
  const problems: string[] = []
  const inComponentsUi = file.path.startsWith('components/ui/')

  const fileMarker = /^\s*\/\*\s*design-ok-file:(.*?)\s*\*\/\s*$/.exec(lines[0] ?? '')
  const fileReason = fileMarker ? fileMarker[1].trim() : null
  if (fileMarker && !fileReason) problems.push(`${file.path}:1 design-ok-file needs a reason`)

  lines.forEach((raw, index) => {
    const line = index + 1
    const marker = /\bdesign-ok:(.*)$/.exec(raw)
    if (marker) {
      const reason = marker[1].replace(/\*\/\s*$/, '').trim()
      if (!reason) problems.push(`${file.path}:${line} design-ok needs a reason`)
      else markers.push({ line, reason })
    }
    if (/^\s*(\/\/|\/\*|\*)/.test(raw)) return
    const code = raw.replace(/\/\*.*?\*\//g, '').replace(/(^|\s)\/\/.*$/, '')
    for (const match of matchesOf(HEX, code)) hits.push({ line, kind: 'hex', value: match.slice(1, -1) })
    for (const match of matchesOf(ARBITRARY, code)) hits.push({ line, kind: 'arbitrary-colour', value: match })
    for (const match of matchesOf(RGB_HSL, code)) hits.push({ line, kind: 'rgb-hsl', value: match })
    if (!inComponentsUi) for (const match of matchesOf(RAW_ELEMENT, code)) hits.push({ line, kind: 'raw-element', value: match })
    for (const match of matchesOf(PALETTE, code)) hits.push({ line, kind: 'palette', value: match })
  })

  return { hits, markers, problems, fileReason }
}

/** Applies the markers: returns the hits that still count, and the exceptions they grant. */
function applyExceptions(file: SourceFile): { counted: Hit[]; exceptions: Exception[]; problems: string[] } {
  const { hits, markers, problems, fileReason } = scan(file)
  if (fileReason) {
    if (hits.length === 0) problems.push(`${file.path}:1 design-ok-file marker exempts nothing: remove it`)
    const exceptions: Exception[] = hits.length ? [{ path: file.path, line: null, reason: fileReason }] : []
    return { counted: [], exceptions, problems }
  }
  const exceptions: Exception[] = []
  const counted: Hit[] = []
  const exemptLines = new Set<number>()
  for (const marker of markers) {
    const onLine = hits.filter((hit) => hit.line === marker.line)
    if (onLine.length === 0) {
      problems.push(`${file.path}:${marker.line} design-ok marker exempts nothing: remove it`)
      continue
    }
    exemptLines.add(marker.line)
    exceptions.push({ path: file.path, line: marker.line, reason: marker.reason })
  }
  for (const hit of hits) if (!exemptLines.has(hit.line)) counted.push(hit)
  return { counted, exceptions, problems }
}

/**
 * The check on in-memory files (FR-BB320 AC-8). `baseline` lists the allowed count per file for each baselined kind.
 * A count above its baseline fails, and so does a count below it: the baseline has to come down in the same change.
 */
export function checkDesign(files: SourceFile[], baseline: Baseline): CheckResult {
  const failures: string[] = []
  const exceptions: Exception[] = []
  const counts = { 'raw-element': {}, palette: {}, 'rgb-hsl': {} } as Record<BaselinedKind, Record<string, number>>
  const linesOf: Record<BaselinedKind, Record<string, number[]>> = { 'raw-element': {}, palette: {}, 'rgb-hsl': {} }

  for (const file of [...files].sort((a, b) => (a.path < b.path ? -1 : 1))) {
    if (isExcludedFile(file.path)) continue
    const { counted, exceptions: found, problems } = applyExceptions(file)
    failures.push(...problems)
    exceptions.push(...found)
    for (const hit of counted) {
      if (hit.kind === 'hex') {
        failures.push(`${file.path}:${hit.line} hex colour literal '${hit.value}': use a token class or mark // design-ok: <reason>`)
      } else if (hit.kind === 'arbitrary-colour') {
        failures.push(`${file.path}:${hit.line} arbitrary colour class '${hit.value}': use a token class or mark // design-ok: <reason>`)
      } else {
        const kind = hit.kind
        counts[kind][file.path] = (counts[kind][file.path] ?? 0) + 1
        linesOf[kind][file.path] = [...(linesOf[kind][file.path] ?? []), hit.line]
      }
    }
  }

  let baselineTotal = 0
  for (const kind of BASELINED) {
    const allowedByFile = baseline[kind] ?? {}
    baselineTotal += Object.values(allowedByFile).reduce((sum, n) => sum + n, 0)
    const paths = [...new Set([...Object.keys(counts[kind]), ...Object.keys(allowedByFile)])].sort()
    for (const path of paths) {
      const actual = counts[kind][path] ?? 0
      const allowed = allowedByFile[path] ?? 0
      if (actual > allowed) {
        const lines = linesOf[kind][path]?.join(', ')
        failures.push(`${path}: ${actual} ${kind} (baseline allows ${allowed}): new violation at lines ${lines}`)
      } else if (actual < allowed) {
        failures.push(`${path}: ${actual} ${kind} (baseline ${allowed}): baseline can be lowered to ${actual}`)
      }
    }
  }

  return { ok: failures.length === 0, failures, exceptions, counts, baselineTotal }
}

/** The baseline that matches the current counts exactly, for `--print-baseline`. Empty kinds are left out. */
export function baselineFrom(files: SourceFile[]): Baseline {
  const { counts } = checkDesign(files, {})
  const baseline: Baseline = {}
  for (const kind of BASELINED) {
    const entries = Object.keys(counts[kind]).length
    if (entries > 0) baseline[kind] = Object.fromEntries(Object.entries(counts[kind]).sort(([a], [b]) => (a < b ? -1 : 1)))
  }
  return baseline
}

function listSources(root: string, dir: string = root): SourceFile[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) return listSources(root, full)
    if (!/\.tsx?$/.test(entry.name)) return []
    return [{ path: relative(root, full).split(sep).join('/'), text: readFileSync(full, 'utf8') }]
  })
}

function parseBaseline(text: string): Baseline {
  const data: unknown = JSON.parse(text)
  if (data === null || typeof data !== 'object' || Array.isArray(data)) throw new Error('baseline must be a JSON object')
  for (const [kind, entries] of Object.entries(data)) {
    if (!(BASELINED as string[]).includes(kind)) throw new Error(`baseline has unknown kind "${kind}"`)
    if (entries === null || typeof entries !== 'object') throw new Error(`baseline "${kind}" must map file to count`)
    for (const [path, count] of Object.entries(entries)) {
      if (!Number.isInteger(count) || (count as number) < 0) throw new Error(`baseline "${kind}" ${path} must be a whole number`)
    }
  }
  return data as Baseline
}

function main(argv: string[]): number {
  const here = dirname(fileURLToPath(import.meta.url))
  const src = join(here, '..', 'src')
  const files = listSources(src)
  if (argv.includes('--print-baseline')) {
    console.log(JSON.stringify(baselineFrom(files), null, 2))
    return 0
  }
  let baseline: Baseline
  try {
    baseline = parseBaseline(readFileSync(join(here, 'design-baseline.json'), 'utf8'))
  } catch (error) {
    console.error(`check:design: cannot read scripts/design-baseline.json: ${(error as Error).message}`)
    return 2
  }
  const result = checkDesign(files, baseline)
  for (const exception of result.exceptions) {
    console.log(`design-ok ${exception.path}${exception.line === null ? '' : `:${exception.line}`}: ${exception.reason}`)
  }
  for (const failure of result.failures) console.error(failure)
  console.log(`check:design: ${result.exceptions.length} design-ok exceptions, baseline total ${result.baselineTotal}`)
  if (!result.ok) {
    console.error('check:design FAILED')
    return 1
  }
  console.log('check:design passed ✓')
  return 0
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  process.exitCode = main(process.argv.slice(2))
}
