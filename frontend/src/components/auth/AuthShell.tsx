import type { ReactNode } from 'react'
import { ThemeToggle } from '@/components/ThemeToggle'

/**
 * FR-BB321 AC-6: the public auth pages have no shared layout, so the theme switch sits in a fixed
 * top-end corner above the page markup, which renders unchanged inside.
 */
export function AuthShell({ children }: { children: ReactNode }) {
  return (
    <>
      <div className="fixed top-4 end-4 z-10">
        <ThemeToggle />
      </div>
      {children}
    </>
  )
}
