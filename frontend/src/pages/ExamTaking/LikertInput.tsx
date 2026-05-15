import { cn } from '@/lib/utils'
import type { QuestionOption } from '@/api/sessions'

interface LikertInputProps {
  options: QuestionOption[]
  selected: string | null
  onChange: (optionId: string) => void
}

export function LikertInput({ options, selected, onChange }: LikertInputProps) {
  return (
    <div className="flex flex-wrap gap-2" role="radiogroup">
      {options.map((opt, idx) => {
        const isSelected = opt.id === selected
        return (
          <button
            key={opt.id}
            type="button"
            role="radio"
            aria-checked={isSelected}
            onClick={() => onChange(opt.id)}
            className={cn(
              'min-w-[2.5rem] h-10 px-3 rounded-md border text-sm font-medium transition-colors',
              isSelected
                ? 'bg-primary text-primary-foreground border-primary'
                : 'bg-background border-border hover:bg-muted',
            )}
          >
            {opt.text || idx + 1}
          </button>
        )
      })}
    </div>
  )
}
