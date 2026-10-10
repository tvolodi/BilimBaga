import { afterEach, describe, expect, it, vi } from 'vitest'

// FR-BB320 AC-9: the component gallery at /__design is development-only. The flag is read from
// import.meta.env inside the module, so these cases re-import it under each environment.
async function gallery() {
  vi.resetModules()
  return import('./designGallery')
}

afterEach(() => {
  vi.unstubAllEnvs()
})

describe('design gallery route flag (FR-BB320 AC-9)', () => {
  it('is on in the development build', async () => {
    const { DESIGN_GALLERY_ON, DESIGN_GALLERY_PATH } = await gallery()
    expect(DESIGN_GALLERY_ON).toBe(true)
    expect(DESIGN_GALLERY_PATH).toBe('/__design')
  })

  it('is off in a production build without the opt-in flag, so the production route table has no gallery', async () => {
    vi.stubEnv('DEV', false)
    vi.stubEnv('VITE_ENABLE_DESIGN_GALLERY', '')
    const { DESIGN_GALLERY_ON } = await gallery()
    expect(DESIGN_GALLERY_ON).toBe(false)
  })

  it('is on in a production build only when VITE_ENABLE_DESIGN_GALLERY is exactly "true"', async () => {
    vi.stubEnv('DEV', false)
    vi.stubEnv('VITE_ENABLE_DESIGN_GALLERY', 'true')
    expect((await gallery()).DESIGN_GALLERY_ON).toBe(true)

    vi.stubEnv('VITE_ENABLE_DESIGN_GALLERY', 'yes')
    expect((await gallery()).DESIGN_GALLERY_ON).toBe(false)
  })
})
