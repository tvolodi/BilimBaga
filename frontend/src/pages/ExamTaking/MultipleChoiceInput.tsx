import { useId } from 'react'
import { cn } from '@/lib/utils'
import type { QuestionOption } from '@/api/sessions'

interface MultipleChoiceInputProps {
  options: QuestionOption[]
  selected: string[]
  onChange: (optionIds: string[]) => void
}

export function MultipleChoiceInput({ options, selected, onChange }: MultipleChoiceInputProps) {
  const groupId = useId()

  const toggle = (optId: string) => {
    if (selected.includes(optId)) {
      onChange(selected.filter((id) => id !== optId))
    } else {
      onChange([...selected, optId])
    }
  }

  return (
    <div className="space-y-2">
      {options.map((opt) => {
        const inputId = `${groupId}-${opt.id}`
        const isChecked = selected.includes(opt.id)
        return (
          <label
            key={opt.id}
            htmlFor={inputId}
            className={cn(
              'flex items-center gap-3 p-3 rounded-md border cursor-pointer transition-colors',
              isChecked
                ? 'border-primary bg-primary/5'
                : 'border-border hover:bg-muted',
            )}
          >
            <input
              type="checkbox"
              id={inputId}
              value={opt.id}
              checked={isChecked}
              onChange={() => toggle(opt.id)}
              className="accent-primary"
            />
            <span className="text-sm">{opt.text}</span>
          </label>
        )
      })}
    </div>
  )
}
