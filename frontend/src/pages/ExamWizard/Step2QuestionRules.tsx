import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical, Plus, Trash2, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import { useCategories, flattenCategories } from '@/api/categories'
import { useTags } from '@/api/questions'
import {
  useAddRule,
  useUpdateRule,
  useDeleteRule,
  type QuestionRuleDetail,
  type ExamDetail,
  type ExamApiError,
} from '@/api/exams'
import { Step2QuestionPickerModal } from './Step2QuestionPickerModal'

// ---- Types ------------------------------------------------------------------

interface LocalRule {
  /** undefined means not yet persisted to API */
  id?: string
  mode: 'manual' | 'random'
  category_id: string
  difficulty: 'easy' | 'medium' | 'hard' | ''
  tag_ids: string[]
  count: string
  section_id: string | null
  sort_order: number
  questions?: Array<{ question_id: string; sort_order: number }>
}

// ---- Sortable row -----------------------------------------------------------

function SortableRuleRow({
  rule,
  index,
  onModeToggle,
  onFieldChange,
  onDelete,
  onEditQuestions,
  categories,
  tags,
  isDeleting,
}: {
  rule: LocalRule
  index: number
  onModeToggle: (i: number) => void
  onFieldChange: <K extends keyof LocalRule>(i: number, key: K, value: LocalRule[K]) => void
  onDelete: (i: number) => void
  onEditQuestions: (i: number) => void
  categories: Array<{ id: string; name: string }>
  tags: Array<{ id: string; name: string }>
  isDeleting: boolean
}) {
  const { t } = useTranslation()
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: String(index),
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="border rounded-lg p-4 bg-background space-y-3"
    >
      <div className="flex items-center gap-2">
        {/* Drag handle */}
        <button
          type="button"
          {...attributes}
          {...listeners}
          className="cursor-grab touch-none text-muted-foreground hover:text-foreground"
          aria-label="drag"
        >
          <GripVertical size={16} />
        </button>

        {/* Mode toggle */}
        <div className="flex rounded-md border overflow-hidden">
          <button
            type="button"
            onClick={() => onModeToggle(index)}
            className={`px-3 py-1 text-sm transition-colors ${
              rule.mode === 'manual'
                ? 'bg-primary text-primary-foreground'
                : 'bg-background text-muted-foreground hover:bg-accent'
            }`}
          >
            {t('exam.wizard.step2.mode.manual')}
          </button>
          <button
            type="button"
            onClick={() => onModeToggle(index)}
            className={`px-3 py-1 text-sm transition-colors ${
              rule.mode === 'random'
                ? 'bg-primary text-primary-foreground'
                : 'bg-background text-muted-foreground hover:bg-accent'
            }`}
          >
            {t('exam.wizard.step2.mode.random')}
          </button>
        </div>

        <div className="flex-1" />

        {/* Delete */}
        <button
          type="button"
          onClick={() => onDelete(index)}
          disabled={isDeleting}
          className="text-muted-foreground hover:text-destructive transition-colors"
          aria-label="delete rule"
        >
          <Trash2 size={16} />
        </button>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        {/* Category */}
        <div className="space-y-1">
          <Label className="text-xs">{t('exam.rule.category')}</Label>
          <Select
            value={rule.category_id}
            onChange={(e) => onFieldChange(index, 'category_id', e.target.value)}
          >
            <option value="">—</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </Select>
        </div>

        {/* Difficulty */}
        <div className="space-y-1">
          <Label className="text-xs">{t('exam.rule.difficulty')}</Label>
          <Select
            value={rule.difficulty}
            onChange={(e) => onFieldChange(index, 'difficulty', e.target.value as LocalRule['difficulty'])}
          >
            <option value="">—</option>
            <option value="easy">{t('questionBank.difficulty.easy')}</option>
            <option value="medium">{t('questionBank.difficulty.medium')}</option>
            <option value="hard">{t('questionBank.difficulty.hard')}</option>
          </Select>
        </div>

        {/* Tags */}
        <div className="space-y-1">
          <Label className="text-xs">{t('exam.rule.tags')}</Label>
          <Select
            value=""
            onChange={(e) => {
              const val = e.target.value
              if (!val) return
              const current = rule.tag_ids
              if (current.includes(val)) {
                onFieldChange(index, 'tag_ids', current.filter((t) => t !== val))
              } else {
                onFieldChange(index, 'tag_ids', [...current, val])
              }
            }}
          >
            <option value="">+ {t('exam.rule.tags')}</option>
            {tags.map((tg) => (
              <option key={tg.id} value={tg.id}>
                {rule.tag_ids.includes(tg.id) ? '✓ ' : ''}{tg.name}
              </option>
            ))}
          </Select>
          {rule.tag_ids.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-1">
              {rule.tag_ids.map((tid) => {
                const tag = tags.find((tg) => tg.id === tid)
                return (
                  <span
                    key={tid}
                    className="inline-flex items-center gap-0.5 bg-secondary text-secondary-foreground text-xs rounded px-1.5 py-0.5"
                  >
                    {tag?.name ?? tid}
                    <button
                      type="button"
                      onClick={() => onFieldChange(index, 'tag_ids', rule.tag_ids.filter((t) => t !== tid))}
                      className="hover:text-destructive"
                    >
                      ×
                    </button>
                  </span>
                )
              })}
            </div>
          )}
        </div>

        {/* Count */}
        <div className="space-y-1">
          <Label className="text-xs">{t('exam.rule.count')}</Label>
          <Input
            type="number"
            min={1}
            value={rule.count}
            onChange={(e) => onFieldChange(index, 'count', e.target.value as unknown as LocalRule['count'])}
            className="h-10"
          />
        </div>
      </div>

      {/* Manual: edit questions button */}
      {rule.mode === 'manual' && (
        <div>
          <Button type="button" variant="outline" size="sm" onClick={() => onEditQuestions(index)}>
            {t('exam.wizard.step2.editQuestions')}
            {rule.questions && rule.questions.length > 0 && (
              <span className="ml-1.5 bg-primary text-primary-foreground rounded-full w-5 h-5 flex items-center justify-center text-xs">
                {rule.questions.length}
              </span>
            )}
          </Button>
        </div>
      )}
    </div>
  )
}

