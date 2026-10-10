import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { NON_ASSIGNABLE_PERMISSIONS, type CataloguePermission } from '@/api/roles'

interface PermissionMatrixProps {
  catalogue: CataloguePermission[]
  /** Selected permission ids. */
  selected: ReadonlySet<string>
  onChange: (next: Set<string>) => void
  /** System roles: everything is shown but nothing can be changed. */
  readOnly?: boolean
}

const permKey = (p: CataloguePermission) => `${p.resource}:${p.action}`

/**
 * Permission matrix (FR-BB117 AC-13): one fieldset per resource, one labelled checkbox per
 * action. Non-assignable permissions (AC-6) are shown disabled and cannot be ticked.
 */
export function PermissionMatrix({ catalogue, selected, onChange, readOnly = false }: PermissionMatrixProps) {
  const { t } = useTranslation()

  const groups = useMemo(() => {
    const map = new Map<string, CataloguePermission[]>()
    for (const p of catalogue) {
      const list = map.get(p.resource) ?? []
      list.push(p)
      map.set(p.resource, list)
    }
    return [...map.entries()].sort(([a], [b]) => a.localeCompare(b))
  }, [catalogue])

  function toggle(id: string, checked: boolean) {
    const next = new Set(selected)
    if (checked) next.add(id)
    else next.delete(id)
    onChange(next)
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2" data-testid="permission-matrix">
      {groups.map(([resource, perms]) => (
        <fieldset key={resource} className="min-w-0 rounded-md border p-3">
          <legend className="px-1 text-sm font-semibold">
            {t(`roles.resources.${resource}`, { defaultValue: resource })}
          </legend>
          <ul className="space-y-1">
            {perms.map((p) => {
              const key = permKey(p)
              const blocked = NON_ASSIGNABLE_PERMISSIONS.includes(key)
              const disabled = readOnly || blocked
              const inputId = `perm-${p.id}`
              const label = t(`roles.permissions.${p.resource}.${p.action}`, { defaultValue: key })
              return (
                <li key={p.id} className="flex items-start gap-2">
                  <input
                    id={inputId}
                    type="checkbox"
                    className="mt-1 h-4 w-4 shrink-0 rounded border-border disabled:opacity-50"
                    checked={selected.has(p.id)}
                    disabled={disabled}
                    onChange={(e) => toggle(p.id, e.target.checked)}
                    aria-describedby={blocked && !readOnly ? `${inputId}-hint` : undefined}
                  />
                  <label htmlFor={inputId} className={`break-words text-sm ${disabled ? 'text-muted-foreground' : ''}`}>
                    {label}
                    {blocked && !readOnly && (
                      <span id={`${inputId}-hint`} className="block text-xs">
                        {t('roles.matrix.notAssignable')}
                      </span>
                    )}
                  </label>
                </li>
              )
            })}
          </ul>
        </fieldset>
      ))}
    </div>
  )
}
