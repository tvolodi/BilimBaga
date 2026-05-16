import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, Download } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useQueryClient } from '@tanstack/react-query'

interface ExportCSVButtonProps {
  examId: string
}

export function ExportCSVButton({ examId }: ExportCSVButtonProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const qc = useQueryClient()

  async function handleExport() {
    setLoading(true)
    try {
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      const res = await fetch(`/api/v1/admin/exams/${examId}/results/export`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        credentials: 'include',
      })
      if (!res.ok) throw new Error(`Export failed: ${res.status}`)
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      const disposition = res.headers.get('Content-Disposition') ?? ''
      const match = /filename="([^"]+)"/.exec(disposition)
      a.download = match ? match[1] : `exam-${examId}-results.csv`
      a.href = url
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Button variant="outline" onClick={handleExport} disabled={loading}>
      {loading ? (
        <>
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          {t('common.loading')}
        </>
      ) : (
        <>
          <Download className="mr-2 h-4 w-4" />
          {t('exam_analytics.export_csv')}
        </>
      )}
    </Button>
  )
}
