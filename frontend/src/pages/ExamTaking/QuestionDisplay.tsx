import { type SessionQuestion, type SavedAnswer } from '@/api/sessions'
import { SingleChoiceInput } from './SingleChoiceInput'
import { MultipleChoiceInput } from './MultipleChoiceInput'
import { LikertInput } from './LikertInput'
import { ShortTextInput } from './ShortTextInput'
import { FlagButton } from './FlagButton'

interface QuestionDisplayProps {
  question: SessionQuestion
  savedAnswer: SavedAnswer | undefined
  onAnswer: (optionIds: string[], textAnswer: string | null) => void
  isFlagged: boolean
  onToggleFlag: () => void
}

export function QuestionDisplay({
  question,
  savedAnswer,
  onAnswer,
  isFlagged,
  onToggleFlag,
}: QuestionDisplayProps) {
  const renderInput = () => {
    switch (question.type) {
      case 'single_choice':
      case 'true_false':
        return (
          <SingleChoiceInput
            options={question.options}
            selected={savedAnswer?.selected_option_ids[0] ?? null}
            onChange={(optId) => onAnswer([optId], null)}
          />
        )
      case 'multiple_choice':
        return (
          <MultipleChoiceInput
            options={question.options}
            selected={savedAnswer?.selected_option_ids ?? []}
            onChange={(optIds) => onAnswer(optIds, null)}
          />
        )
      case 'likert':
        return (
          <LikertInput
            options={question.options}
            selected={savedAnswer?.selected_option_ids[0] ?? null}
            onChange={(optId) => onAnswer([optId], null)}
          />
        )
      case 'short_text':
        return (
          <ShortTextInput
            value={savedAnswer?.text_answer ?? ''}
            onChange={(text) => onAnswer([], text)}
          />
        )
      default:
        return null
    }
  }

  return (
    <div className="bg-card border rounded-lg p-5 space-y-4">
      <div className="flex items-start justify-between gap-3">
        <p className="text-sm font-medium leading-relaxed flex-1">{question.stem}</p>
        <FlagButton isFlagged={isFlagged} onToggle={onToggleFlag} />
      </div>
      {renderInput()}
    </div>
  )
}
