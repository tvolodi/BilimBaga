import { describe, it, expect } from 'vitest'
import { findPartialLocales } from './partialLocales'

const opts = (kk1: string, kk2: string) => [{ kk: { body: kk1 } }, { kk: { body: kk2 } }]

describe('findPartialLocales', () => {
  it('flags a stem-only locale with blank options', () => {
    expect(findPartialLocales(['en', 'kk'], 'en', { kk: { stem: 'Сұрақ' } }, opts('', ''))).toEqual(['kk'])
  })
  it('flags a locale with only some options filled', () => {
    expect(findPartialLocales(['en', 'kk'], 'en', {}, opts('A', ' '))).toEqual(['kk'])
  })
  it('accepts a fully translated locale and an untranslated one', () => {
    expect(findPartialLocales(['en', 'kk'], 'en', { kk: { stem: 'S' } }, opts('A', 'B'))).toEqual([])
    expect(findPartialLocales(['en', 'kk'], 'en', { kk: { stem: '' } }, opts('', ''))).toEqual([])
  })
  it('ignores the default locale and questions without options', () => {
    expect(findPartialLocales(['en'], 'en', { en: { stem: 'S' } }, [{ en: { body: '' } }])).toEqual([])
    expect(findPartialLocales(['en', 'kk'], 'en', { kk: { stem: 'S' } }, [])).toEqual([])
  })
})
