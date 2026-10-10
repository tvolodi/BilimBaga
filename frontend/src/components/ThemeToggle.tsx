import { useRef, type KeyboardEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Monitor, Moon, Sun, type LucideIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useTheme, type ThemePreference } from '@/components/ThemeProvider'

interface ThemeOption {
  value: ThemePreference
  labelKey: 'theme.light' | 'theme.system' | 'theme.dark'
  Icon: LucideIcon
}

const OPTIONS: readonly ThemeOption[] = [
  { value: 'light', labelKey: 'theme.light', Icon: Sun },
  { value: 'system', labelKey: 'theme.system', Icon: Monitor },
  { value: 'dark', labelKey: 'theme.dark', Icon: Moon },
]

/**
 * FR-BB321 AC-5: three-way theme switch (Light, System, Dark) as a radio group.
 * Arrow keys move and select (WAI-ARIA radio group); roving tabindex keeps one tab stop.
 */
export function ThemeToggle() {
  const { t } = useTranslation()
  const { preference, setPreference, switchEnabled } = useTheme()
  const buttons = useRef<Partial<Record<ThemePreference, HTMLButtonElement | null>>>({})

  function handleKeyDown(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    let next: number
    switch (event.key) {
      case 'ArrowRight':
      case 'ArrowDown':
        next = (index + 1) % OPTIONS.length
        break
      case 'ArrowLeft':
      case 'ArrowUp':
        next = (index - 1 + OPTIONS.length) % OPTIONS.length
        break
      case 'Home':
        next = 0
        break
      case 'End':
        next = OPTIONS.length - 1
        break
      default:
        return
    }
    event.preventDefault()
    const option = OPTIONS[next]
    setPreference(option.value)
    buttons.current[option.value]?.focus()
  }

  // While the switch is off (THEME_SWITCH_ENABLED) the page is light and the control is hidden.
  if (!switchEnabled) return null

  return (
    <div
      role="radiogroup"
      aria-label={t('theme.label')}
      className="inline-flex items-center gap-0.5 rounded-md border border-border p-0.5"
    >
      {OPTIONS.map((option, index) => {
        const checked = preference === option.value
        const { Icon } = option
        return (
          <Button
            key={option.value}
            ref={(element) => {
              buttons.current[option.value] = element
            }}
            type="button"
            role="radio"
            aria-checked={checked}
            tabIndex={checked ? 0 : -1}
            variant={checked ? 'secondary' : 'ghost'}
            size="sm"
            onClick={() => setPreference(option.value)}
            onKeyDown={(event) => handleKeyDown(event, index)}
            className="max-sm:min-h-11 max-sm:min-w-11 px-2"
          >
            <Icon aria-hidden="true" size={16} />
            <span className="sr-only">{t(option.labelKey)}</span>
          </Button>
        )
      })}
    </div>
  )
}
