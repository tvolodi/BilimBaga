// FR-BB24 AC-11: a non-default locale that is "present" (stem or any option text
// non-blank) must have non-blank text for every option, otherwise the API
// answers 422. Returns the locales that violate this so the editor can block the
// save with a clear message instead of sending a stem-only locale.
export function findPartialLocales(
  locales: readonly string[],
  defaultLocale: string,
  stems: Record<string, { stem?: string } | undefined>,
  optionBodies: Record<string, { body?: string } | undefined>[],
): string[] {
  if (optionBodies.length === 0) return []
  return locales.filter((loc) => {
    if (loc === defaultLocale) return false
    const blank = (s?: string) => !s || s.trim() === ''
    const texts = optionBodies.map((o) => o[loc]?.body)
    const present = !blank(stems[loc]?.stem) || texts.some((t) => !blank(t))
    return present && texts.some((t) => blank(t))
  })
}
