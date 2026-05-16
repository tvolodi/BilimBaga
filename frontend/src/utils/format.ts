export function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}m ${s}s`
}

export const formatNumber = (
  value: number,
  locale: string,
  opts?: Intl.NumberFormatOptions,
): string => new Intl.NumberFormat(locale, opts).format(value)

export const formatPercent = (value: number, locale: string): string =>
  new Intl.NumberFormat(locale, {
    style: 'percent',
    maximumFractionDigits: 1,
  }).format(value / 100)

export const formatDate = (iso: string, locale: string): string =>
  new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(iso))