// ---- Main component ---------------------------------------------------------

interface Step2QuestionRulesProps {
  examId: string
  sectionId: string | null
  exam: ExamDetail | undefined
  onBack: () => void
  onNext: () => void
}

function ruleToLocal(r: QuestionRuleDetail, sectionId: string | null): LocalRule {
  return {
    id: r.id,
    mode: r.mode,
    category_id: r.category_id ?? '',
    difficulty: (r.difficulty as LocalRule['difficulty']) ?? '',
    tag_ids: r.tag_ids ?? [],
    count: String(r.count),
    section_id: sectionId,
    sort_order: r.sort_order,
    questions: r.questions,
  }
}

export function Step2QuestionRules({
  examId,
  sectionId,
  exam,
  onBack,
  onNext,
}: Step2QuestionRulesProps) {
  const { t } = useTranslation()

  const { data: categoriesTree = [] } = useCategories()
  const { data: tagsData } = useTags()
  const categories = flattenCategories(categoriesTree)
  const tags = tagsData ?? []

  const addRule = useAddRule(examId)
  const updateRule = useUpdateRule(examId)
  const deleteRule = useDeleteRule(examId)

  const [rules, setRules] = useState<LocalRule[]>(() =>
    (exam?.rules ?? []).map((r) => ruleToLocal(r, sectionId)),
  )
  const [pickerOpen, setPickerOpen] = useState(false)
  const [pickerRuleIndex, setPickerRuleIndex] = useState<number | null>(null)
  const [apiError, setApiError] = useState<string | null>(null)
  const [isSaving, setIsSaving] = useState(false)

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over || active.id === over.id) return
    const oldIndex = Number(active.id)
    const newIndex = Number(over.id)
    setRules((prev) => {
      const reordered = arrayMove(prev, oldIndex, newIndex)
      return reordered.map((r, i) => ({ ...r, sort_order: i }))
    })
  }

  function addNewRule() {
    setRules((prev) => [
      ...prev,
      {
        mode: 'random',
        category_id: '',
        difficulty: '',
        tag_ids: [],
        count: '1',
        section_id: sectionId,
        sort_order: prev.length,
      },
    ])
  }

  function toggleMode(i: number) {
    setRules((prev) =>
      prev.map((r, idx) =>
        idx === i ? { ...r, mode: r.mode === 'manual' ? 'random' : 'manual' } : r,
      ),
    )
  }

  function changeField<K extends keyof LocalRule>(i: number, key: K, value: LocalRule[K]) {
    setRules((prev) => prev.map((r, idx) => (idx === i ? { ...r, [key]: value } : r)))
  }

  async function deleteRuleAt(i: number) {
    const rule = rules[i]
    if (rule.id) {
      try {
        await deleteRule.mutateAsync(rule.id)
      } catch (err) {
        setApiError((err as ExamApiError).message)
        return
      }
    }
    setRules((prev) => prev.filter((_, idx) => idx !== i).map((r, idx) => ({ ...r, sort_order: idx })))
  }

  function openPicker(i: number) {
    setPickerRuleIndex(i)
    setPickerOpen(true)
  }

  function handlePickerConfirm(questionIds: string[]) {
    if (pickerRuleIndex === null) return
    setRules((prev) =>
      prev.map((r, i) =>
        i === pickerRuleIndex
          ? { ...r, questions: questionIds.map((qid, idx) => ({ question_id: qid, sort_order: idx })) }
          : r,
      ),
    )
    setPickerOpen(false)
  }

  const saveAndNext = useCallback(async () => {
    setApiError(null)
    setIsSaving(true)
    try {
      for (let i = 0; i < rules.length; i++) {
        const rule = rules[i]
        const count = Number(rule.count)
        if (!count || count < 1) {
          setApiError(`Rule ${i + 1}: count must be at least 1`)
          setIsSaving(false)
          return
        }
        const payload = {
          section_id: rule.section_id,
          mode: rule.mode,
          category_id: rule.category_id || null,
          tag_ids: rule.tag_ids,
          difficulty: rule.difficulty || null,
          count,
          sort_order: rule.sort_order,
        }
        if (rule.id) {
          await updateRule.mutateAsync({ ruleId: rule.id, body: payload })
        } else {
          await addRule.mutateAsync(payload)
        }
      }
      onNext()
    } catch (err) {
      setApiError((err as ExamApiError).message)
    } finally {
      setIsSaving(false)
    }
  }, [rules, addRule, updateRule, onNext])

  const pickerRule = pickerRuleIndex !== null ? rules[pickerRuleIndex] : null

  return (
    <div className="space-y-4">
      {apiError && (
        <div className="rounded-md bg-destructive/10 border border-destructive/20 px-4 py-3 text-sm text-destructive">
          {apiError}
        </div>
      )}

      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={rules.map((_, i) => String(i))} strategy={verticalListSortingStrategy}>
          <div className="space-y-3">
            {rules.map((rule, i) => (
              <SortableRuleRow
                key={`rule-${i}`}
                rule={rule}
                index={i}
                onModeToggle={toggleMode}
                onFieldChange={changeField}
                onDelete={deleteRuleAt}
                onEditQuestions={openPicker}
                categories={categories}
                tags={tags}
                isDeleting={deleteRule.isPending}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>

      <Button type="button" variant="outline" onClick={addNewRule} className="w-full">
        <Plus size={16} className="mr-2" />
        {t('exam.wizard.step2.addRule')}
      </Button>

      <div className="flex justify-between pt-4">
        <Button type="button" variant="outline" onClick={onBack}>
          {t('exam.wizard.back')}
        </Button>
        <Button type="button" onClick={saveAndNext} disabled={isSaving}>
          {isSaving ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              {t('exam.wizard.saving')}
            </>
          ) : (
            t('exam.wizard.next')
          )}
        </Button>
      </div>

      {pickerOpen && pickerRule !== null && (
        <Step2QuestionPickerModal
          examId={examId}
          sectionId={sectionId}
          ruleId={pickerRule.id}
          selectedIds={pickerRule.questions?.map((q) => q.question_id) ?? []}
          onConfirm={handlePickerConfirm}
          onClose={() => setPickerOpen(false)}
        />
      )}
    </div>
  )
}
