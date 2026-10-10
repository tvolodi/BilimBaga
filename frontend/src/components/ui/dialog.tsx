import * as React from 'react'
import { cn } from '@/lib/utils'

const FOCUSABLE_SELECTOR =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

interface DialogContextValue {
  /** Id the Dialog owns for DialogTitle; DialogContent points aria-labelledby at it once a title is mounted. */
  titleId: string
  descriptionId: string
  hasTitle: boolean
  hasDescription: boolean
  setHasTitle: (present: boolean) => void
  setHasDescription: (present: boolean) => void
}

const DialogContext = React.createContext<DialogContextValue | null>(null)

interface DialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  children: React.ReactNode
}

function Dialog({ open, onOpenChange, children }: DialogProps) {
  const containerRef = React.useRef<HTMLDivElement>(null)
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
  const context = React.useMemo<DialogContextValue>(
    () => ({ titleId, descriptionId, hasTitle, hasDescription, setHasTitle, setHasDescription }),
    [titleId, descriptionId, hasTitle, hasDescription],
  )

  React.useEffect(() => {
    if (!open) return
    // Initial focus: move into the dialog unless a child already took it (autoFocus or ref focus).
    const container = containerRef.current
    if (container && !container.contains(document.activeElement)) {
      const target =
        container.querySelector<HTMLElement>(FOCUSABLE_SELECTOR) ??
        container.querySelector<HTMLElement>('[role="dialog"]')
      target?.focus()
    }
    // Remember the trigger so focus returns to it on close (a11y).
    function handleKeyDown(e: KeyboardEvent) {
      // A nested widget (for example a Radix popover) that already handled the key owns it.
      if (e.defaultPrevented) return
      if (e.key === 'Escape') {
        onOpenChangeRef.current(false)
        return
      }
      if (e.key !== 'Tab') return
      // Focus trap: keep Tab / Shift+Tab inside the dialog.
      const focusable = containerRef.current?.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)
      if (!focusable || focusable.length === 0) return
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      const active = document.activeElement
      if (e.shiftKey && (active === first || !containerRef.current?.contains(active))) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && (active === last || !containerRef.current?.contains(active))) {
        e.preventDefault()
        first.focus()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      openerRef.current?.focus?.()
      openerRef.current = null
    }
  }, [open])

  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        className="fixed inset-0 bg-black/50"
        onClick={() => onOpenChange(false)}
        aria-hidden="true"
      />
      <DialogContext.Provider value={context}>
        <div ref={containerRef} className="relative z-50">{children}</div>
      </DialogContext.Provider>
    </div>
  )
}

interface DialogContentProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode
}

function DialogContent({ className, children, ...props }: DialogContentProps) {
  const ctx = React.useContext(DialogContext)
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={ctx?.hasTitle ? ctx.titleId : undefined}
      aria-describedby={ctx?.hasDescription ? ctx.descriptionId : undefined}
      tabIndex={-1}
      className={cn(
        'bg-background rounded-lg shadow-lg p-6 w-full max-w-md mx-4',
        className,
      )}
      {...props}
    >
      {children}
    </div>
  )
}

function DialogHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('flex flex-col space-y-1.5 mb-4', className)} {...props} />
}

// The Dialog owns the title and description ids; each registers its presence so DialogContent
// only references ids that exist in the DOM.
function DialogTitle({ className, id, ...props }: React.HTMLAttributes<HTMLHeadingElement>) {
  const ctx = React.useContext(DialogContext)
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

function DialogDescription({ className, id, ...props }: React.HTMLAttributes<HTMLParagraphElement>) {
  const ctx = React.useContext(DialogContext)
  const setHasDescription = ctx?.setHasDescription
  React.useEffect(() => {
    setHasDescription?.(true)
    return () => setHasDescription?.(false)
  }, [setHasDescription])
  return (
    <p className={cn('text-sm text-muted-foreground', className)} {...props} id={ctx ? ctx.descriptionId : id} />
  )
}

function DialogFooter({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn('flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2 mt-4', className)} {...props} />
  )
}

export { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter }
