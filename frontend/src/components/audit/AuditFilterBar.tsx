import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Select } from '@/components/ui/select'
import { AUDIT_ACTIONS, type AuditFilters } from '@/api/audit'

interface AuditFilterBarProps {
  filters: AuditFilters
  onChange: (f: AuditFilters) => void
}

// Known entity types derived from AUDIT_ACTIONS prefixes.
const ENTITY_TYPES = ['user', 'exam', 'session', 'answer', 'certificate', 'question', 'tenant']

export function AuditFilterBar({ filters, onChange }: AuditFilterBarProps) {
  const { t } = useTranslation()

  // Debounced actor name input — fires onChange 300ms after the user stops typing.
  const [actorInput, setActorInput] = useState(filters.actor ?? '')
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Keep actorInput in sync when filters are cleared externally.
  useEffect(() => {
    setActorInput(filters.actor ?? '')
  }, [filters.actor])

  function handleActorChange(value: string) {
    setActorInput(value)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      onChange({ ...filters, actor: value || undefined })
    }, 300)
  }

  function handleActionToggle(action: string) {
    const current = filters.actions ?? []
    const next = current.includes(action)
      ? current.filter((a) => a !== action)
      : [...current, action]
    onChange({ ...filters, actions: next.length ? next : undefined })
  }

  function handleClear() {
    setActorInput('')
    onChange({})
  }

  const hasFilters =
    filters.from || filters.to || filters.actor || (filters.actions?.length ?? 0) > 0 || filters.entityType

  return (
    <div className="space-y-3 rounded-lg border bg-card p-4">
      {/* Date range row */}
      <div className="flex flex-wrap gap-3">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-muted-foreground">{t('audit.filter_from')}</label>
          <Input
            type="datetime-local"
            className="h-9 w-48 text-sm"
            value={filters.from ? filters.from.replace('Z', '').slice(0, 16) : ''}
            onChange={(e) => {
              const val = e.target.value
              onChange({
                ...filters,
                from: val ? new Date(val).toISOString() : undefined,
              })
            }}
          />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-muted-foreground">{t('audit.filter_to')}</label>
          <Input
            type="datetime-local"
            className="h-9 w-48 text-sm"
            value={filters.to ? filters.to.replace('Z', '').slice(0, 16) : ''}
            onChange={(e) => {
              const val = e.target.value
              onChange({
                ...filters,
                to: val ? new Date(val).toISOString() : undefined,
              })
            }}
          />
        </div>

        {/* Actor name search */}
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-muted-foreground">{t('audit.filter_actor')}</label>
          <Input
            type="text"
            className="h-9 w-48 text-sm"
            placeholder={t('audit.filter_actor')}
            value={actorInput}
            onChange={(e) => handleActorChange(e.target.value)}
          />
        </div>

        {/* Entity type filter */}
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-muted-foreground">{t('audit.filter_entity')}</label>
          <Select
            className="h-9 w-40 text-sm"
            value={filters.entityType ?? ''}
            onChange={(e) =>
              onChange({ ...filters, entityType: e.target.value || undefined })
            }
          >
            <option value="">{t('audit.filter_entity')}</option>
            {ENTITY_TYPES.map((et) => (
              <option key={et} value={et}>
                {et}
              </option>
            ))}
          </Select>
        </div>

        {/* Clear button */}
        {hasFilters && (
          <div className="flex flex-col justify-end">
            <Button variant="outline" size="sm" onClick={handleClear}>
              {t('audit.filter_clear')}
            </Button>
          </div>
        )}
      </div>

      {/* Action multi-select — rendered as a list of toggleable chips */}
      <div className="flex flex-col gap-1">
        <label className="text-xs font-medium text-muted-foreground">{t('audit.filter_action')}</label>
        <div className="flex flex-wrap gap-1.5">
          {AUDIT_ACTIONS.map((action) => {
            const selected = filters.actions?.includes(action) ?? false
            return (
              <button
                key={action}
                type="button"
                onClick={() => handleActionToggle(action)}
                className={[
                  'rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors',
                  selected
                    ? 'border-primary bg-primary text-primary-foreground'
                    : 'border-muted bg-muted text-muted-foreground hover:border-primary/50',
                ].join(' ')}
              >
                {action}
              </button>
            )
          })}
        </div>
      </div>
    </div>
  )
}
