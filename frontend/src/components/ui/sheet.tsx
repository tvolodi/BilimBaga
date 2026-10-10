import * as React from 'react'
import { cn } from '@/lib/utils'

const FOCUSABLE_SELECTOR =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

interface SheetContextValue {
  /** The panel (SheetContent) element: the focus trap and initial focus work inside it. */
  contentRef: React.RefObject<HTMLDivElement | null>
  /** Ids the Sheet owns for SheetTitle and SheetDescription; the panel references them once mounted. */
  titleId: string
  descriptionId: string
  hasTitle: boolean
  hasDescription: boolean
  setHasTitle: (present: boolean) => void
  setHasDescription: (present: boolean) => void
}

const SheetContext = React.createContext<SheetContextValue | null>(null)

interface SheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  children: React.ReactNode
}

function Sheet({ open, onOpenChange, children }: SheetProps) {
  const contentRef = React.useRef<HTMLDivElement>(null)
  // Latest callback in a ref so re-renders do not re-run the effect (which would steal focus).
  const onOpenChangeRef = React.useRef(onOpenChange)
  onOpenChangeRef.current = onOpenChange
  // Capture the opener during render, before children mount (an autoFocus child would otherwise win).
  const openerRef = React.useRef<HTMLElement | null>(null)
  if (open && !openerRef.current) openerRef.current = document.activeElement as HTMLElement | null

  const titleId = React.useId()
  const descriptionId = React.useId()
  const [hasTitle, setHasTitle] = React.useState(false)
  const [hasDescription, setHasDescription] = React.useState(false)
  const context = React.useMemo<SheetContextValue>(
    () => ({
      contentRef,
      titleId,
      descriptionId,
      hasTitle,
      hasDescription,
      setHasTitle,
      setHasDescription,
    }),
    [titleId, descriptionId, hasTitle, hasDescription],
  )

  React.useEffect(() => {
    if (!open) return
    // Initial focus: move into the panel unless a child already took it (autoFocus or ref focus).
    const panel = contentRef.current
    if (panel && !panel.contains(document.activeElement)) {
      (panel.querySelector<HTMLElement>(FOCUSABLE_SELECTOR) ?? panel).focus()
    }
    function handleKeyDown(e: KeyboardEvent) {
      // A nested widget (for example a Radix popover) that already handled the key owns it.
      if (e.defaultPrevented) return
      if (e.key === 'Escape') {
        onOpenChangeRef.current(false)
        return
      }
      if (e.key !== 'Tab') return
      // Focus trap: keep Tab / Shift+Tab inside the panel.
      const current = contentRef.current
      const focusable = current?.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)
      if (!current || !focusable || focusable.length === 0) return
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      const active = document.activeElement
      const inside = current.contains(active)
      if (e.shiftKey && (active === first || !inside)) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && (active === last || !inside)) {
        e.preventDefault()
        first.focus()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      // Return focus to the trigger on close (a11y).
      openerRef.current?.focus?.()
      openerRef.current = null
    }
  }, [open])

  if (!open) return null
  return (
    <SheetContext.Provider value={context}>
      <div className="fixed inset-0 z-50">
        <div
          className="fixed inset-0 bg-black/50"
          onClick={() => onOpenChange(false)}
          aria-hidden="true"
        />
        {children}
      </div>
    </SheetContext.Provider>
  )
}

interface SheetContentProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode
  side?: 'left' | 'right'
}

function SheetContent({ className, children, side = 'right', ...props }: SheetContentProps) {
  const ctx = React.useContext(SheetContext)
  return (
    <div
      ref={ctx?.contentRef}
      role="dialog"
      aria-modal="true"
      aria-labelledby={ctx?.hasTitle ? ctx.titleId : undefined}
      aria-describedby={ctx?.hasDescription ? ctx.descriptionId : undefined}
      tabIndex={-1}
      className={cn(
        'fixed z-50 bg-background shadow-xl p-6 overflow-y-auto',
        side === 'right' ? 'right-0 top-0 h-full w-full max-w-md' : 'left-0 top-0 h-full w-full max-w-md',
        className,
      )}
      {...props}
    >
      {children}
    </div>
  )
}

function SheetHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('flex flex-col space-y-1.5 mb-6', className)} {...props} />
}

// The Sheet owns the title and description ids; each registers its presence so the panel
// only references ids that exist in the DOM.
function SheetTitle({ className, id, ...props }: React.HTMLAttributes<HTMLHeadingElement>) {
  const ctx = React.useContext(SheetContext)
  const setHasTitle = ctx?.setHasTitle
  React.useEffect(() => {
    setHasTitle?.(true)
    return () => setHasTitle?.(false)
  }, [setHasTitle])
  return (
    <h2
      className={cn('text-lg font-semibold leading-none tracking-tight', className)}
      {...props}
      id={ctx ? ctx.titleId : id}
    />
  )
}

function SheetDescription({ className, id, ...props }: React.HTMLAttributes<HTMLParagraphElement>) {
  const ctx = React.useContext(SheetContext)
  const setHasDescription = ctx?.setHasDescription
  React.useEffect(() => {
    setHasDescription?.(true)
    return () => setHasDescription?.(false)
  }, [setHasDescription])
  return (
    <p className={cn('text-sm text-muted-foreground', className)} {...props} id={ctx ? ctx.descriptionId : id} />
  )
}

function SheetFooter({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn('flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2 mt-6', className)} {...props} />
  )
}

export { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetFooter }
