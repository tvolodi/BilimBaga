import { useId } from 'react'
import { cn } from '@/lib/utils'
import type { QuestionOption } from '@/api/sessions'

interface SingleChoiceInputProps {
  options: QuestionOption[]
  selected: string | null
  onChange: (optionId: string) => void
}

export function SingleChoiceInput({ options, selected, onChange }: SingleChoiceInputProps) {
  const groupId = useId()

  return (
    <div className="space-y-2" role="radiogroup">
      {options.map((opt) => {
        const inputId = `${groupId}-${opt.id}`
        const isSelected = opt.id === selected
        return (
          <label
            key={opt.id}
            htmlFor={inputId}
            className={cn(
              'flex items-center gap-3 p-3 rounded-md border cursor-pointer transition-colors',
              isSelected
                ? 'border-primary bg-primary/5'
                : 'border-border hover:bg-muted',
            )}
          >
            <input
              type="radio"
              id={inputId}
              name={groupId}
              value={opt.id}
              checked={isSelected}
              onChange={() => onChange(opt.id)}
              className="accent-primary"
            />
            <span className="text-sm">{opt.text}</span>
          </label>
        )
      })}
    </div>
  )
}
