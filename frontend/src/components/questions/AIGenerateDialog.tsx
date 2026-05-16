import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Select } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { useGenerateQuestions, type DraftQuestion } from '@/api/ai'
import { useCreateQuestion } from '@/api/questions'
import { useCategories } from '@/api/questions'

// ---- Types ------------------------------------------------------------------

interface CurrentUser {
  role: string
}

// ---- Helpers ----------------------------------------------------------------

/** Returns true if the given role is examiner-level or higher. */
function isExaminerOrAbove(role: string): boolean {
  return ['examiner', 'department_admin', 'super_admin'].includes(role)
}

function flattenCategories(categories: { id: string; name: string; children?: { id: string; name: string; children?: unknown[] }[] }[]): { id: string; name: string }[] {
  const result: { id: string; name: string }[] = []
  function walk(cats: typeof categories) {
    for (const cat of cats) {
      result.push({ id: cat.id, name: cat.name })
      if (cat.children?.length) walk(cat.children as typeof categories)
    }
  }
  walk(categories)
  return result
}

// ---- Draft Preview Row ------------------------------------------------------

interface DraftRowProps {
  question: DraftQuestion
  index: number
  selected: boolean
  onToggle: () => void
  expanded: boolean
  onExpand: () => void
}

function DraftRow({ question, index, selected, onToggle, expanded, onExpand }: DraftRowProps) {
  const { t } = useTranslation()
  return (
    <div className="border rounded-md p-3 space-y-2 text-sm">
      <div className="flex items-start gap-3">
        <input
          type="checkbox"
          checked={selected}
          onChange={onToggle}
          className="mt-0.5 rounded"
          aria-label={`${t('question.selectDraft')} ${index + 1}`}
        />
        <div className="flex-1 min-w-0">
          <p className="font-medium truncate">{question.stem}</p>
          <p className="text-muted-foreground text-xs mt-0.5">
            {question.type} · {question.difficulty}
          </p>
        </div>
        <button
          onClick={onExpand}
          className="text-xs text-primary hover:underline whitespace-nowrap"
        >
          {expanded ? '▲' : '▼'}
        </button>
      </div>

      {expanded && (
        <div className="ml-6 space-y-1 text-xs text-muted-foreground">
          <p className="italic">{question.explanation}</p>
          <ul className="space-y-0.5 mt-1">
            {question.options.map((opt, i) => (
              <li
                key={i}
                className={opt.is_correct ? 'text-green-700 font-medium' : ''}
              >
                {opt.is_correct ? '✓ ' : '○ '}{opt.text}
              </li>
            ))}
          </ul>
          {question.tags.length > 0 && (
            <p className="mt-1">{question.tags.join(', ')}</p>
          )}
        </div>
      )}
    </div>
  )
}

// ---- Main Dialog ------------------------------------------------------------

interface AIGenerateDialogProps {
  open: boolean
  onClose: () => void
  onSuccess: (count: number) => void
}

