import { useState, useRef, useCallback, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useDebounceValue } from 'usehooks-ts'
import { useQueryClient } from '@tanstack/react-query'
import { downloadFile, downloadErrorKey } from '@/api/download'
import {
  Search,
  Plus,
  Upload,
  Sparkles,
  MoreVertical,
  ChevronUp,
  ChevronDown,
  CheckCircle2,
  XCircle,
  ChevronLeft,
  ChevronRight,
  AlertCircle,
  X,
  Clock,
  Trash2,
  Edit,
  Archive,
} from 'lucide-react'
import { AIGenerateDialog } from '@/components/questions/AIGenerateDialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
  TableCell,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Select } from '@/components/ui/select'
import {
  useQuestions,
  useCategories,
  useTags,
  useQuestionVersions,
  useTransitionStatus,
  useDeleteQuestion,
  useImportQuestions,
  type QuestionListItem,
  type QuestionFilters,
  type Category,
  type Tag,
  type ImportDryRunResult,
} from '@/api/questions'

// ---- Helpers ----------------------------------------------------------------

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffSec = Math.floor(diffMs / 1000)
  const diffMin = Math.floor(diffSec / 60)
  const diffHour = Math.floor(diffMin / 60)
  const diffDay = Math.floor(diffHour / 24)
  if (diffDay > 30) return date.toLocaleDateString()
  if (diffDay > 0) return `${diffDay}d ago`
  if (diffHour > 0) return `${diffHour}h ago`
  if (diffMin > 0) return `${diffMin}m ago`
  return 'just now'
}

function flattenCategories(categories: Category[]): Category[] {
  const result: Category[] = []
  function walk(cats: Category[]) {
    for (const cat of cats) {
      result.push(cat)
      if (cat.children?.length) walk(cat.children)
    }
  }
  walk(categories)
  return result
}

// ---- Sub-components ---------------------------------------------------------

function LoadingSkeleton() {
  return (
    <div className="space-y-2 animate-pulse">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="h-12 bg-muted rounded" />
      ))}
    </div>
  )
}

interface NotificationBannerProps {
  type: 'success' | 'error'
  message: string
  onDismiss: () => void
}

function NotificationBanner({ type, message, onDismiss }: NotificationBannerProps) {
  return (
    <div
      className={`flex items-center gap-3 px-4 py-3 rounded-md text-sm ${
        type === 'error'
          ? 'bg-red-50 border border-red-200 text-red-800'
          : 'bg-green-50 border border-green-200 text-green-800'
      }`}
    >
      <AlertCircle size={16} className="flex-shrink-0" />
      <span className="flex-1">{message}</span>
      <button onClick={onDismiss} className="p-0.5 rounded hover:bg-black/10">
        <X size={14} />
      </button>
    </div>
  )
}

interface DifficultyBadgeProps {
  difficulty: QuestionListItem['difficulty']
}

function DifficultyBadge({ difficulty }: DifficultyBadgeProps) {
  const { t } = useTranslation()
  const variant =
    difficulty === 'easy' ? 'success' : difficulty === 'medium' ? 'warning' : 'destructive'
  return (
    <Badge variant={variant}>{t(`questionBank.difficulty.${difficulty}`)}</Badge>
  )
}

interface StatusBadgeProps {
  status: QuestionListItem['status']
}

function QuestionStatusBadge({ status }: StatusBadgeProps) {
  const { t } = useTranslation()
  const variant =
    status === 'active'
      ? 'success'
      : status === 'draft'
        ? 'secondary'
        : status === 'review'
          ? 'warning'
          : 'outline'
  return <Badge variant={variant}>{t(`questionBank.status.${status}`)}</Badge>
}

interface TypeBadgeProps {
  type: QuestionListItem['type']
}

function TypeBadge({ type }: TypeBadgeProps) {
  const { t } = useTranslation()
  return <Badge variant="outline">{t(`questionBank.type.${type}`)}</Badge>
}

interface LocaleCoverageIconsProps {
  coverage: string[]
  allLocales?: string[]
}

function LocaleCoverageIcons({ coverage, allLocales = ['en', 'kk', 'ru'] }: LocaleCoverageIconsProps) {
  const { t } = useTranslation()
  return (
    <div className="flex gap-1">
      {allLocales.map((locale) => {
        const present = coverage.includes(locale)
        return present ? (
          <CheckCircle2
            key={locale}
            size={14}
            className="text-green-600"
            aria-label={t('questionBank.localeCoverage.present', { locale: locale.toUpperCase() })}
          />
        ) : (
          <XCircle
            key={locale}
            size={14}
            className="text-red-400"
            aria-label={t('questionBank.localeCoverage.missing', { locale: locale.toUpperCase() })}
          />
        )
      })}
    </div>
  )
}

