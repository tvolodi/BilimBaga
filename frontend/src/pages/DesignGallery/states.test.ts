import { readdirSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { GALLERY_SECTIONS, type GalleryComponent } from './states'

// FR-BB320 AC-9: every component in components/ui is shown, with the states the FR lists.
// Each ui file maps to the gallery component that shows it.
const UI_TO_GALLERY: Record<string, GalleryComponent> = {
  button: 'button',
  badge: 'badge',
  input: 'input',
  select: 'select',
  label: 'label',
  card: 'card',
  table: 'table',
  skeleton: 'skeleton',
  dialog: 'dialog',
  sheet: 'sheet',
  popover: 'popover',
  'date-time-picker': 'date-time-picker',
}

function stateIds(component: GalleryComponent): string[] {
  return GALLERY_SECTIONS.find((s) => s.component === component)?.states.map((s) => s.id) ?? []
}

describe('design gallery states (FR-BB320 AC-9)', () => {
  it('has one section for every component in components/ui', () => {
    const uiDir = join(process.cwd(), 'src', 'components', 'ui')
    const primitives = readdirSync(uiDir)
      .filter((f) => f.endsWith('.tsx') && !f.endsWith('.test.tsx'))
      .map((f) => f.replace(/\.tsx$/, ''))
    const shown = new Set(GALLERY_SECTIONS.map((s) => s.component))

    for (const name of primitives) {
      expect(UI_TO_GALLERY[name], `${name}.tsx has no gallery mapping`).toBeDefined()
      expect(shown.has(UI_TO_GALLERY[name]), `${name} has no section`).toBe(true)
    }
  })

  it('shows every Button variant, every size and the disabled state', () => {
    const ids = stateIds('button')
    for (const variant of ['default', 'secondary', 'destructive', 'outline', 'ghost', 'link']) {
      expect(ids).toContain(`variant-${variant}`)
    }
    for (const size of ['sm', 'default', 'lg', 'icon']) {
      expect(ids).toContain(`size-${size}`)
    }
    expect(ids).toContain('disabled')
  })

  it('shows every Badge variant, including the semantic status pairs', () => {
    const ids = stateIds('badge')
    for (const variant of ['default', 'secondary', 'destructive', 'outline', 'success', 'warning', 'danger', 'info']) {
      expect(ids).toContain(`variant-${variant}`)
    }
  })

  it('shows Input, Select and Label in default, filled, disabled and aria-invalid states', () => {
    for (const component of ['input', 'select'] as const) {
      expect(stateIds(component)).toEqual(expect.arrayContaining(['default', 'filled', 'disabled', 'invalid']))
    }
    expect(stateIds('label')).toEqual(expect.arrayContaining(['default', 'disabled', 'invalid']))
  })

  it('shows the Card with header, content and footer, and the Table with rows and an empty row', () => {
    expect(stateIds('card')).toContain('header-content-footer')
    expect(stateIds('table')).toEqual(expect.arrayContaining(['rows', 'empty']))
  })

  it('shows the date-time picker in default and disabled states', () => {
    expect(stateIds('date-time-picker')).toEqual(expect.arrayContaining(['default', 'disabled']))
  })
})