export function AIGenerateDialog({ open, onClose, onSuccess }: AIGenerateDialogProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const user = qc.getQueryData<CurrentUser>(['auth', 'currentUser'])

  // Only examiner+ can see/use this dialog.
  if (!user || !isExaminerOrAbove(user.role)) return null

  const { data: rawCategories } = useCategories()
  const categories = rawCategories ? flattenCategories(rawCategories) : []

  const generateMutation = useGenerateQuestions()
  const createMutation = useCreateQuestion()

  // Form state
  const [categoryId, setCategoryId] = useState('')
  const [difficulty, setDifficulty] = useState<'easy' | 'medium' | 'hard'>('medium')
  const [count, setCount] = useState(5)
  const [contextText, setContextText] = useState('')

  // Draft state
  const [drafts, setDrafts] = useState<DraftQuestion[]>([])
  const [selectedIndexes, setSelectedIndexes] = useState<Set<number>>(new Set())
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null)
  const [confirmProgress, setConfirmProgress] = useState<number | null>(null)
  const [errorMsg, setErrorMsg] = useState<string | null>(null)

  const hasDrafts = drafts.length > 0

  function reset() {
    setDrafts([])
    setSelectedIndexes(new Set())
    setExpandedIndex(null)
    setConfirmProgress(null)
    setErrorMsg(null)
    generateMutation.reset()
  }

  function handleClose() {
    reset()
    onClose()
  }

  async function handleGenerate() {
    if (!categoryId) {
      setErrorMsg(t('question.aiGenerateNoCategoryError'))
      return
    }
    setErrorMsg(null)
    reset()
    try {
      const result = await generateMutation.mutateAsync({
        category_id: categoryId,
        difficulty,
        count,
        context_text: contextText || undefined,
      })
      setDrafts(result.questions)
      // Pre-select all drafts.
      setSelectedIndexes(new Set(result.questions.map((_, i) => i)))
    } catch (err) {
      const code = (err as Error & { code?: string }).code
      if (code === 'AI_RATE_LIMITED') {
        setErrorMsg(t('errors.aiRateLimited'))
      } else {
        setErrorMsg(t('errors.aiUnavailable'))
      }
    }
  }

  function toggleSelect(index: number) {
    setSelectedIndexes((prev) => {
      const next = new Set(prev)
      if (next.has(index)) next.delete(index)
      else next.add(index)
      return next
    })
  }

  function selectAll() {
    setSelectedIndexes(new Set(drafts.map((_, i) => i)))
  }

  async function handleConfirm() {
    const selected = drafts.filter((_, i) => selectedIndexes.has(i))
    if (selected.length === 0) return

    setConfirmProgress(0)
    let confirmed = 0

    for (const draft of selected) {
      // Map draft type to the existing question type values.
      const typeMap: Record<string, string> = {
        single_choice: 'single',
        multiple_choice: 'multiple',
        true_false: 'truefalse',
      }
      const mappedType = typeMap[draft.type] ?? 'single'

      try {
        await createMutation.mutateAsync({
          type: mappedType,
          difficulty: draft.difficulty,
          category_id: categoryId,
          default_locale: 'en',
          stem: draft.stem,
          explanation: draft.explanation,
          answer_options: draft.options.map((opt, idx) => ({
            sort_order: idx,
            is_correct: opt.is_correct,
            body: opt.text,
          })),
          tag_ids: [],
        })
        confirmed++
        setConfirmProgress(confirmed)
      } catch {
        // Continue with remaining drafts even if one fails.
      }
    }

    onSuccess(confirmed)
    handleClose()
  }

  const allSelected = selectedIndexes.size === drafts.length && drafts.length > 0

  return (
    <Dialog open={open} onOpenChange={(o) => !o && handleClose()}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t('question.aiGenerateTitle')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* Generation form */}
          {!hasDrafts && (
            <div className="space-y-3">
              {/* Category */}
              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('questionBank.column.category')}
                </label>
                <Select
                  className="w-full"
                  value={categoryId}
                  onChange={(e) => setCategoryId(e.target.value)}
                >
                  <option value="">{t('questionBank.filter.allCategories')}</option>
                  {categories.map((cat) => (
                    <option key={cat.id} value={cat.id}>
                      {cat.name}
                    </option>
                  ))}
                </Select>
              </div>

              {/* Difficulty */}
              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('question.aiGenerateDifficulty')}
                </label>
                <Select
                  className="w-full"
                  value={difficulty}
                  onChange={(e) => setDifficulty(e.target.value as 'easy' | 'medium' | 'hard')}
                >
                  <option value="easy">{t('questionBank.difficulty.easy')}</option>
                  <option value="medium">{t('questionBank.difficulty.medium')}</option>
                  <option value="hard">{t('questionBank.difficulty.hard')}</option>
                </Select>
              </div>

              {/* Count */}
              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('question.aiGenerateCount')}
                </label>
                <Input
                  type="number"
                  min={1}
                  max={10}
                  value={count}
                  onChange={(e) => setCount(Math.min(10, Math.max(1, Number(e.target.value))))}
                  className="w-32"
                />
              </div>

              {/* Context */}
              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('question.aiGenerateContext')}
                </label>
                <textarea
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring resize-none"
                  rows={4}
                  maxLength={2000}
                  placeholder={t('question.aiGenerateContextHint')}
                  value={contextText}
                  onChange={(e) => setContextText(e.target.value)}
                />
                <p className="text-xs text-muted-foreground text-right mt-0.5">
                  {contextText.length}/2000
                </p>
              </div>
            </div>
          )}

          {/* Error */}
          {errorMsg && (
            <p className="text-sm text-red-600">{errorMsg}</p>
          )}

          {/* Loading */}
          {generateMutation.isPending && (
            <p className="text-sm text-muted-foreground animate-pulse">
              {t('question.aiGenerating')}
            </p>
          )}

          {/* Draft preview */}
          {hasDrafts && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium">{t('question.aiGeneratePreview')}</p>
                <div className="flex gap-2">
                  {!allSelected && (
                    <button onClick={selectAll} className="text-xs text-primary hover:underline">
                      {t('question.selectAll')}
                    </button>
                  )}
                  <button
                    onClick={reset}
                    className="text-xs text-muted-foreground hover:underline"
                  >
                    ↺
                  </button>
                </div>
              </div>
              <div className="space-y-2 max-h-80 overflow-y-auto pr-1">
                {drafts.map((draft, i) => (
                  <DraftRow
                    key={i}
                    index={i}
                    question={draft}
                    selected={selectedIndexes.has(i)}
                    onToggle={() => toggleSelect(i)}
                    expanded={expandedIndex === i}
                    onExpand={() => setExpandedIndex(expandedIndex === i ? null : i)}
                  />
                ))}
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            {t('common.cancel')}
          </Button>
          {!hasDrafts ? (
            <Button
              onClick={handleGenerate}
              disabled={generateMutation.isPending}
            >
              {generateMutation.isPending
                ? t('question.aiGenerating')
                : t('question.aiGenerate')}
            </Button>
          ) : (
            <Button
              onClick={handleConfirm}
              disabled={selectedIndexes.size === 0 || confirmProgress !== null}
            >
              {confirmProgress !== null
                ? `${t('questionBank.import.progress', { n: confirmProgress, total: selectedIndexes.size })}`
                : `${t('question.confirmSelected')} (${selectedIndexes.size})`}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
