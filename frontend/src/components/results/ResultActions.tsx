import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { downloadFile, downloadErrorKey } from '@/api/download'
import { Button } from '@/components/ui/button'

interface ResultActionsProps {
  sessionId: string
  passed: boolean
  certificateEnabled: boolean
  canRetake: boolean
  examId: string
}

export function ResultActions({
  sessionId,
  passed,
  certificateEnabled,
  canRetake,
}: ResultActionsProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const qc = useQueryClient()
  const [downloadError, setDownloadError] = useState<string | null>(null)

  async function handleDownloadCertificate() {
    setDownloadError(null)
    try {
      await downloadFile(
        qc,
        `/api/v1/portal/sessions/${sessionId}/certificate`,
        `certificate-${sessionId}.pdf`,
      )
    } catch (err) {
      setDownloadError(downloadErrorKey(err))
    }
  }

  return (
    <div className="flex flex-col gap-2 w-full sm:flex-row sm:justify-center">
      {downloadError && (
        <p role="alert" className="text-sm text-red-700">
          {t(downloadError)}
        </p>
      )}
      {passed && certificateEnabled && (
        <Button variant="default" onClick={handleDownloadCertificate}>
          {t('result.download_certificate')}
        </Button>
      )}
      {canRetake && (
        <Button variant="outline" onClick={() => navigate('/portal')}>
          {t('result.retake_exam')}
        </Button>
      )}
      <Button variant="ghost" onClick={() => navigate('/portal')}>
        {t('exam.taking.result.backToPortal')}
      </Button>
    </div>
  )
}
