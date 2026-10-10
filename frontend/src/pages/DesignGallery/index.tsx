/* eslint-disable i18next/no-literal-string -- FR-BB320 AC-9: developer-only gallery. Its labels stay out of the locale files, which ship to production. */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { Badge, type BadgeProps } from '@/components/ui/badge'
import { Button, type ButtonProps } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { DateTimePicker } from '@/components/ui/date-time-picker'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Select } from '@/components/ui/select'
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { applyLocale, SUPPORTED_LOCALES } from '@/lib/locale'
import { cn } from '@/lib/utils'
import { GALLERY_SECTIONS, type GallerySection, type GalleryState } from './states'

// FR-BB320 AC-9: the development-only component gallery. The route is registered only when
// DESIGN_GALLERY_ON is true (see App.tsx), so this module is not in a production build.

function Sample({ section, state }: { section: GallerySection; state: GalleryState }) {
  const p = state.props ?? {}
  const id = `${section.component}-${state.id}`
  const formProps = {
    defaultValue: p.defaultValue as string | undefined,
    disabled: p.disabled === true,
    'aria-invalid': p['aria-invalid'] === true ? true : undefined,
  }

  switch (section.component) {
    case 'button':
      return p.size === 'icon' ? (
        <Button
          variant={p.variant as ButtonProps['variant']}
          size="icon"
          aria-label="Add"
          disabled={p.disabled === true}
        >
          <Plus size={16} aria-hidden="true" />
        </Button>
      ) : (
        <Button
          variant={p.variant as ButtonProps['variant']}
          size={p.size as ButtonProps['size']}
          disabled={p.disabled === true}
        >
          Button
        </Button>
      )

    case 'badge':
      return <Badge variant={p.variant as BadgeProps['variant']}>Badge</Badge>

    case 'input':
      return (
        <div className="max-w-xs space-y-1">
          <Label htmlFor={id}>Label</Label>
          <Input id={id} {...formProps} />
        </div>
      )

    case 'select':
      return (
        <div className="max-w-xs space-y-1">
          <Label htmlFor={id}>Label</Label>
          <Select id={id} {...formProps}>
            <option>Option A</option>
            <option>Option B</option>
          </Select>
        </div>
      )

    case 'label':
      return (
        <div className="max-w-xs space-y-1">
          <Label htmlFor={id}>Label</Label>
          <Input id={id} {...formProps} />
        </div>
      )

    case 'card':
      return (
        <Card className="max-w-sm">
          <CardHeader>
            <CardTitle>Card title</CardTitle>
          </CardHeader>
          <CardContent>Card content</CardContent>
          {/* Card has no footer primitive; the footer is styled with the same tokens. */}
          <div className="border-t p-4 text-sm text-muted-foreground">Card footer</div>
        </Card>
      )

    case 'table':
      return (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {state.id === 'empty' ? (
              <TableRow>
                <TableCell colSpan={2} className="text-center text-muted-foreground">
                  No rows
                </TableCell>
              </TableRow>
            ) : (
              <>
                <TableRow>
                  <TableCell>Sample one</TableCell>
                  <TableCell>Active</TableCell>
                </TableRow>
                <TableRow>
                  <TableCell>Sample two</TableCell>
                  <TableCell>Draft</TableCell>
                </TableRow>
              </>
            )}
          </TableBody>
        </Table>
      )

    case 'skeleton':
      return (
        <div className="max-w-sm space-y-2">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-4 w-64" />
        </div>
      )

    case 'dialog':
      return <DialogSample />

    case 'sheet':
      return <SheetSample />

    case 'popover':
      return <PopoverSample />

    case 'date-time-picker':
      return <DateTimeSample disabled={p.disabled === true} />
  }
}

// Overlays: the instance opens on demand. A fixed overlay on mount would cover the gallery.
function DialogSample() {
  const [open, setOpen] = useState(false)
  return (
    <>
      <Button variant="outline" onClick={() => setOpen(true)}>
        Open
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Dialog title</DialogTitle>
            <DialogDescription>Dialog description</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

function SheetSample() {
  const [open, setOpen] = useState(false)
  return (
    <>
      <Button variant="outline" onClick={() => setOpen(true)}>
        Open
      </Button>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent side="right">
          <SheetHeader>
            <SheetTitle>Sheet title</SheetTitle>
            <SheetDescription>Sheet description</SheetDescription>
          </SheetHeader>
          <SheetFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>
              Close
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </>
  )
}

// Popover opens on mount, so the gallery shows an open instance.
function PopoverSample() {
  return (
    <Popover defaultOpen>
      <PopoverTrigger asChild>
        <Button variant="outline">Popover</Button>
      </PopoverTrigger>
      <PopoverContent>
        <p className="text-sm">Popover content</p>
      </PopoverContent>
    </Popover>
  )
}

function DateTimeSample({ disabled }: { disabled: boolean }) {
  const [value, setValue] = useState<string | null>(null)
  return (
    <DateTimePicker
      id={`date-time-${disabled ? 'disabled' : 'default'}`}
      label="Date and time"
      value={value}
      onChange={setValue}
      disabled={disabled}
    />
  )
}

function SectionView({ section }: { section: GallerySection }) {
  const [dark, setDark] = useState(false)
  const headingId = `gallery-heading-${section.component}`

  return (
    <section data-testid={`gallery-section-${section.component}`} aria-labelledby={headingId} className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 id={headingId} className="text-base font-semibold">
          {section.title}
        </h2>
        <div role="group" aria-label="Theme" className="flex gap-1">
          <Button size="sm" variant={dark ? 'outline' : 'default'} aria-pressed={!dark} onClick={() => setDark(false)}>
            Light
          </Button>
          <Button size="sm" variant={dark ? 'default' : 'outline'} aria-pressed={dark} onClick={() => setDark(true)}>
            Dark
          </Button>
        </div>
      </div>
      {/* The .dark class on this container switches every token inside it to the dark theme. */}
      <div
        data-testid={`gallery-states-${section.component}`}
        className={cn('space-y-4 rounded-lg border bg-background p-4 text-foreground', dark && 'dark')}
      >
        {section.states.map((state) => (
          <div key={state.id} className="space-y-2">
            <p className="text-xs text-muted-foreground">{state.label}</p>
            <Sample section={section} state={state} />
          </div>
        ))}
      </div>
    </section>
  )
}

export function DesignGalleryPage() {
  const { i18n } = useTranslation()

  return (
    <main
      id="main-content"
      data-testid="design-gallery"
      className="min-h-screen space-y-8 bg-background p-6 text-foreground"
    >
      <header className="flex flex-wrap items-center justify-between gap-4">
        <h1 className="text-xl font-semibold">Design gallery</h1>
        <div role="group" aria-label="Language" className="flex gap-1">
          {SUPPORTED_LOCALES.map((locale) => {
            const current = i18n.language === locale.code
            return (
              <Button
                key={locale.code}
                size="sm"
                variant={current ? 'default' : 'outline'}
                aria-pressed={current}
                onClick={() => applyLocale(i18n, locale.code)}
              >
                {locale.label}
              </Button>
            )
          })}
        </div>
      </header>
      {GALLERY_SECTIONS.map((section) => (
        <SectionView key={section.component} section={section} />
      ))}
    </main>
  )
}
