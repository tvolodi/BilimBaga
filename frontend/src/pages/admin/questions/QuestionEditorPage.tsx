import { useState, useEffect, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
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
import { GripVertical, ArrowLeft, Check, X, ChevronDown, ChevronUp } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import {
  useQuestion,
  useCreateQuestion,
  useUpdateQuestion,
  useUpdateQuestionStatus,
  useCategories,
  useTags,
  useCreateTag,
  type QuestionDetail,
  type Category,
} from '@/api/questions'

// ---- Types ------------------------------------------------------------------

type QuestionType = 'single' | 'multiple' | 'truefalse' | 'shorttext' | 'likert'
type Difficulty = 'easy' | 'medium' | 'hard'
type Status = 'draft' | 'review' | 'active' | 'archived'

const MISSING_MARK = '○'
const LOCALES = ['en', 'kk', 'ru'] as const
type Locale = (typeof LOCALES)[number]

interface LocalAnswerOption {
  tempId: string
  id?: string
  sort_order: number
  is_correct: boolean
  likert_weight: number | null
  likert_polarity: string | null
  translations: Record<string, { body: string }>
}

interface FormState {
  type: QuestionType
  difficulty: Difficulty
  category_id: string
  tag_ids: string[]
  model_answer: string
  auto_grade: boolean
  translations: Record<string, { stem: string; explanation: string }>
  answer_options: LocalAnswerOption[]
}

// ---- Helpers ----------------------------------------------------------------

function flattenCategories(cats: Category[]): Category[] {
  const out: Category[] = []
  function walk(list: Category[]) {
    for (const c of list) {
      out.push(c)
      if (c.children?.length) walk(c.children)
    }
  }
  walk(cats)
  return out
}

function getCategoryPath(cats: Category[], id: string): string {
  const flat = flattenCategories(cats)
  const map = new Map(flat.map((c) => [c.id, c]))

  function buildPath(catId: string): string[] {
    const cat = map.get(catId)
    if (!cat) return []
    if (cat.parent_id) return [...buildPath(cat.parent_id), cat.name]
    return [cat.name]
  }

  return buildPath(id).join(' › ')
}

function makeEmptyForm(): FormState {
  const translations: FormState['translations'] = {}
  for (const loc of LOCALES) {
    translations[loc] = { stem: '', explanation: '' }
  }
  return {
    type: 'single',
    difficulty: 'medium',
    category_id: '',
    tag_ids: [],
    model_answer: '',
    auto_grade: false,
    translations,
    answer_options: [
      {
        tempId: crypto.randomUUID(),
        sort_order: 0,
        is_correct: false,
        likert_weight: null,
        likert_polarity: null,
        translations: Object.fromEntries(LOCALES.map((l) => [l, { body: '' }])),
      },
      {
        tempId: crypto.randomUUID(),
        sort_order: 1,
        is_correct: false,
        likert_weight: null,
        likert_polarity: null,
        translations: Object.fromEntries(LOCALES.map((l) => [l, { body: '' }])),
      },
    ],
  }
}

function detailToForm(q: QuestionDetail): FormState {
  const translations: FormState['translations'] = {}
  for (const loc of LOCALES) {
    translations[loc] = {
      stem: q.translations[loc]?.stem ?? '',
      explanation: q.translations[loc]?.explanation ?? '',
    }
  }
  const answer_options: LocalAnswerOption[] = (q.answer_options ?? []).map((opt) => ({
    tempId: opt.id,
    id: opt.id,
    sort_order: opt.sort_order,
    is_correct: opt.is_correct,
    likert_weight: opt.likert_weight,
    likert_polarity: opt.likert_polarity,
    translations: Object.fromEntries(
      LOCALES.map((l) => [l, { body: opt.translations[l]?.text ?? '' }]),
    ),
  }))
  return {
    type: q.type,
    difficulty: q.difficulty,
    category_id: q.category_id,
    tag_ids: q.tag_ids ?? [],
    model_answer: q.model_answer ?? '',
    auto_grade: q.auto_grade ?? false,
    translations,
    answer_options,
  }
}

// ---- Sub-components ---------------------------------------------------------

function AutoSaveStatus({ status }: { status: 'idle' | 'saving' | 'saved' | 'error' }) {
  const { t } = useTranslation()
  if (status === 'idle') return null
  if (status === 'saving')
    return <span className="text-xs text-muted-foreground">{t('questionEditor.autosave.saving')}</span>
  if (status === 'saved')
    return <span className="text-xs text-green-600">{t('questionEditor.autosave.saved')}</span>
  return <span className="text-xs text-red-500">{t('questionEditor.error.saveFailed')}</span>
}

interface StatusBadgeProps {
  status: Status
}

function EditorStatusBadge({ status }: StatusBadgeProps) {
  const { t } = useTranslation()
  const variant =
    status === 'active'
      ? 'success'
      : status === 'draft'
        ? 'secondary'
        : status === 'review'
          ? 'warning'
          : 'outline'
  return <Badge variant={variant}>{t(`questionEditor.status.${status}`)}</Badge>
}

interface CoverageTabsProps {
  activeLocale: Locale
  onLocaleChange: (locale: Locale) => void
  translations: FormState['translations']
}

function CoverageTabs({ activeLocale, onLocaleChange, translations }: CoverageTabsProps) {
  const { t } = useTranslation()
  return (
    <div className="flex gap-1 border-b mb-4">
      {LOCALES.map((loc) => {
        const hasContent = translations[loc]?.stem?.trim().length > 0
        return (
          <button
            key={loc}
            onClick={() => onLocaleChange(loc)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors flex items-center gap-1.5 ${
              activeLocale === loc
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <span>{loc.toUpperCase()}</span>
            {hasContent ? (
              <span title={t('questionEditor.locale.coverage.complete')}><Check size={12} className="text-green-500" /></span>
            ) : (
              <span className="text-xs text-muted-foreground" title={t('questionEditor.locale.coverage.missing')}>{MISSING_MARK}</span>
            )}
          </button>
        )
      })}
    </div>
  )
}

interface SortableAnswerOptionProps {
  option: LocalAnswerOption
  activeLocale: Locale
  questionType: QuestionType
  onUpdate: (tempId: string, updated: Partial<LocalAnswerOption>) => void
  onUpdateTranslation: (tempId: string, locale: Locale, body: string) => void
  onDelete: (tempId: string) => void
  onToggleCorrect: (tempId: string) => void
  t: (key: string) => string
}

function SortableAnswerOption({
  option,
  activeLocale,
  questionType,
  onUpdate,
  onUpdateTranslation,
  onDelete,
  onToggleCorrect,
  t,
}: SortableAnswerOptionProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: option.tempId,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  const isLocked = questionType === 'truefalse'

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex items-start gap-2 p-3 border rounded-md bg-background"
    >
      {!isLocked && (
        <button
          {...attributes}
          {...listeners}
          className="mt-2 cursor-grab text-muted-foreground hover:text-foreground"
          aria-label={t('common.dragHandle')}
        >
          <GripVertical size={16} aria-hidden="true" />
        </button>
      )}
      <div className="flex-1 space-y-2">
        <Input
          value={option.translations[activeLocale]?.body ?? ''}
          onChange={(e) => onUpdateTranslation(option.tempId, activeLocale, e.target.value)}
          disabled={isLocked}
          placeholder={t('questionEditor.option.placeholder')}
          className="text-sm"
        />
        {questionType === 'likert' && (
          <div className="flex gap-2">
            <div className="flex-1">
              <Label className="text-xs">{t('questionEditor.option.likertWeight')}</Label>
              <Input
                type="number"
                value={option.likert_weight ?? ''}
                onChange={(e) =>
                  onUpdate(option.tempId, {
                    likert_weight: e.target.value ? Number(e.target.value) : null,
                  })
                }
                className="text-sm h-8"
              />
            </div>
            <div className="flex-1">
              <Label className="text-xs">{t('questionEditor.option.likertPolarity')}</Label>
              <Select
                value={option.likert_polarity ?? ''}
                onChange={(e) =>
                  onUpdate(option.tempId, { likert_polarity: e.target.value || null })
                }
                className="h-8 text-sm"
              >
                <option value="">{t('questionEditor.option.selectPolarity')}</option>
                <option value="positive">{t('questionEditor.option.positive')}</option>
                <option value="negative">{t('questionEditor.option.negative')}</option>
              </Select>
            </div>
          </div>
        )}
      </div>
      {(questionType === 'single' || questionType === 'truefalse') && (
        <input
          type="radio"
          checked={option.is_correct}
          onChange={() => onToggleCorrect(option.tempId)}
          className="mt-3"
          aria-label={t('questionEditor.option.markCorrect')}
        />
      )}
      {questionType === 'multiple' && (
        <input
          type="checkbox"
          checked={option.is_correct}
          onChange={() => onToggleCorrect(option.tempId)}
          className="mt-3"
          aria-label={t('questionEditor.option.markCorrect')}
        />
      )}
      {!isLocked && (
        <button
          onClick={() => onDelete(option.tempId)}
          className="mt-2 text-muted-foreground hover:text-destructive"
          aria-label={t('common.removeOption')}
        >
          <X size={16} aria-hidden="true" />
        </button>
      )}
    </div>
  )
}

interface TagComboboxProps {
  tagIds: string[]
  onTagIdsChange: (ids: string[]) => void
}

function TagCombobox({ tagIds, onTagIdsChange }: TagComboboxProps) {
  const { t } = useTranslation()
  const { data: allTags = [] } = useTags()
  const createTag = useCreateTag()
  const [inputValue, setInputValue] = useState('')
  const [showDropdown, setShowDropdown] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const selectedTags = allTags.filter((tag) => (tagIds ?? []).includes(tag.id))
  const filtered = allTags.filter(
    (tag) =>
      !(tagIds ?? []).includes(tag.id) &&
      tag.name.toLowerCase().includes(inputValue.toLowerCase()),
  )

  async function handleSelect(id: string) {
    onTagIdsChange([...tagIds, id])
    setInputValue('')
    setShowDropdown(false)
  }

  async function handleCreate() {
    const name = inputValue.trim()
    if (!name) return
    try {
      const newTag = await createTag.mutateAsync({ name })
      onTagIdsChange([...tagIds, newTag.id])
      setInputValue('')
      setShowDropdown(false)
    } catch {
      // ignore
    }
  }

  function handleRemove(id: string) {
    onTagIdsChange(tagIds.filter((tid) => tid !== id))
  }

  return (
    <div className="relative">
      {/* Mouse convenience only: the inner <input> is the keyboard-focusable control. */}
      {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions */}
      <div className="flex flex-wrap gap-1.5 p-2 border rounded-md min-h-10 cursor-text" onClick={() => inputRef.current?.focus()}>
        {selectedTags.map((tag) => (
          <span
            key={tag.id}
            className="inline-flex items-center gap-1 bg-secondary text-secondary-foreground rounded-full px-2.5 py-0.5 text-xs"
          >
            {tag.name}
            <button
              onClick={() => handleRemove(tag.id)}
              className="hover:text-destructive"
              aria-label={`${t('common.remove')} ${tag.name}`}
            >
              <X size={10} aria-hidden="true" />
            </button>
          </span>
        ))}
        <input
          ref={inputRef}
          value={inputValue}
          onChange={(e) => {
            setInputValue(e.target.value)
            setShowDropdown(true)
          }}
          onFocus={() => setShowDropdown(true)}
          onBlur={() => setTimeout(() => setShowDropdown(false), 200)}
          placeholder={selectedTags.length === 0 ? t('questionEditor.tags.placeholder') : ''}
          className="outline-none bg-transparent text-sm flex-1 min-w-20"
        />
      </div>
      {showDropdown && (inputValue.length > 0 || filtered.length > 0) && (
        <div className="absolute top-full left-0 right-0 z-50 mt-1 border rounded-md bg-background shadow-lg max-h-48 overflow-y-auto">
          {filtered.map((tag) => (
            <button
              key={tag.id}
              onMouseDown={(e) => { e.preventDefault(); handleSelect(tag.id) }}
              className="w-full text-left px-3 py-1.5 text-sm hover:bg-muted"
            >
              {tag.name}
            </button>
          ))}
          {inputValue.trim() && !allTags.find((t) => t.name.toLowerCase() === inputValue.trim().toLowerCase()) && (
            <button
              onMouseDown={(e) => { e.preventDefault(); handleCreate() }}
              className="w-full text-left px-3 py-1.5 text-sm hover:bg-muted text-primary font-medium"
            >
              {t('questionEditor.tags.createNew', { name: inputValue.trim() })}
            </button>
          )}
        </div>
      )}
    </div>
  )
}

interface CategoryPickerProps {
  value: string
  onChange: (id: string) => void
  categories: Category[]
}

function CategoryPicker({ value, onChange, categories }: CategoryPickerProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const flat = flattenCategories(categories)

  const path = value ? getCategoryPath(categories, value) : ''

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <span className={path ? 'text-foreground' : 'text-muted-foreground'}>
          {path || t('questionEditor.category.placeholder')}
        </span>
        {open ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
      </button>
      {open && (
        <div className="absolute top-full left-0 right-0 z-50 mt-1 border rounded-md bg-background shadow-lg max-h-56 overflow-y-auto">
          {flat.map((cat) => {
            const depth = getCategoryPath(categories, cat.id).split(' › ').length - 1
            return (
              <button
                key={cat.id}
                onMouseDown={(e) => { e.preventDefault(); onChange(cat.id); setOpen(false) }}
                className={`w-full text-left px-3 py-1.5 text-sm hover:bg-muted ${
                  cat.id === value ? 'bg-muted font-medium' : ''
                }`}
                style={{ paddingLeft: `${12 + depth * 12}px` }}
              >
                {cat.name}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

interface QuestionPreviewProps {
  form: FormState
  activeLocale: Locale
}

function QuestionPreview({ form, activeLocale }: QuestionPreviewProps) {
  const { t } = useTranslation()
  const stem = form.translations[activeLocale]?.stem || ''
  const options = form.answer_options

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
        {t('questionEditor.preview.title')}
      </h3>
      <div className="p-4 border rounded-md bg-muted/30 min-h-32 space-y-3">
        {stem ? (
          <p className="text-sm font-medium">{stem}</p>
        ) : (
          <p className="text-sm text-muted-foreground italic">{t('questionEditor.preview.stemEmpty')}</p>
        )}
        {form.type !== 'shorttext' && options.length > 0 && (
          <ul className="space-y-1.5">
            {options.map((opt) => (
              <li
                key={opt.tempId}
                className={`flex items-center gap-2 text-sm px-3 py-1.5 rounded border ${
                  opt.is_correct ? 'border-green-400 bg-green-50' : 'border-border'
                }`}
              >
                {(form.type === 'single' || form.type === 'truefalse') && (
                  <span className="w-4 h-4 rounded-full border-2 border-muted-foreground flex-shrink-0" />
                )}
                {form.type === 'multiple' && (
                  <span className="w-4 h-4 rounded border-2 border-muted-foreground flex-shrink-0" />
                )}
                <span>{opt.translations[activeLocale]?.body || <em className="text-muted-foreground">{t('questionEditor.preview.optionEmpty')}</em>}</span>
              </li>
            ))}
          </ul>
        )}
        {form.type === 'shorttext' && (
          <div className="border rounded p-2 bg-background text-sm text-muted-foreground italic">
            {t('questionEditor.preview.shortTextPlaceholder')}
          </div>
        )}
      </div>
    </div>
  )
}

// ---- Main Page --------------------------------------------------------------

export function QuestionEditorPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id?: string }>()
  const isEditMode = Boolean(id)

  const { data: question, isLoading } = useQuestion(id ?? null)
  const { data: categories = [] } = useCategories()
  const createQuestion = useCreateQuestion()
  const updateQuestion = useUpdateQuestion()
  const updateStatus = useUpdateQuestionStatus()

  const [form, setForm] = useState<FormState>(() => makeEmptyForm())
  const [activeLocale, setActiveLocale] = useState<Locale>('en')
  const [explanationOpen, setExplanationOpen] = useState(false)
  const [isDirty, setIsDirty] = useState(false)
  const [autoSaveStatus, setAutoSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle')
  const [notification, setNotification] = useState<{ type: 'success' | 'error'; message: string } | null>(null)
  const autoSaveTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const dirtyRef = useRef(isDirty)
  dirtyRef.current = isDirty

  // Populate form from loaded question
  useEffect(() => {
    if (question) {
      setForm(detailToForm(question))
      setIsDirty(false)
    }
  }, [question])

  // Auto-save logic (edit mode only)
  const doAutoSave = useCallback(async () => {
    if (!id || !dirtyRef.current) return
    setAutoSaveStatus('saving')
    try {
      const updated = await updateQuestion.mutateAsync({ id, payload: buildUpdatePayload(form) })
      setIsDirty(false)
      setAutoSaveStatus('saved')
      // Active questions create a new version (new ID). Navigate so the page
      // stays on the live question instead of the now-archived original.
      if (updated.id !== id) {
        navigate(`/admin/questions/${updated.id}/edit`, { replace: true })
        return
      }
      setTimeout(() => setAutoSaveStatus('idle'), 3000)
    } catch {
      setAutoSaveStatus('error')
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, form, navigate])

  useEffect(() => {
    if (!isEditMode) return
    autoSaveTimerRef.current = setInterval(() => {
      doAutoSave()
    }, 30_000)
    return () => {
      if (autoSaveTimerRef.current) clearInterval(autoSaveTimerRef.current)
    }
  }, [isEditMode, doAutoSave])

  // ---- Form helpers ----

  function markDirty() {
    setIsDirty(true)
  }

  function updateFormField<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
    markDirty()
  }

  function updateTranslation(locale: Locale, field: 'stem' | 'explanation', value: string) {
    setForm((prev) => ({
      ...prev,
      translations: {
        ...prev.translations,
        [locale]: { ...prev.translations[locale], [field]: value },
      },
    }))
    markDirty()
  }

  function addOption() {
    setForm((prev) => ({
      ...prev,
      answer_options: [
        ...prev.answer_options,
        {
          tempId: crypto.randomUUID(),
          sort_order: prev.answer_options.length,
          is_correct: false,
          likert_weight: null,
          likert_polarity: null,
          translations: Object.fromEntries(LOCALES.map((l) => [l, { body: '' }])),
        },
      ],
    }))
    markDirty()
  }

  function deleteOption(tempId: string) {
    setForm((prev) => ({
      ...prev,
      answer_options: prev.answer_options.filter((o) => o.tempId !== tempId),
    }))
    markDirty()
  }

  function updateOption(tempId: string, updated: Partial<LocalAnswerOption>) {
    setForm((prev) => ({
      ...prev,
      answer_options: prev.answer_options.map((o) =>
        o.tempId === tempId ? { ...o, ...updated } : o,
      ),
    }))
    markDirty()
  }

  function updateOptionTranslation(tempId: string, locale: Locale, body: string) {
    setForm((prev) => ({
      ...prev,
      answer_options: prev.answer_options.map((o) =>
        o.tempId === tempId
          ? { ...o, translations: { ...o.translations, [locale]: { body } } }
          : o,
      ),
    }))
    markDirty()
  }

  function toggleCorrect(tempId: string) {
    setForm((prev) => {
      const isSingle = prev.type === 'single' || prev.type === 'truefalse'
      return {
        ...prev,
        answer_options: prev.answer_options.map((o) => {
          if (isSingle) {
            return { ...o, is_correct: o.tempId === tempId }
          }
          return o.tempId === tempId ? { ...o, is_correct: !o.is_correct } : o
        }),
      }
    })
    markDirty()
  }

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over || active.id === over.id) return
    setForm((prev) => {
      const ids = prev.answer_options.map((o) => o.tempId)
      const oldIndex = ids.indexOf(active.id as string)
      const newIndex = ids.indexOf(over.id as string)
      const reordered = arrayMove(prev.answer_options, oldIndex, newIndex).map((o, idx) => ({
        ...o,
        sort_order: idx,
      }))
      return { ...prev, answer_options: reordered }
    })
    markDirty()
  }

  // ---- Build payloads ----

  function buildUpdatePayload(f: FormState) {
    const translations: Record<string, { stem: string; explanation: string }> = {}
    for (const loc of LOCALES) {
      if (f.translations[loc]?.stem || f.translations[loc]?.explanation) {
        translations[loc] = {
          stem: f.translations[loc].stem,
          explanation: f.translations[loc].explanation,
        }
      }
    }
    return {
      difficulty: f.difficulty,
      category_id: f.category_id,
      auto_grade: f.auto_grade,
      model_answer: f.type === 'shorttext' ? (f.model_answer || null) : null,
      translations,
      answer_options: f.answer_options.map((o) => ({
        id: o.id,
        sort_order: o.sort_order,
        is_correct: o.is_correct,
        likert_weight: o.likert_weight ?? undefined,
        likert_polarity: o.likert_polarity ?? undefined,
        translations: Object.fromEntries(
          LOCALES.map((l) => [l, { text: o.translations[l]?.body ?? '' }]),
        ),
      })),
      tag_ids: f.tag_ids,
    }
  }

  // ---- Save / Submit actions ----

  async function handleSaveDraft() {
    if (isEditMode && id) {
      if (autoSaveTimerRef.current) clearInterval(autoSaveTimerRef.current)
      setAutoSaveStatus('saving')
      try {
        const updated = await updateQuestion.mutateAsync({ id, payload: buildUpdatePayload(form) })
        setIsDirty(false)
        setAutoSaveStatus('saved')
        // Active questions create a new version (new ID). Navigate so the page
        // stays on the live question instead of the now-archived original.
        if (updated.id !== id) {
          navigate(`/admin/questions/${updated.id}/edit`, { replace: true })
          return
        }
        setTimeout(() => setAutoSaveStatus('idle'), 3000)
      } catch {
        setAutoSaveStatus('error')
        setNotification({ type: 'error', message: t('questionEditor.error.saveFailed') })
      }
      return
    }

    // Create mode — use the active locale tab as the default_locale so the backend
    // validates the stem the user actually filled in (not always 'en').
    const defaultLocale: Locale =
      form.translations[activeLocale]?.stem?.trim()
        ? activeLocale
        : (LOCALES.find((l) => form.translations[l]?.stem?.trim()) ?? activeLocale)
    try {
      const createTranslations: Record<string, { stem: string; explanation?: string }> = {}
      for (const loc of LOCALES) {
        if (form.translations[loc]?.stem || form.translations[loc]?.explanation) {
          createTranslations[loc] = {
            stem: form.translations[loc].stem,
            ...(form.translations[loc].explanation ? { explanation: form.translations[loc].explanation } : {}),
          }
        }
      }
      if (!createTranslations[defaultLocale]) {
        createTranslations[defaultLocale] = { stem: form.translations[defaultLocale]?.stem ?? '' }
      }
      const created = await createQuestion.mutateAsync({
        type: form.type,
        difficulty: form.difficulty,
        category_id: form.category_id,
        default_locale: defaultLocale,
        auto_grade: form.type === 'shorttext' ? form.auto_grade : undefined,
        model_answer: form.type === 'shorttext' ? (form.model_answer || null) : undefined,
        translations: createTranslations,
        answer_options:
          form.type !== 'shorttext'
            ? form.answer_options.map((o) => ({
                sort_order: o.sort_order,
                is_correct: o.is_correct,
                likert_weight: o.likert_weight ?? undefined,
                likert_polarity: o.likert_polarity ?? undefined,
                translations: Object.fromEntries(
                  LOCALES.map((l) => [l, { text: o.translations[l]?.body ?? '' }]),
                ),
              }))
            : undefined,
        tag_ids: form.tag_ids.length > 0 ? form.tag_ids : undefined,
      })
      navigate(`/admin/questions/${created.id}/edit`, { replace: true })
    } catch {
      setNotification({ type: 'error', message: t('questionEditor.error.saveFailed') })
    }
  }

  async function handleStatusTransition(target: string) {
    if (!id) return
    try {
      await updateStatus.mutateAsync({ id, target_status: target })
      setNotification({ type: 'success', message: t('questionEditor.status.transitionSuccess') })
    } catch {
      setNotification({ type: 'error', message: t('questionEditor.error.statusTransitionFailed') })
    }
  }

  const currentStatus = question?.status

  // ---- Render ----

  if (isEditMode && isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <span className="text-muted-foreground text-sm">{t('questionBank.skeleton.loading')}</span>
      </div>
    )
  }

  const pageTitle = isEditMode
    ? form.translations[activeLocale]?.stem?.slice(0, 60) || t('questionEditor.title.edit')
    : t('questionEditor.title.new')

  const showAnswerOptions = form.type !== 'shorttext'
  const isLockedType = isEditMode

  return (
    <div className="flex flex-col h-full">
      {/* Notification */}
      {notification && (
        <div
          className={`flex items-center gap-3 px-4 py-3 text-sm border-b ${
            notification.type === 'error'
              ? 'bg-red-50 border-red-200 text-red-800'
              : 'bg-green-50 border-green-200 text-green-800'
          }`}
        >
          <span className="flex-1">{notification.message}</span>
          <button onClick={() => setNotification(null)} className="p-0.5 rounded hover:bg-black/10">
            <X size={14} />
          </button>
        </div>
      )}

      {/* Toolbar */}
      <div className="flex items-center gap-3 px-4 py-3 border-b bg-background sticky top-0 z-10">
        <button
          onClick={() => navigate('/admin/questions')}
          className="text-muted-foreground hover:text-foreground"
          aria-label={t('common.back')}
        >
          <ArrowLeft size={18} aria-hidden="true" />
        </button>
        <h1 className="text-sm font-semibold flex-1 truncate">{pageTitle}</h1>
        {currentStatus && <EditorStatusBadge status={currentStatus} />}
        <AutoSaveStatus status={autoSaveStatus} />

        {/* Status transition buttons */}
        {currentStatus === 'draft' && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleStatusTransition('review')}
            disabled={updateStatus.isPending}
          >
            {t('questionEditor.action.submitForReview')}
          </Button>
        )}
        {currentStatus === 'review' && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleStatusTransition('active')}
            disabled={updateStatus.isPending}
          >
            {t('questionEditor.action.approve')}
          </Button>
        )}
        {currentStatus === 'active' && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleStatusTransition('archived')}
            disabled={updateStatus.isPending}
          >
            {t('questionEditor.action.archive')}
          </Button>
        )}
        {currentStatus === 'archived' && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleStatusTransition('draft')}
            disabled={updateStatus.isPending}
          >
            {t('questionEditor.action.restoreToDraft')}
          </Button>
        )}

        <Button
          size="sm"
          onClick={handleSaveDraft}
          disabled={createQuestion.isPending || updateQuestion.isPending}
        >
          {isEditMode ? t('questionEditor.action.save') : t('questionEditor.action.saveDraft')}
        </Button>
      </div>

      {/* Locale tabs */}
      <div className="px-4 pt-3">
        <CoverageTabs
          activeLocale={activeLocale}
          onLocaleChange={setActiveLocale}
          translations={form.translations}
        />
      </div>

      {/* Two-column layout */}
      <div className="flex-1 grid grid-cols-1 lg:grid-cols-2 gap-0 overflow-hidden">
        {/* Left panel — form */}
        <div className="overflow-y-auto p-4 space-y-5 border-r">
          {/* Metadata section */}
          <section className="space-y-4">
            <h2 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
              {t('questionEditor.section.metadata')}
            </h2>

            {/* Question type */}
            <div className="space-y-1.5">
              <Label>{t('questionEditor.field.type')}</Label>
              <Select
                value={form.type}
                onChange={(e) => updateFormField('type', e.target.value as QuestionType)}
                disabled={isLockedType}
              >
                {(['single', 'multiple', 'truefalse', 'shorttext', 'likert'] as QuestionType[]).map(
                  (qt) => (
                    <option key={qt} value={qt}>
                      {t(`questionEditor.type.${qt}`)}
                    </option>
                  ),
                )}
              </Select>
            </div>

            {/* Difficulty */}
            <div className="space-y-1.5">
              <Label>{t('questionEditor.field.difficulty')}</Label>
              <Select
                value={form.difficulty}
                onChange={(e) => updateFormField('difficulty', e.target.value as Difficulty)}
              >
                {(['easy', 'medium', 'hard'] as Difficulty[]).map((d) => (
                  <option key={d} value={d}>
                    {t(`questionEditor.difficulty.${d}`)}
                  </option>
                ))}
              </Select>
            </div>

            {/* Category */}
            <div className="space-y-1.5">
              <Label>{t('questionEditor.field.category')}</Label>
              <CategoryPicker
                value={form.category_id}
                onChange={(id) => updateFormField('category_id', id)}
                categories={categories}
              />
            </div>

            {/* Tags */}
            <div className="space-y-1.5">
              <Label>{t('questionEditor.field.tags')}</Label>
              <TagCombobox
                tagIds={form.tag_ids}
                onTagIdsChange={(ids) => updateFormField('tag_ids', ids)}
              />
            </div>
          </section>

          {/* Translation section */}
          <section className="space-y-4">
            <h2 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
              {t('questionEditor.section.translation')} — {activeLocale.toUpperCase()}
            </h2>

            {/* Stem */}
            <div className="space-y-1.5">
              <Label>{t('questionEditor.field.stem')}</Label>
              <textarea
                value={form.translations[activeLocale]?.stem ?? ''}
                onChange={(e) => updateTranslation(activeLocale, 'stem', e.target.value)}
                rows={3}
                className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 resize-y"
                placeholder={t('questionEditor.field.stemPlaceholder')}
              />
            </div>

            {/* Explanation (collapsible) */}
            <div className="space-y-1.5">
              {!explanationOpen ? (
                <button
                  type="button"
                  onClick={() => setExplanationOpen(true)}
                  className="text-sm text-primary hover:underline"
                >
                  + {t('questionEditor.explanation.add')}
                </button>
              ) : (
                <>
                  <div className="flex items-center justify-between">
                    <Label>{t('questionEditor.field.explanation')}</Label>
                    <button
                      type="button"
                      onClick={() => setExplanationOpen(false)}
                      className="text-xs text-muted-foreground hover:text-foreground"
                    >
                      {t('questionEditor.explanation.hide')}
                    </button>
                  </div>
                  <textarea
                    value={form.translations[activeLocale]?.explanation ?? ''}
                    onChange={(e) => {
                      if (e.target.value.length <= 1000)
                        updateTranslation(activeLocale, 'explanation', e.target.value)
                    }}
                    rows={3}
                    maxLength={1000}
                    className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 resize-y"
                    placeholder={t('questionEditor.field.explanationPlaceholder')}
                  />
                  <p className="text-xs text-muted-foreground text-right">
                    {form.translations[activeLocale]?.explanation?.length ?? 0}/1000
                  </p>
                </>
              )}
            </div>
          </section>

          {/* Answer options section */}
          {showAnswerOptions && (
            <section className="space-y-3">
              <h2 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
                {t('questionEditor.section.answerOptions')}
              </h2>

              <DndContext
                sensors={sensors}
                collisionDetection={closestCenter}
                onDragEnd={handleDragEnd}
              >
                <SortableContext
                  items={form.answer_options.map((o) => o.tempId)}
                  strategy={verticalListSortingStrategy}
                >
                  <div className="space-y-2">
                    {form.answer_options.map((option) => (
                      <SortableAnswerOption
                        key={option.tempId}
                        option={option}
                        activeLocale={activeLocale}
                        questionType={form.type}
                        onUpdate={updateOption}
                        onUpdateTranslation={updateOptionTranslation}
                        onDelete={deleteOption}
                        onToggleCorrect={toggleCorrect}
                        t={t}
                      />
                    ))}
                  </div>
                </SortableContext>
              </DndContext>

              {form.type !== 'truefalse' && (
                <Button variant="outline" size="sm" onClick={addOption}>
                  + {t('questionEditor.addOption')}
                </Button>
              )}
            </section>
          )}

          {/* Auto-grading section — shorttext only */}
          {form.type === 'shorttext' && (
            <section className="space-y-3">
              <h2 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
                {t('questionEditor.section.autoGrading')}
              </h2>
              <div className="space-y-1.5">
                <Label htmlFor="model-answer">{t('questionEditor.field.modelAnswer')}</Label>
                <textarea
                  id="model-answer"
                  value={form.model_answer}
                  onChange={(e) => {
                    const val = e.target.value
                    updateFormField('model_answer', val)
                    // If model answer is cleared, disable auto_grade.
                    if (!val.trim()) updateFormField('auto_grade', false)
                  }}
                  rows={3}
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 resize-y"
                  placeholder={t('questionEditor.field.modelAnswerPlaceholder')}
                />
              </div>
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="auto-grade-toggle"
                  checked={form.auto_grade}
                  disabled={!form.model_answer.trim()}
                  onChange={(e) => updateFormField('auto_grade', e.target.checked)}
                  className="h-4 w-4 cursor-pointer disabled:cursor-not-allowed"
                />
                <Label htmlFor="auto-grade-toggle">{t('questionEditor.field.autoGrade')}</Label>
              </div>
              <p className="text-xs text-muted-foreground">
                {t('questionEditor.field.autoGradeHelp')}
              </p>
            </section>
          )}
        </div>

        {/* Right panel — preview */}
        <div className="overflow-y-auto p-4 bg-muted/10">
          <QuestionPreview form={form} activeLocale={activeLocale} />
        </div>
      </div>
    </div>
  )
}
