import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { useAssignableRoles } from '@/hooks/useAssignableRoles'
import { userErrorKey } from '@/lib/assignableRoles'
import { useImportUsers, type ImportPreview } from '@/api/users'

interface ImportModalProps {
  open: boolean
  onClose: () => void
}

export function ImportModal({ open, onClose }: ImportModalProps) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)
  const importUsers = useImportUsers()
  const { roles: assignable } = useAssignableRoles()

  const [preview, setPreview] = useState<ImportPreview | null>(null)
  const [committed, setCommitted] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handlePreview() {
    const file = fileRef.current?.files?.[0]
    if (!file) {
      setError('Please select a CSV file.')
      return
    }
    setError(null)
    try {
      const result = await importUsers.mutateAsync({ file, commit: false })
      setPreview(result)
      setCommitted(false)
    } catch (err) {
      const key = userErrorKey(err)
      setError(key ? t(key) : err instanceof Error ? err.message : t('users.messages.error_generic'))
    }
  }

  async function handleCommit() {
    const file = fileRef.current?.files?.[0]
    if (!file) return
    setError(null)
    try {
      const result = await importUsers.mutateAsync({ file, commit: true })
      setPreview(result)
      setCommitted(true)
    } catch (err) {
      const key = userErrorKey(err)
      setError(key ? t(key) : err instanceof Error ? err.message : t('users.messages.error_generic'))
    }
  }

  function handleClose() {
    setPreview(null)
    setCommitted(false)
    setError(null)
    if (fileRef.current) fileRef.current.value = ''
    onClose()
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) handleClose() }}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t('users.import_modal.title')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-1">
            <label className="text-sm font-medium">{t('users.import_modal.upload_label')}</label>
            <input
              ref={fileRef}
              type="file"
              accept=".csv"
              className="block w-full text-sm text-muted-foreground file:mr-4 file:py-2 file:px-4 file:rounded-md file:border-0 file:text-sm file:font-medium file:bg-primary file:text-primary-foreground hover:file:cursor-pointer"
            />
            <p className="text-xs text-muted-foreground">{t('users.import_modal.upload_hint')}</p>
            <p className="text-xs text-muted-foreground">
              {assignable.length > 0
                ? t('users.import_modal.assignable_roles_hint', { roles: assignable.map((r) => r.name).join(', ') })
                : t('users.import_modal.no_assignable_roles')}
            </p>
          </div>

          {error && <p className="text-sm text-red-600">{error}</p>}

          {preview && (
            <div className="space-y-3">
              <div className="flex gap-4 text-sm">
                <Badge variant="success">
                  {t('users.import_modal.valid_count', { count: preview.valid.length })}
                </Badge>
                {preview.errors.length > 0 && (
                  <Badge variant="destructive">
                    {t('users.import_modal.error_count', { count: preview.errors.length })}
                  </Badge>
                )}
              </div>

              {preview.errors.length > 0 && (
                <div className="max-h-48 overflow-y-auto border rounded">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>#</TableHead>
                        <TableHead>{t('users.columns.email')}</TableHead>
                        <TableHead>{t('users.columns.name')}</TableHead>
                        <TableHead>{t('users.columns.error')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {preview.errors.map((row) => (
                        <TableRow key={row.row_num} className="bg-red-50">
                          <TableCell>{row.row_num}</TableCell>
                          <TableCell>{row.email}</TableCell>
                          <TableCell>{row.full_name}</TableCell>
                          <TableCell className="text-red-600 text-xs">{row.error}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            {t('users.import_modal.close')}
          </Button>
          {!committed && (
            <Button variant="outline" onClick={handlePreview} disabled={importUsers.isPending}>
              {t('users.import_modal.preview_button')}
            </Button>
          )}
          {preview && !committed && preview.valid.length > 0 && (
            <Button onClick={handleCommit} disabled={importUsers.isPending}>
              {t('users.import_modal.commit_button')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
