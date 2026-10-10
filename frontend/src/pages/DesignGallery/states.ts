// FR-BB320 AC-9: the data behind the development-only design gallery. Each section is one component in
// components/ui; each state is one way to show it. The renderer in index.tsx reads these lists.
// The labels are developer identifiers. They stay out of the locale files, which ship to production.

export type GalleryComponent =
  | 'button'
  | 'badge'
  | 'input'
  | 'select'
  | 'label'
  | 'card'
  | 'table'
  | 'skeleton'
  | 'dialog'
  | 'sheet'
  | 'popover'
  | 'date-time-picker'

export interface GalleryState {
  /** Unique within its section. */
  id: string
  /** The state as a developer reads it. */
  label: string
  /** Props the renderer gives the primitive for this state. */
  props?: Record<string, string | boolean>
}

export interface GallerySection {
  component: GalleryComponent
  title: string
  states: GalleryState[]
}

const BUTTON_VARIANTS = ['default', 'secondary', 'destructive', 'outline', 'ghost', 'link'] as const
const BUTTON_SIZES = ['sm', 'default', 'lg', 'icon'] as const
const BADGE_VARIANTS = [
  'default',
  'secondary',
  'destructive',
  'outline',
  'success',
  'warning',
  'danger',
  'info',
] as const

// The form controls share one set of states.
const FORM_STATES: GalleryState[] = [
  { id: 'default', label: 'default' },
  { id: 'filled', label: 'filled', props: { defaultValue: 'Sample value' } },
  { id: 'disabled', label: 'disabled', props: { disabled: true } },
  { id: 'invalid', label: 'aria-invalid', props: { 'aria-invalid': true } },
]

export const GALLERY_SECTIONS: GallerySection[] = [
  {
    component: 'button',
    title: 'Button',
    states: [
      ...BUTTON_VARIANTS.map((variant) => ({ id: `variant-${variant}`, label: variant, props: { variant } })),
      ...BUTTON_SIZES.map((size) => ({ id: `size-${size}`, label: size, props: { size } })),
      { id: 'disabled', label: 'disabled', props: { disabled: true } },
    ],
  },
  {
    component: 'badge',
    title: 'Badge',
    states: BADGE_VARIANTS.map((variant) => ({ id: `variant-${variant}`, label: variant, props: { variant } })),
  },
  { component: 'input', title: 'Input', states: FORM_STATES },
  { component: 'select', title: 'Select', states: FORM_STATES },
  { component: 'label', title: 'Label', states: FORM_STATES },
  {
    component: 'card',
    title: 'Card',
    states: [{ id: 'header-content-footer', label: 'header, content, footer' }],
  },
  {
    component: 'table',
    title: 'Table',
    states: [
      { id: 'rows', label: 'with rows' },
      { id: 'empty', label: 'empty row' },
    ],
  },
  { component: 'skeleton', title: 'Skeleton', states: [{ id: 'default', label: 'default' }] },
  // Dialog and Sheet are overlays: their instance opens on demand, not on mount.
  { component: 'dialog', title: 'Dialog', states: [{ id: 'open', label: 'open' }] },
  { component: 'sheet', title: 'Sheet', states: [{ id: 'open', label: 'open' }] },
  // Popover opens on mount, so the gallery shows an open instance.
  { component: 'popover', title: 'Popover', states: [{ id: 'open', label: 'open' }] },
  {
    component: 'date-time-picker',
    title: 'DateTimePicker',
    states: [
      { id: 'default', label: 'default' },
      { id: 'disabled', label: 'disabled', props: { disabled: true } },
    ],
  },
]
