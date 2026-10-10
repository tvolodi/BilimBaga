/** The gallery route (FR-BB320 AC-9). */
export const DESIGN_GALLERY_PATH = '/__design'

/**
 * FR-BB320 AC-9: the component gallery exists only in development, or when the build sets
 * VITE_ENABLE_DESIGN_GALLERY=true. Keep the expression on import.meta.env: Vite replaces those reads with
 * literals, so a production build folds this to false and drops the gallery chunk.
 */
export const DESIGN_GALLERY_ON: boolean =
  import.meta.env.DEV || import.meta.env.VITE_ENABLE_DESIGN_GALLERY === 'true'
