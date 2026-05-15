import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, Loader2, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Select } from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { DateTimePicker } from '@/components/ui/date-time-picker'
import { useUsers } from '@/api/users'
import { useDepartments } from '@/api/departments'
import {
  useExamAssignments,
  useAddAssignment,
  useDeleteAssignment,
  type ExamAssignment,
  type ExamApiError,
} from '@/api/exams'

// ---- Types ------------------------------------------------------------------

type AssigneeType = 'user' | 'department' | 'all'

interface PendingAssignment {
  assignee_type: AssigneeType
  assignee_id: string
  deadline: string | null
}

// ---- Component --------------------------------------------------------------

interface Step3AssignmentsProps {
  examId: string
  canMutate: boolean
  onBack: () => void
  onNext: () => void
}

export function Step3Assignments({ examId, canMutate, onBack, onNext }: Step3AssignmentsProps) {
  const { t } = useTranslation()

  const { data: assignments = [], isLoading } = useExamAssignments(examId)
  const { data: usersData } = useUsers({ per_page: 200 })
  const { data: departments = [] } = useDepartments()
  const addAssignment = useAddAssignment(examId)
  const deleteAssignment = useDeleteAssignment(examId)

  const users = usersData?.items ?? []

  const [pending, setPending] = useState<PendingAssignment>({
    assignee_type: 'user',
    assignee_id: '',
    deadline: null,
  })
  const [addError, setAddError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)

  function getAssigneeName(a: ExamAssignment): string {
    if (a.assignee_type === 'all') return t('exam.assignment.type.all')
    if (a.assignee_type === 'user') {
      const user = users.find((u) => u.id === a.assignee_id)
      return user ? user.full_name : (a.assignee_id ?? '—')
    }
    if (a.assignee_type === 'department') {
      const dept = departments.find((d) => d.id === a.assignee_id)
      return dept ? dept.name : (a.assignee_id ?? '—')
    }
    return '—'
  }

  async function handleAdd() {
    setAddError(null)
    if (pending.assignee_type !== 'all' && !pending.assignee_id) {
      setAddError(t('exam.error.assigneeRequired'))
      return
    }
    try {
      await addAssignment.mutateAsync({
        assignee_type: pending.assignee_type,
        assignee_id: pending.assignee_type !== 'all' ? pending.assignee_id : null,
        deadline: pending.deadline,
      })
      setPending({ assignee_type: 'user', assignee_id: '', deadline: null })
      setShowAdd(false)
    } catch (err) {
      setAddError((err as ExamApiError).message)
    }
  }

  async function handleDelete(assignmentId: string) {
    try {
      await deleteAssignment.mutateAsync(assignmentId)
    } catch (err) {
      setAddError((err as ExamApiError).message)
    }
  }

  return (
    <div className="space-y-4">
      <h2 className="text-base font-medium">{t('exam.wizard.step3.title')}</h2>

      {addError && (
        <div className="flex items-center gap-2 rounded-md bg-destructive/10 border border-destructive/20 px-3 py-2 text-sm text-destructive">
          <AlertCircle size={14} className="shrink-0" />
          {addError}
        </div>
      )}

      {isLoading && (
        <div className="flex justify-center py-4">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      )}

      {/* Existing assignments */}
      {assignments.length === 0 && !isLoading && (
        <p className="text-sm text-muted-foreground">{t('exam.assignment.empty')}</p>
      )}

      <div className="space-y-2">
        {assignments.map((a) => (
          <div
            key={a.id}
            className="flex items-center justify-between rounded-lg border px-4 py-3"
          >
            <div>
              <span className="text-sm font-medium">{getAssigneeName(a)}</span>
              <span className="ml-2 text-xs text-muted-foreground capitalize">
                ({t(`exam.assignment.type.${a.assignee_type}`)})
              </span>
              {a.deadline && (
                <p className="text-xs text-muted-foreground mt-0.5">
                  {t('exam.assignment.deadline')}: {new Date(a.deadline).toLocaleDateString()}
                </p>
              )}
            </div>
            {canMutate && (
              <button
                type="button"
                onClick={() => handleDelete(a.id)}
                disabled={deleteAssignment.isPending}
                className="text-muted-foreground hover:text-destructive transition-colors"
                aria-label="remove assignment"
              >
                <Trash2 size={16} />
              </button>
            )}
          </div>
        ))}
      </div>

      {/* Add assignee form (admin+ only) */}
      {canMutate && (
        <div>
          {!showAdd ? (
            <Button type="button" variant="outline" onClick={() => setShowAdd(true)}>
              <Plus size={16} className="mr-2" />
              {t('exam.wizard.step3.addAssignee')}
            </Button>
          ) : (
            <div className="rounded-lg border p-4 space-y-3">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {/* Assignee type */}
                <div className="space-y-1">
                  <Label>{t('exam.assignment.type')}</Label>
                  <Select
                    value={pending.assignee_type}
                    onChange={(e) =>
                      setPending((p) => ({
                        ...p,
                        assignee_type: e.target.value as AssigneeType,
                        assignee_id: '',
                      }))
                    }
                  >
                    <option value="user">{t('exam.assignment.type.user')}</option>
                    <option value="department">{t('exam.assignment.type.department')}</option>
                    <option value="all">{t('exam.assignment.type.all')}</option>
                  </Select>
                </div>

                {/* Assignee selector (conditional) */}
                {pending.assignee_type === 'user' && (
                  <div className="space-y-1">
                    <Label>{t('exam.assignment.type.user')}</Label>
                    <Select
                      value={pending.assignee_id}
                      onChange={(e) => setPending((p) => ({ ...p, assignee_id: e.target.value }))}
                    >
                      <option value="">—</option>
                      {users.map((u) => (
                        <option key={u.id} value={u.id}>
                          {u.full_name} ({u.email})
                        </option>
                      ))}
                    </Select>
                  </div>
                )}

                {pending.assignee_type === 'department' && (
                  <div className="space-y-1">
                    <Label>{t('exam.assignment.type.department')}</Label>
                    <Select
                      value={pending.assignee_id}
                      onChange={(e) => setPending((p) => ({ ...p, assignee_id: e.target.value }))}
                    >
                      <option value="">—</option>
                      {departments.map((d) => (
                        <option key={d.id} value={d.id}>{d.name}</option>
                      ))}
                    </Select>
                  </div>
                )}
              </div>

              {/* Deadline */}
              <DateTimePicker
                label={t('exam.assignment.deadline')}
                value={pending.deadline}
                onChange={(v) => setPending((p) => ({ ...p, deadline: v }))}
              />

              <div className="flex gap-2">
                <Button
                  type="button"
                  onClick={handleAdd}
                  disabled={addAssignment.isPending}
                  size="sm"
                >
                  {addAssignment.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {t('common.save')}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setShowAdd(false)
                    setAddError(null)
                  }}
                >
                  {t('common.cancel')}
                </Button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Navigation */}
      <div className="flex justify-between pt-4">
        <Button type="button" variant="outline" onClick={onBack}>
          {t('exam.wizard.back')}
        </Button>
        <Button type="button" onClick={onNext}>
          {t('exam.wizard.next')}
        </Button>
      </div>
    </div>
  )
}
