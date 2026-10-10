import { describe, expect, it } from 'vitest'
import { tenantOverrides } from './tenantColours'

describe('tenantOverrides in the light theme', () => {
  it('applies a primary that reaches 4.5:1 on white, with its foreground', () => {
    const overrides = tenantOverrides({ primary_color: '#123456', accent_color: '#c8a84b' }, 'light')
    expect(overrides).toEqual({
      primary: '#123456',
      primaryForeground: '#ffffff',
      accent: '#c8a84b',
    })
  })

  it('ignores a primary below 4.5:1 on white and keeps the accent', () => {
    const overrides = tenantOverrides({ primary_color: '#0ea5e9', accent_color: '#c8a84b' }, 'light')
    expect(overrides.primary).toBeNull()
    expect(overrides.primaryForeground).toBeNull()
    expect(overrides.accent).toBe('#c8a84b')
  })

  it('ignores a value that is not hex', () => {
    const overrides = tenantOverrides({ primary_color: 'red} body { display: none' }, 'light')
    expect(overrides.primary).toBeNull()
  })

  it('removes every override when there is no primary', () => {
    expect(tenantOverrides(undefined, 'light')).toEqual({ primary: null, primaryForeground: null, accent: null })
    expect(tenantOverrides({ primary_color: '', accent_color: '#c8a84b' }, 'light')).toEqual({
      primary: null,
      primaryForeground: null,
      accent: null,
    })
  })

  it('drops an accent that is not hex and keeps the primary', () => {
    const overrides = tenantOverrides({ primary_color: '#123456', accent_color: 'gold' }, 'light')
    expect(overrides.primary).toBe('#123456')
    expect(overrides.accent).toBeNull()
  })
})

describe('tenantOverrides in the dark theme', () => {
  it('derives the dark primary from a tenant colour that is not the default', () => {
    const overrides = tenantOverrides({ primary_color: '#1b3a6b' }, 'dark')
    expect(overrides.primary).toBe('#6c97da')
    expect(overrides.primaryForeground).toBe('#0f1623')
  })

  it('leaves the dark pair in force for the design default primary', () => {
    const overrides = tenantOverrides({ primary_color: '#2E6DB4' }, 'dark')
    expect(overrides.primary).toBeNull()
    expect(overrides.primaryForeground).toBeNull()
  })

  it('leaves the dark pair in force for a primary that was ignored', () => {
    const overrides = tenantOverrides({ primary_color: '#0ea5e9' }, 'dark')
    expect(overrides.primary).toBeNull()
  })

  it('derives the accent, and leaves the token for the design default accent', () => {
    expect(tenantOverrides({ primary_color: '#1b3a6b', accent_color: '#B91C1C' }, 'dark').accent).toBe('#e96d6d')
    expect(tenantOverrides({ primary_color: '#1b3a6b', accent_color: '#c8a84b' }, 'dark').accent).toBeNull()
  })

  it('removes every override when there is no primary', () => {
    expect(tenantOverrides({ accent_color: '#B91C1C' }, 'dark')).toEqual({
      primary: null,
      primaryForeground: null,
      accent: null,
    })
  })
})