// ---- Row Actions Dropdown ---------------------------------------------------

interface RowActionsProps {
  question: QuestionListItem
  onEdit: () => void
  onArchive: () => void
  onViewVersions: () => void
  onDelete: () => void
}

function RowActionsMenu({ question, onEdit, onArchive, onViewVersions, onDelete }: RowActionsProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [open])

  const canArchive = question.status !== 'archived'
  const canDelete = question.status === 'draft'

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((o) => !o)}
        className="p-1 rounded hover:bg-muted"
        aria-label={t('common.rowActions')}
      >
        <MoreVertical size={16} aria-hidden="true" />
      </button>
      {open && (
        <div className="absolute right-0 z-20 mt-1 w-44 bg-background border rounded-md shadow-lg py-1 text-sm">
          <button
            onClick={() => { setOpen(false); onEdit() }}
            className="w-full flex items-center gap-2 px-3 py-1.5 hover:bg-muted"
          >
            <Edit size={14} />
            {t('questionBank.actions.edit')}
          </button>
          <button
            onClick={() => { setOpen(false); onViewVersions() }}
            className="w-full flex items-center gap-2 px-3 py-1.5 hover:bg-muted"
          >
            <Clock size={14} />
            {t('questionBank.actions.viewVersions')}
          </button>
          {canArchive && (
            <button
              onClick={() => { setOpen(false); onArchive() }}
              className="w-full flex items-center gap-2 px-3 py-1.5 hover:bg-muted"
            >
              <Archive size={14} />
              {t('questionBank.actions.archive')}
            </button>
          )}
          {canDelete && (
            <button
              onClick={() => { setOpen(false); onDelete() }}
              className="w-full flex items-center gap-2 px-3 py-1.5 hover:bg-muted text-red-600"
            >
              <Trash2 size={14} />
              {t('questionBank.actions.delete')}
            </button>
          )}
        </div>
      )}
    </div>
  )
}

// ---- Status Transition Dropdown ---------------------------------------------

interface StatusTransitionProps {
  question: QuestionListItem
  onTransition: (id: string, target: string) => void
}

const STATUS_TRANSITIONS: Record<QuestionListItem['status'], QuestionListItem['status'][]> = {
  draft: ['review'],
  review: ['active', 'draft'],
  active: ['archived'],
  archived: ['draft'],
}

