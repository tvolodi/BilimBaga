import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, X, Loader2 } from 'lucide-react'
import { useDebounceValue } from 'usehooks-ts'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { useQuestions, type QuestionListItem } from '@/api/questions'
import { useCategories, flattenCategories } from '@/api/categories'
import { useSetRuleQuestions, type ExamApiError } from '@/api/exams'

// ---- Props ------------------------------------------------------------------

interface Step2QuestionPickerModalProps {
  examId: string
  sectionId: string | null
  ruleId?: string
  selectedIds: string[]
  onConfirm: (questionIds: string[]) => void
  onClose: () => void
}

// ---- Component --------------------------------------------------------------

export function Step2QuestionPickerModal({
  examId,
  ruleId,
  selectedIds,
  onConfirm,
  onClose,
}: Step2QuestionPickerModalProps) {
  const { t } = useTranslation()

  const [search, setSearch] = useState('')
  const [debouncedSearch] = useDebounceValue(search, 300)
  const [categoryId, setCategoryId] = useState('')
  const [difficulty, setDifficulty] = useState('')
  const [page, setPage] = useState(1)
  const [checked, setChecked] = useState<Set<string>>(new Set(selectedIds))
  const [apiError, setApiError] = useState<string | null>(null)

  const { data: categoriesTree = [] } = useCategories()
  const categories = flattenCategories(categoriesTree)

  const { data: questionsData, isLoading } = useQuestions({
    search: debouncedSearch || undefined,
    category_id: categoryId || undefined,
    difficulties: difficulty ? [difficulty as 'easy' | 'medium' | 'hard'] : undefined,
    statuses: ['active'],
    page,
    per_page: 20,
  })

  const setRuleQuestions = useSetRuleQuestions(examId)

  // Reset page when filters change
  useEffect(() => {
    setPage(1)
  }, [debouncedSearch, categoryId, difficulty])

  function toggleQuestion(id: string) {
    setChecked((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const handleConfirm = useCallback(async () => {
    const ids = Array.from(checked)
    if (ruleId) {
      // Persist to API
      try {
        await setRuleQuestions.mutateAsync({
          ruleId,
          questions: ids.map((qid, i) => ({ question_id: qid, sort_order: i })),
        })
      } catch (err) {
        setApiError((err as ExamApiError).message)
        return
      }
    }
    onConfirm(ids)
  }, [checked, ruleId, setRuleQuestions, onConfirm])

  const items: QuestionListItem[] = questionsData?.items ?? []
  const total = questionsData?.meta.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / 20))

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-w-2xl max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>{t('exam.wizard.step2.editQuestions')}</DialogTitle>
        </DialogHeader>

        {/* Filters */}
        <div className="flex gap-2 flex-wrap">
          <div className="relative flex-1 min-w-40">
            <Search size={14} className="absolute left-2.5 top-3 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('questionBank.filter.search')}
              className="pl-8"
            />
          </div>
          <Select value={categoryId} onChange={(e) => setCategoryId(e.target.value)} className="w-40">
            <option value="">{t('questionBank.filter.allCategories')}</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </Select>
          <Select value={difficulty} onChange={(e) => setDifficulty(e.target.value)} className="w-36">
            <option value="">—</option>
            <option value="easy">{t('questionBank.difficulty.easy')}</option>
            <option value="medium">{t('questionBank.difficulty.medium')}</option>
            <option value="hard">{t('questionBank.difficulty.hard')}</option>
          </Select>
        </div>

        {apiError && (
          <div className="rounded-md bg-destructive/10 border border-destructive/20 px-3 py-2 text-sm text-destructive">
            {apiError}
          </div>
        )}

        {/* Question list */}
        <div className="flex-1 overflow-y-auto space-y-1 min-h-0">
          {isLoading && (
            <div className="flex justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          )}
          {!isLoading && items.length === 0 && (
            <div className="text-center py-8 text-muted-foreground text-sm">
              {t('questionBank.empty.title')}
            </div>
          )}
          {items.map((q) => (
            <label
              key={q.id}
              className={`flex items-start gap-3 p-3 rounded-md cursor-pointer transition-colors hover:bg-accent ${
                checked.has(q.id) ? 'bg-accent' : ''
              }`}
            >
              <input
                type="checkbox"
                className="mt-0.5 shrink-0"
                checked={checked.has(q.id)}
                onChange={() => toggleQuestion(q.id)}
              />
              <div className="min-w-0 flex-1">
                <p className="text-sm truncate">{q.stem_preview}</p>
                <div className="flex gap-2 mt-0.5">
                  <span className="text-xs text-muted-foreground">{q.category_name}</span>
                  <span className="text-xs text-muted-foreground capitalize">{q.difficulty}</span>
                </div>
              </div>
            </label>
          ))}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between text-sm text-muted-foreground pt-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
            >
              {t('users.pagination.previous')}
            </Button>
            <span>
              {page} / {totalPages}
            </span>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              {t('users.pagination.next')}
            </Button>
          </div>
        )}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            <X size={14} className="mr-1.5" />
            {t('common.cancel')}
          </Button>
          <Button type="button" onClick={handleConfirm} disabled={setRuleQuestions.isPending}>
            {setRuleQuestions.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {t('exam.wizard.step2.editQuestions')}
            {checked.size > 0 && (
              <span className="ml-1.5 bg-primary-foreground/20 rounded-full px-1.5 text-xs">
                {checked.size}
              </span>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
