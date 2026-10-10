import { useEffect, useState } from 'react'

/** Tailwind's `sm` breakpoint (40rem, 640px). */
const SM_UP = '(min-width: 40rem)'

function readSmUp(): boolean {
  // Without matchMedia (jsdom, some embedded views) the layout is treated as wide, as it was before.
  return typeof window.matchMedia === 'function' ? window.matchMedia(SM_UP).matches : true
}

/** True at the `sm` breakpoint and up, following the viewport while mounted (FR-BB321 AC-6, #472). */
export function useSmUp(): boolean {
  const [smUp, setSmUp] = useState(readSmUp)

  useEffect(() => {
    if (typeof window.matchMedia !== 'function') return
    const query = window.matchMedia(SM_UP)
    const update = () => setSmUp(query.matches)
    query.addEventListener('change', update)
    return () => query.removeEventListener('change', update)
  }, [])

  return smUp
}
