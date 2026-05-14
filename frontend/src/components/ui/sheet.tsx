import * as React from 'react'
import { cn } from '@/lib/utils'

interface SheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  children: React.ReactNode
}

function Sheet({ open, onOpenChange, children }: SheetProps) {
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50">
      <div
        className="fixed inset-0 bg-black/50"
        onClick={() => onOpenChange(false)}
        aria-hidden="true"
      />
      {children}
    </div>
  )
}

interface SheetContentProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode
  side?: 'left' | 'right'
}

function SheetContent({ className, children, side = 'right', ...props }: SheetContentProps) {
  return (
    <div
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

function SheetTitle({ className, ...props }: React.HTMLAttributes<HTMLHeadingElement>) {
  return <h2 className={cn('text-lg font-semibold leading-none tracking-tight', className)} {...props} />
}

function SheetFooter({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn('flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2 mt-6', className)} {...props} />
  )
}

export { Sheet, SheetContent, SheetHeader, SheetTitle, SheetFooter }