function StatusTransitionCell({ question, onTransition }: StatusTransitionProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const transitions = STATUS_TRANSITIONS[question.status]

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [open])

  return (
    <div ref={ref} className="relative inline-block">
      <button onClick={() => setOpen((o) => !o)} className="focus:outline-none">
        <QuestionStatusBadge status={question.status} />
      </button>
      {open && transitions.length > 0 && (
        <div className="absolute left-0 z-20 mt-1 w-36 bg-background border rounded-md shadow-lg py-1 text-sm">
          {transitions.map((target) => (
            <button
              key={target}
              onClick={() => { setOpen(false); onTransition(question.id, target) }}
              className="w-full px-3 py-1.5 text-left hover:bg-muted"
            >
              → {t(`questionBank.status.${target}`)}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

// ---- Tag Multi-Select -------------------------------------------------------

interface TagMultiSelectProps {
  tags: Tag[]
  selectedIds: string[]
  onChange: (ids: string[]) => void
  placeholder: string
}

function TagMultiSelect({ tags, selectedIds, onChange, placeholder }: TagMultiSelectProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const selectedCount = selectedIds.length

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [open])

  function toggle(id: string) {
    if (selectedIds.includes(id)) {
      onChange(selectedIds.filter((x) => x !== id))
    } else {
      onChange([...selectedIds, id])
    }
  }

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((o) => !o)}
        className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
      >
        <span className={selectedCount === 0 ? 'text-muted-foreground' : ''}>
          {selectedCount === 0 ? placeholder : `${selectedCount} selected`}
        </span>
        <ChevronDown size={14} className="text-muted-foreground" />
      </button>
      {open && (
        <div className="absolute left-0 z-20 mt-1 w-56 max-h-60 overflow-y-auto bg-background border rounded-md shadow-lg py-1 text-sm">
          {tags.map((tag) => (
            <label
              key={tag.id}
              className="flex items-center gap-2 px-3 py-1.5 hover:bg-muted cursor-pointer"
            >
              <input
                type="checkbox"
                checked={selectedIds.includes(tag.id)}
                onChange={() => toggle(tag.id)}
                className="rounded"
              />
              {tag.name}
            </label>
          ))}
        </div>
      )}
    </div>
  )
}

// ---- Multi-Select Chips (for difficulty/status) -----------------------------

interface ChipsMultiSelectProps<T extends string> {
  options: { value: T; label: string }[]
  selected: T[]
  onChange: (values: T[]) => void
}

function ChipsMultiSelect<T extends string>({
  options,
  selected,
  onChange,
}: ChipsMultiSelectProps<T>) {
  function toggle(value: T) {
    if (selected.includes(value)) {
      onChange(selected.filter((v) => v !== value))
    } else {
      onChange([...selected, value])
    }
  }

  return (
    <div className="flex gap-1.5 flex-wrap">
      {options.map(({ value, label }) => (
        <button
          key={value}
          onClick={() => toggle(value)}
          className={`px-3 py-1 rounded-full text-xs font-medium border transition-colors ${
            selected.includes(value)
              ? 'bg-primary text-primary-foreground border-primary'
              : 'bg-background text-foreground border-input hover:bg-muted'
          }`}
        >
          {label}
        </button>
      ))}
    </div>
  )
}

// ---- Version History Slideover ----------------------------------------------

interface VersionHistorySlideoverProps {
  questionId: string | null
  onClose: () => void
}

function VersionHistorySlideover({ questionId, onClose }: VersionHistorySlideoverProps) {
  const { t } = useTranslation()
  const { data: versions, isLoading } = useQuestionVersions(questionId ?? '')

  return (
    <Sheet open={!!questionId} onOpenChange={(open) => !open && onClose()}>
      <SheetContent side="right">
        <SheetHeader>
          <SheetTitle>{t('questionBank.versions.title')}</SheetTitle>
        </SheetHeader>
        <div className="mt-4">
          {isLoading && (
            <div className="space-y-2 animate-pulse">
              {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="h-16 bg-muted rounded" />
              ))}
            </div>
          )}
          {versions && versions.length === 0 && (
            <p className="text-sm text-muted-foreground">{t('questionBank.versions.empty')}</p>
          )}
          {versions && versions.length > 0 && (
            <div className="space-y-3">
              {versions.map((v) => (
                <div
                  key={v.version}
                  className="border rounded-md p-3 text-sm space-y-1"
                >
                  <div className="flex items-center justify-between">
                    <span className="font-medium">{t('questionBank.versions.label', { version: v.version })}</span>
                    <span className="text-muted-foreground text-xs">
                      {formatRelativeTime(v.created_at)}
                    </span>
                  </div>
                  <p className="text-muted-foreground">{v.created_by_name}</p>
                  {v.change_summary && (
                    <p className="text-xs text-foreground">{v.change_summary}</p>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

// ---- Import Modal -----------------------------------------------------------

interface ImportModalProps {
  open: boolean
  onClose: () => void
  onSuccess: () => void
}

function ImportModal({ open, onClose, onSuccess }: ImportModalProps) {
  const { t } = useTranslation()
  const importMutation = useImportQuestions()
  const [file, setFile] = useState<File | null>(null)
  const [dryRunResult, setDryRunResult] = useState<ImportDryRunResult | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  function reset() {
    setFile(null)
    setDryRunResult(null)
    importMutation.reset()
  }

  function handleClose() {
    reset()
    onClose()
  }

  async function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0]
    if (!f) return
    setFile(f)
    setDryRunResult(null)
    try {
      const result = await importMutation.mutateAsync({ file: f, dryRun: true })
      setDryRunResult(result)
    } catch {
      // error shown via importMutation.error
    }
  }

  async function handleConfirm() {
    if (!file) return
    try {
      await importMutation.mutateAsync({ file, dryRun: false })
      onSuccess()
      handleClose()
    } catch {
      // error shown via importMutation.error
    }
  }

  const hasErrors = (dryRunResult?.error_rows.length ?? 0) > 0
  const hasWarnings = (dryRunResult?.warning_rows.length ?? 0) > 0

  return (
    <Dialog open={open} onOpenChange={(o) => !o && handleClose()}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t('questionBank.import.title')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* File picker */}
          <div
            role="button"
            tabIndex={0}
            className="border-2 border-dashed border-input rounded-md p-6 text-center cursor-pointer hover:border-primary transition-colors"
            onClick={() => fileInputRef.current?.click()}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                fileInputRef.current?.click()
              }
            }}
          >
            <Upload size={24} className="mx-auto mb-2 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              {file ? file.name : t('questionBank.import.dropzone')}
            </p>
            <input
              ref={fileInputRef}
              type="file"
              accept=".csv,.json"
              className="hidden"
              onChange={handleFileChange}
            />
          </div>

          {/* Loading state */}
          {importMutation.isPending && !dryRunResult && (
            <p className="text-sm text-muted-foreground animate-pulse">
              {t('questionBank.skeleton.loading')}
            </p>
          )}

          {/* Error from mutation */}
          {importMutation.isError && (
            <p className="text-sm text-red-600">{importMutation.error?.message}</p>
          )}

          {/* Dry run results */}
          {dryRunResult && (
            <div className="space-y-3">
              <p className="text-sm font-medium">
                {t('questionBank.import.validCount', { count: dryRunResult.valid_count })}
              </p>

              {hasErrors && (
                <div>
                  <p className="text-sm font-medium text-red-700 mb-1">
                    {t('questionBank.import.errors')}
                  </p>
                  <div className="max-h-36 overflow-y-auto space-y-1">
                    {dryRunResult.error_rows.map((row) => (
                      <div
                        key={row.row}
                        className="text-xs bg-red-50 border border-red-200 rounded px-2 py-1.5"
                      >
                        <span className="font-medium">{t('questionBank.import.row', { row: row.row })}</span>{' '}
                        {row.errors.join(', ')}
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {hasWarnings && (
                <div>
                  <p className="text-sm font-medium text-yellow-700 mb-1">
                    {t('questionBank.import.warnings')}
                  </p>
                  <div className="max-h-36 overflow-y-auto space-y-1">
                    {dryRunResult.warning_rows.map((row) => (
                      <div
                        key={row.row}
                        className="text-xs bg-yellow-50 border border-yellow-200 rounded px-2 py-1.5"
                      >
                        <span className="font-medium">{t('questionBank.import.row', { row: row.row })}</span>{' '}
                        {t('questionBank.import.similarTo', {
                          stem: row.similarity_match.stem_preview,
                          score: Math.round(row.similarity_match.score * 100),
                        })}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            {t('questionBank.import.cancel')}
          </Button>
          {dryRunResult && (
            <Button
              onClick={handleConfirm}
              disabled={importMutation.isPending}
            >
              {t('questionBank.import.confirm')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---- Delete Confirm Dialog --------------------------------------------------

interface DeleteConfirmDialogProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  isPending: boolean
}

function DeleteConfirmDialog({ open, onClose, onConfirm, isPending }: DeleteConfirmDialogProps) {
  const { t } = useTranslation()
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('questionBank.confirmDelete.title')}</DialogTitle>
          <DialogDescription>{t('questionBank.confirmDelete.description')}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isPending}>
            {t('questionBank.confirmDelete.cancel')}
          </Button>
          <Button variant="destructive" onClick={onConfirm} disabled={isPending}>
            {t('questionBank.confirmDelete.confirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---- Pagination -------------------------------------------------------------

interface PaginationProps {
  page: number
  total: number
  perPage: number
  onPageChange: (page: number) => void
  onPerPageChange: (perPage: number) => void
}

function Pagination({ page, total, perPage, onPageChange, onPerPageChange }: PaginationProps) {
  const { t } = useTranslation()
  const totalPages = Math.max(1, Math.ceil(total / perPage))

  return (
    <div className="flex items-center justify-between text-sm">
      <div className="flex items-center gap-2">
        <span className="text-muted-foreground">{t('questionBank.pagination.pageSize')}</span>
        <Select
          className="w-20"
          value={String(perPage)}
          onChange={(e) => onPerPageChange(Number(e.target.value))}
        >
          {[20, 50, 100].map((n) => (
            <option key={n} value={n}>
              {n}
            </option>
          ))}
        </Select>
      </div>
      <div className="flex items-center gap-2">
        <span className="text-muted-foreground">
          {t('questionBank.pagination.pageOf', { page, total: totalPages })}
        </span>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
        >
          <ChevronLeft size={14} />
          {t('questionBank.pagination.previous')}
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
        >
          {t('questionBank.pagination.next')}
          <ChevronRight size={14} />
        </Button>
      </div>
    </div>
  )
}

// ---- Bulk Action Bar --------------------------------------------------------

interface BulkActionBarProps {
  selectedIds: string[]
  allItems: QuestionListItem[]
  filters: QuestionFilters
  onArchive: () => void
  onClearSelection: () => void
  archiveProgress: number | null
}

function BulkActionBar({
  selectedIds,
  allItems: _allItems,
  filters,
  onArchive,
  onClearSelection,
  archiveProgress,
}: BulkActionBarProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [exportError, setExportError] = useState<string | null>(null)
  const count = selectedIds.length

  function buildExportUrl(format: 'csv' | 'json') {
    const params = new URLSearchParams()
    if (count > 100) {
      // use current filter state
      if (filters.statuses?.length) params.set('statuses', filters.statuses.join(','))
      if (filters.difficulties?.length) params.set('difficulties', filters.difficulties.join(','))
      if (filters.category_id) params.set('category_id', filters.category_id)
      if (filters.tag_ids?.length) params.set('tag_ids', filters.tag_ids.join(','))
      if (filters.search) params.set('search', filters.search)
    } else {
      params.set('ids', selectedIds.join(','))
    }
    params.set('format', format)
    return `/api/v1/questions/export?${params.toString()}`
  }

  async function handleExport(format: 'csv' | 'json') {
    setExportError(null)
    try {
      await downloadFile(qc, buildExportUrl(format), `questions.${format}`)
    } catch (err) {
      setExportError(downloadErrorKey(err))
    }
  }

  return (
    <div className="fixed bottom-0 left-0 right-0 z-30 bg-background border-t shadow-lg px-6 py-3">
      <div className="flex items-center gap-4 max-w-7xl mx-auto">
        <span className="text-sm font-medium">
          {t('questionBank.bulk.selected', { count })}
        </span>
        {exportError && (
          <span role="alert" className="text-xs text-red-700">
            {t(exportError)}
          </span>
        )}
        {count > 100 && (
          <span className="text-xs text-muted-foreground">
            {t('questionBank.bulk.exportLimit')}
          </span>
        )}
        <Button variant="outline" size="sm" onClick={() => handleExport('csv')}>
          {t('questionBank.bulk.exportCsv')}
        </Button>
        <Button variant="outline" size="sm" onClick={() => handleExport('json')}>
          {t('questionBank.bulk.exportJson')}
        </Button>
        <Button variant="outline" size="sm" onClick={onArchive} disabled={archiveProgress !== null}>
          {archiveProgress !== null
            ? t('questionBank.import.progress', { n: archiveProgress, total: count })
            : t('questionBank.bulk.archive')}
        </Button>
        <button
          onClick={onClearSelection}
          className="ml-auto p-1 rounded hover:bg-muted text-muted-foreground"
        >
          <X size={16} />
        </button>
      </div>
    </div>
  )
}

// ---- Main Page --------------------------------------------------------------

type SortColumn = 'created_at' | 'updated_at' | 'difficulty'
type SortOrder = 'asc' | 'desc'

function parseCSV(params: URLSearchParams, key: string): string[] {
  const val = params.get(key)
  return val ? val.split(',').filter(Boolean) : []
}

interface CurrentUser {
  role: string
}

function isExaminerOrAbove(role: string): boolean {
  return ['examiner', 'department_admin', 'super_admin'].includes(role)
}

export function QuestionBankPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const currentUser = qc.getQueryData<CurrentUser>(['auth', 'currentUser'])
  const canUseAI = currentUser ? isExaminerOrAbove(currentUser.role) : false
  const [searchParams, setSearchParams] = useSearchParams()

  // Derive filter state from URL
  const page = Number(searchParams.get('page') ?? '1')
  const perPage = Number(searchParams.get('per_page') ?? '20')
  const sort = (searchParams.get('sort') as SortColumn) || 'updated_at'
  const order = (searchParams.get('order') as SortOrder) || 'desc'
  const categoryId = searchParams.get('category_id') ?? undefined
  const tagIds = parseCSV(searchParams, 'tag_ids')
  const statuses = parseCSV(searchParams, 'statuses') as QuestionListItem['status'][]
  const difficulties = parseCSV(searchParams, 'difficulties') as QuestionListItem['difficulty'][]
  const type = (searchParams.get('type') as QuestionListItem['type']) || undefined
  const localeMissing = searchParams.get('locale_missing') ?? undefined
  const searchParam = searchParams.get('search') ?? ''

  // Local search state for debounce
  const [searchInput, setSearchInput] = useState(searchParam)
  const [debouncedSearch] = useDebounceValue(searchInput, 300)

  // Sync debounced search to URL
  const isFirstRender = useRef(true)
  useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false
      return
    }
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (debouncedSearch) {
        next.set('search', debouncedSearch)
      } else {
        next.delete('search')
      }
      next.set('page', '1')
      return next
    }, { replace: true })
  }, [debouncedSearch]) // eslint-disable-line react-hooks/exhaustive-deps

  function updateParam(key: string, value: string | undefined) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (value) next.set(key, value); else next.delete(key)
      next.set('page', '1')
      return next
    }, { replace: true })
  }

  function updateParams(updates: Record<string, string | undefined>) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      for (const [key, value] of Object.entries(updates)) {
        if (value) next.set(key, value); else next.delete(key)
      }
      next.set('page', '1')
      return next
    }, { replace: true })
  }

  function clearFilters() {
    setSearchInput('')
    setSearchParams(new URLSearchParams({ page: '1', per_page: String(perPage) }), {
      replace: true,
    })
  }

  function handleSort(col: SortColumn) {
    const newOrder = sort === col && order === 'asc' ? 'desc' : 'asc'
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set('sort', col)
      next.set('order', newOrder)
      return next
    }, { replace: true })
  }

  function sortIcon(col: SortColumn) {
    if (sort !== col) return null
    return order === 'asc' ? (
      <ChevronUp size={14} className="inline ml-0.5" />
    ) : (
      <ChevronDown size={14} className="inline ml-0.5" />
    )
  }

  // Build filters object for API
  const filters: QuestionFilters = {
    page,
    per_page: perPage,
    sort,
    order,
    category_id: categoryId,
    tag_ids: tagIds.length ? tagIds : undefined,
    statuses: statuses.length ? statuses : undefined,
    difficulties: difficulties.length ? difficulties : undefined,
    type,
    locale_missing: localeMissing,
    search: debouncedSearch || undefined,
  }

  // Data queries
  const { data, isLoading, isError } = useQuestions(filters)
  const { data: categories } = useCategories()
  const { data: tags } = useTags()

  // Mutations
  const transitionMutation = useTransitionStatus()
  const deleteMutation = useDeleteQuestion()

  // Selection state
  const [selectedIds, setSelectedIds] = useState<string[]>([])

  // UI state
  const [versionsQuestionId, setVersionsQuestionId] = useState<string | null>(null)
  const [importOpen, setImportOpen] = useState(false)
  const [aiGenerateOpen, setAIGenerateOpen] = useState(false)
  const [deleteTargetId, setDeleteTargetId] = useState<string | null>(null)
  const [notification, setNotification] = useState<{
    type: 'success' | 'error'
    message: string
  } | null>(null)
  const [archiveProgress, setArchiveProgress] = useState<number | null>(null)

  function showNotification(type: 'success' | 'error', message: string) {
    setNotification({ type, message })
    setTimeout(() => setNotification(null), 5000)
  }

  const items = data?.items ?? []
  const allPageSelected =
    items.length > 0 && items.every((item) => selectedIds.includes(item.id))

  function toggleSelectAll() {
    if (allPageSelected) {
      setSelectedIds((prev) => prev.filter((id) => !items.find((item) => item.id === id)))
    } else {
      setSelectedIds((prev) => {
        const newIds = items.map((i) => i.id).filter((id) => !prev.includes(id))
        return [...prev, ...newIds]
      })
    }
  }

  function toggleRow(id: string) {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    )
  }

  async function handleStatusTransition(id: string, targetStatus: string) {
    try {
      await transitionMutation.mutateAsync({ id, target_status: targetStatus })
    } catch (err) {
      showNotification(
        'error',
        `${t('questionBank.error.statusTransitionFailed')}: ${(err as Error).message}`,
      )
    }
  }

  async function handleBulkArchive() {
    setArchiveProgress(0)
    const results = await Promise.allSettled(
      selectedIds.map((id) => transitionMutation.mutateAsync({ id, target_status: 'archived' })),
    )
    setArchiveProgress(null)
    const success = results.filter((r) => r.status === 'fulfilled').length
    const failed = results.filter((r) => r.status === 'rejected').length
    showNotification(
      failed === 0 ? 'success' : 'error',
      t('questionBank.bulk.archivePartialResult', { success, failed }),
    )
    if (success > 0) {
      setSelectedIds((prev) =>
        prev.filter((_, i) => results[i]?.status !== 'fulfilled'),
      )
    }
  }

  async function handleDelete() {
    if (!deleteTargetId) return
    try {
      await deleteMutation.mutateAsync(deleteTargetId)
      setDeleteTargetId(null)
    } catch (err) {
      showNotification(
        'error',
        `${t('questionBank.error.deleteFailed')}: ${(err as Error).message}`,
      )
    }
  }

  const flatCategories = useCallback(
    () => (categories ? flattenCategories(categories) : []),
    [categories],
  )()

  const difficultyOptions: { value: QuestionListItem['difficulty']; label: string }[] = [
    { value: 'easy', label: t('questionBank.difficulty.easy') },
    { value: 'medium', label: t('questionBank.difficulty.medium') },
    { value: 'hard', label: t('questionBank.difficulty.hard') },
  ]

  const statusOptions: { value: QuestionListItem['status']; label: string }[] = [
    { value: 'draft', label: t('questionBank.status.draft') },
    { value: 'review', label: t('questionBank.status.review') },
    { value: 'active', label: t('questionBank.status.active') },
    { value: 'archived', label: t('questionBank.status.archived') },
  ]

  const hasActiveFilters =
    !!categoryId ||
    tagIds.length > 0 ||
    statuses.length > 0 ||
    difficulties.length > 0 ||
    !!type ||
    !!localeMissing ||
    !!debouncedSearch

  return (
    <div className="space-y-4 pb-20">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t('questionBank.title')}</h1>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setImportOpen(true)}>
            <Upload size={16} className="mr-1" />
            {t('questionBank.importButton')}
          </Button>
          {canUseAI && (
            <Button variant="outline" onClick={() => setAIGenerateOpen(true)}>
              <Sparkles size={16} className="mr-1" />
              {t('question.aiGenerate')}
            </Button>
          )}
          <Button onClick={() => navigate('/admin/questions/new')}>
            <Plus size={16} className="mr-1" />
            {t('questionBank.newQuestion')}
          </Button>
        </div>
      </div>

      {/* Notification banner */}
      {notification && (
        <NotificationBanner
          type={notification.type}
          message={notification.message}
          onDismiss={() => setNotification(null)}
        />
      )}

      {/* Filter bar */}
      <div className="space-y-3">
        {/* Search */}
        <div className="relative">
          <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder={t('questionBank.filter.search')}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
          />
        </div>

        {/* Filters row */}
        <div className="flex gap-3 flex-wrap items-center">
          {/* Category */}
          <Select
            className="w-44"
            value={categoryId ?? ''}
            onChange={(e) => updateParam('category_id', e.target.value || undefined)}
          >
            <option value="">{t('questionBank.filter.allCategories')}</option>
            {flatCategories.map((cat) => (
              <option key={cat.id} value={cat.id}>
                {cat.name}
              </option>
            ))}
          </Select>

          {/* Type */}
          <Select
            className="w-40"
            value={type ?? ''}
            onChange={(e) =>
              updateParam('type', (e.target.value as QuestionListItem['type']) || undefined)
            }
          >
            <option value="">{t('questionBank.filter.allTypes')}</option>
            <option value="single">{t('questionBank.type.single')}</option>
            <option value="multiple">{t('questionBank.type.multiple')}</option>
            <option value="truefalse">{t('questionBank.type.truefalse')}</option>
            <option value="shorttext">{t('questionBank.type.shorttext')}</option>
            <option value="likert">{t('questionBank.type.likert')}</option>
          </Select>

          {/* Locale missing */}
          <Select
            className="w-44"
            value={localeMissing ?? ''}
            onChange={(e) => updateParam('locale_missing', e.target.value || undefined)}
          >
            <option value="">{t('questionBank.filter.allLocales')}</option>
            <option value="kk">KK</option>
            <option value="ru">RU</option>
            <option value="en">EN</option>
          </Select>

          {/* Tag multi-select */}
          {tags && tags.length > 0 && (
            <div className="w-44">
              <TagMultiSelect
                tags={tags}
                selectedIds={tagIds}
                onChange={(ids) =>
                  updateParam('tag_ids', ids.length ? ids.join(',') : undefined)
                }
                placeholder={t('questionBank.filter.tag')}
              />
            </div>
          )}

          {/* Clear filters */}
          {hasActiveFilters && (
            <Button variant="ghost" size="sm" onClick={clearFilters}>
              <X size={14} className="mr-1" />
              {t('questionBank.filter.clearAll')}
            </Button>
          )}
        </div>

        {/* Difficulty chips */}
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted-foreground">{t('questionBank.filter.difficulty')}:</span>
          <ChipsMultiSelect
            options={difficultyOptions}
            selected={difficulties}
            onChange={(vals) =>
              updateParam('difficulties', vals.length ? vals.join(',') : undefined)
            }
          />
        </div>

        {/* Status chips */}
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted-foreground">{t('questionBank.filter.status')}:</span>
          <ChipsMultiSelect
            options={statusOptions}
            selected={statuses}
            onChange={(vals) =>
              updateParam('statuses', vals.length ? vals.join(',') : undefined)
            }
          />
        </div>
      </div>

      {/* Table */}
      {isLoading && <LoadingSkeleton />}
      {isError && (
        <p className="text-red-600 text-sm">{t('questionBank.error.loadFailed')}</p>
      )}
      {!isLoading && !isError && data && (
        <>
          {data.items.length === 0 ? (
            <div className="text-center py-16 space-y-2">
              <p className="text-lg font-medium">{t('questionBank.empty.title')}</p>
              <p className="text-muted-foreground text-sm">
                {t('questionBank.empty.description')}
              </p>
              {hasActiveFilters && (
                <Button variant="outline" size="sm" onClick={clearFilters}>
                  {t('questionBank.filter.clearAll')}
                </Button>
              )}
            </div>
          ) : (
            <div className="overflow-x-auto rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-10">
                      <input
                        type="checkbox"
                        checked={allPageSelected}
                        onChange={toggleSelectAll}
                        className="rounded"
                        aria-label="Select all"
                      />
                    </TableHead>
                    <TableHead
                      className="cursor-pointer select-none min-w-64"
                      onClick={() => handleSort('updated_at')}
                    >
                      {t('questionBank.column.stem')}
                    </TableHead>
                    <TableHead>{t('questionBank.column.type')}</TableHead>
                    <TableHead
                      className="cursor-pointer select-none"
                      onClick={() => handleSort('difficulty')}
                    >
                      {t('questionBank.column.difficulty')}
                      {sortIcon('difficulty')}
                    </TableHead>
                    <TableHead>{t('questionBank.column.category')}</TableHead>
                    <TableHead>{t('questionBank.column.status')}</TableHead>
                    <TableHead>{t('questionBank.column.coverage')}</TableHead>
                    <TableHead>{t('questionBank.column.createdBy')}</TableHead>
                    <TableHead
                      className="cursor-pointer select-none"
                      onClick={() => handleSort('updated_at')}
                    >
                      {t('questionBank.column.updatedAt')}
                      {sortIcon('updated_at')}
                    </TableHead>
                    <TableHead className="w-10" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.items.map((item) => (
                    <TableRow
                      key={item.id}
                      className={selectedIds.includes(item.id) ? 'bg-muted/50' : ''}
                    >
                      <TableCell>
                        <input
                          type="checkbox"
                          checked={selectedIds.includes(item.id)}
                          onChange={() => toggleRow(item.id)}
                          className="rounded"
                          aria-label={`Select question ${item.id}`}
                        />
                      </TableCell>
                      <TableCell className="max-w-xs">
                        <button
                          className="text-left text-sm font-medium hover:underline truncate block w-full"
                          onClick={() => navigate(`/admin/questions/${item.id}/edit`)}
                          title={item.stem_preview}
                        >
                          {item.stem_preview.length > 120
                            ? `${item.stem_preview.slice(0, 120)}…`
                            : item.stem_preview}
                        </button>
                        {item.tags.length > 0 && (
                          <div className="flex gap-1 mt-1 flex-wrap">
                            {item.tags.slice(0, 3).map((tag) => (
                              <span
                                key={tag}
                                className="text-xs bg-muted px-1.5 py-0.5 rounded-sm text-muted-foreground"
                              >
                                {tag}
                              </span>
                            ))}
                            {item.tags.length > 3 && (
                              <span className="text-xs text-muted-foreground">
                                +{item.tags.length - 3}
                              </span>
                            )}
                          </div>
                        )}
                      </TableCell>
                      <TableCell>
                        <TypeBadge type={item.type} />
                      </TableCell>
                      <TableCell>
                        <DifficultyBadge difficulty={item.difficulty} />
                      </TableCell>
                      <TableCell className="text-sm whitespace-nowrap">
                        {item.category_name}
                      </TableCell>
                      <TableCell>
                        <StatusTransitionCell
                          question={item}
                          onTransition={handleStatusTransition}
                        />
                      </TableCell>
                      <TableCell>
                        <LocaleCoverageIcons coverage={item.locale_coverage} />
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                        {item.created_by_name}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                        {formatRelativeTime(item.updated_at)}
                      </TableCell>
                      <TableCell>
                        <RowActionsMenu
                          question={item}
                          onEdit={() => navigate(`/admin/questions/${item.id}/edit`)}
                          onArchive={() =>
                            handleStatusTransition(item.id, 'archived')
                          }
                          onViewVersions={() => setVersionsQuestionId(item.id)}
                          onDelete={() => setDeleteTargetId(item.id)}
                        />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}

          {/* Pagination */}
          {data.items.length > 0 && (
            <Pagination
              page={page}
              total={data.meta.total}
              perPage={perPage}
              onPageChange={(p) => {
                setSearchParams((prev) => {
                  const next = new URLSearchParams(prev)
                  next.set('page', String(p))
                  return next
                }, { replace: true })
              }}
              onPerPageChange={(pp) =>
                updateParams({ per_page: String(pp), page: '1' })
              }
            />
          )}
        </>
      )}

      {/* Bulk action bar */}
      {selectedIds.length > 0 && (
        <BulkActionBar
          selectedIds={selectedIds}
          allItems={items}
          filters={filters}
          onArchive={handleBulkArchive}
          onClearSelection={() => setSelectedIds([])}
          archiveProgress={archiveProgress}
        />
      )}

      {/* Version history slideover */}
      <VersionHistorySlideover
        questionId={versionsQuestionId}
        onClose={() => setVersionsQuestionId(null)}
      />

      {/* Import modal */}
      <ImportModal
        open={importOpen}
        onClose={() => setImportOpen(false)}
        onSuccess={() => showNotification('success', 'Import completed')}
      />

      {/* Delete confirm dialog */}
      <DeleteConfirmDialog
        open={!!deleteTargetId}
        onClose={() => setDeleteTargetId(null)}
        onConfirm={handleDelete}
        isPending={deleteMutation.isPending}
      />

      {/* AI Generate dialog (examiner+ only) */}
      {canUseAI && (
        <AIGenerateDialog
          open={aiGenerateOpen}
          onClose={() => setAIGenerateOpen(false)}
          onSuccess={(count) =>
            showNotification('success', t('questionBank.import.progress', { n: count, total: count }))
          }
        />
      )}
    </div>
  )
}
