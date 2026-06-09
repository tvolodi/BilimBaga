import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
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

  async function handleDownloadCertificate() {
    const res = await fetch(`/api/v1/portal/sessions/${sessionId}/certificate`)
    if (!res.ok) return
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `certificate-${sessionId}.pdf`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="flex flex-col gap-2 w-full sm:flex-row sm:justify-center">
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
