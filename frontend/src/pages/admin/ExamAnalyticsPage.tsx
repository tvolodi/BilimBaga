import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, ChevronDown, ChevronUp } from 'lucide-react'
import { useExamAnalytics } from '@/api/analytics'
import { useAIInsights } from '@/api/ai'
import { useTenantConfig } from '@/api/useTenantConfig'
import { RequireRole } from '@/components/RequireRole'
import { Skeleton } from '@/components/ui/skeleton'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ScoreDistributionChart } from '@/components/analytics/ScoreDistributionChart'
import { PassRateChart } from '@/components/analytics/PassRateChart'
import { StatsSummaryRow } from '@/components/analytics/StatsSummaryRow'
import { QuestionDifficultyTable } from '@/components/analytics/QuestionDifficultyTable'
import { ExportCSVButton } from '@/components/analytics/ExportCSVButton'
import type { QuestionStat } from '@/api/analytics'

type SortCol = 'correct_rate' | 'avg_time_seconds'
type SortDir = 'asc' | 'desc'

function sortQuestions(
  questions: QuestionStat[],
  col: SortCol,
  dir: SortDir,
): QuestionStat[] {
  return [...questions].sort((a, b) => {
    const av = a[col] ?? 0
    const bv = b[col] ?? 0
    return dir === 'asc' ? av - bv : bv - av
  })
}

function formatDate(iso: string, locale: string): string {
  try {
    return new Date(iso).toLocaleString(locale, {
      dateStyle: 'medium',
      timeStyle: 'short',
    })
  } catch {
    return iso
  }
}

export function AIInsightsCard({ examId }: { examId: string }) {
  const { t, i18n } = useTranslation()
  const [refresh, setRefresh] = useState(false)
  const [open, setOpen] = useState(false)

  const { data, isLoading, isError, refetch } = useAIInsights(examId, refresh)

  const handleRegenerate = () => {
    setRefresh(true)
    refetch().finally(() => setRefresh(false))
  }

  return (
    <Card data-testid="ai-insights-card">
      <CardHeader className="flex flex-row items-center justify-between py-4">
        <div className="flex items-center gap-2">
          <h2 className="text-base font-semibold">{t('exam_analytics.aiInsightsTitle')}</h2>
          <Badge variant="outline" className="text-xs">{t('common.aiGenerated')}</Badge>
        </div>
        <div className="flex items-center gap-2">
          {data && (
            <span className="text-xs text-muted-foreground">
              {t('exam_analytics.generatedAt', { date: formatDate(data.generated_at, i18n.language) })}
              {data.cached && ` · ${t('exam_analytics.cached')}`}
            </span>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={handleRegenerate}
            disabled={isLoading}
          >
            {t('exam_analytics.regenerate')}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setOpen((v) => !v)}
            aria-label={t('common.toggleExpand')}
            aria-expanded={open}
          >
            {open ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          </Button>
        </div>
      </CardHeader>

      {open && (
        <CardContent>
          {isLoading && <Skeleton className="h-20 w-full" data-testid="ai-insights-skeleton" />}
          {isError && (
            <p className="text-destructive text-sm" data-testid="ai-insights-error">
              {t('errors.aiUnavailable')}
            </p>
          )}
          {data && !isLoading && (
            <ul className="list-disc list-inside space-y-1" data-testid="ai-insights-list">
              {data.insights.map((insight, i) => (
                <li key={i} className="text-sm">{insight}</li>
              ))}
            </ul>
          )}
        </CardContent>
      )}
    </Card>
  )
}

function PageSkeleton() {
  return (
    <div className="space-y-6" data-testid="analytics-skeleton">
      {/* Stats row skeleton */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <Card key={i}>
            <CardContent className="p-6">
              <Skeleton className="h-4 w-32 mb-3" />
              <Skeleton className="h-10 w-20" />
            </CardContent>
          </Card>
        ))}
      </div>
      {/* Charts skeleton */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader><Skeleton className="h-5 w-40" /></CardHeader>
          <CardContent><Skeleton className="h-60 w-full" /></CardContent>
        </Card>
        <Card>
          <CardHeader><Skeleton className="h-5 w-32" /></CardHeader>
          <CardContent><Skeleton className="h-60 w-full" /></CardContent>
        </Card>
      </div>
      {/* Table skeleton */}
      <Card>
        <CardHeader><Skeleton className="h-5 w-48" /></CardHeader>
        <CardContent>
          <Skeleton className="h-64 w-full" />
        </CardContent>
      </Card>
    </div>
  )
}

function ExamAnalyticsContent({ examId }: { examId: string }) {
  const { t } = useTranslation()
  const { data, isLoading, isError } = useExamAnalytics(examId)
  const { data: tenantConfig } = useTenantConfig()
  const primaryColor = tenantConfig?.primary_color ?? '#6366f1'

  const [sortCol, setSortCol] = useState<SortCol>('correct_rate')
  const [sortDir, setSortDir] = useState<SortDir>('asc')

  function handleSort(col: SortCol) {
    if (col === sortCol) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc')
    } else {
      setSortCol(col)
      setSortDir('asc')
    }
  }

  if (isLoading) return <PageSkeleton />

  if (isError || !data) {
    return (
      <p className="text-sm text-destructive">{t('common.loadError')}</p>
    )
  }

  if (data.total_attempts === 0) {
    return (
      <div className="rounded-lg border bg-muted/30 p-12 text-center">
        <p className="text-muted-foreground">{t('exam_analytics.no_data')}</p>
      </div>
    )
  }

  const sortedQuestions = sortQuestions(data.per_question_stats, sortCol, sortDir)

  return (
    <div className="space-y-6">
      {/* Stats summary */}
      <StatsSummaryRow
        avgScore={data.avg_score}
        medianScore={data.median_score}
        totalAttempts={data.total_attempts}
        uniqueParticipants={data.unique_participants}
      />

      {/* Charts */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <h2 className="text-base font-semibold">
              {t('exam_analytics.score_distribution')}
            </h2>
          </CardHeader>
          <CardContent>
            <ScoreDistributionChart
              distribution={data.score_distribution}
              primaryColor={primaryColor}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <h2 className="text-base font-semibold">{t('exam_analytics.pass_rate')}</h2>
          </CardHeader>
          <CardContent className="flex justify-center">
            <PassRateChart passRate={data.pass_rate} primaryColor={primaryColor} />
          </CardContent>
        </Card>
      </div>

      {/* Question difficulty table */}
      <Card>
        <CardHeader>
          <h2 className="text-base font-semibold">{t('exam_analytics.question_difficulty')}</h2>
        </CardHeader>
        <CardContent>
          <QuestionDifficultyTable
            questions={sortedQuestions}
            sortCol={sortCol}
            sortDir={sortDir}
            onSort={handleSort}
          />
        </CardContent>
      </Card>

      {/* Export */}
      <div className="flex justify-end">
        <ExportCSVButton examId={examId} />
      </div>

      {/* AI Insights (FR-BB74) */}
      <AIInsightsCard examId={examId} />
    </div>
  )
}

export function ExamAnalyticsPage() {
  const { examId } = useParams<{ examId: string }>()
  const { t } = useTranslation()

  if (!examId) return null

  // department_admin is admitted: the analytics API scopes these queries to its department subtree (deptscope).
  return (
    <RequireRole roles={['examiner', 'hr_admin', 'super_admin', 'department_admin']}>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center gap-4">
          <Link
            to="/admin/exams"
            className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            <ArrowLeft size={16} />
            {t('nav.exams')}
          </Link>
          <h1 className="text-xl font-semibold">{t('exam_analytics.title')}</h1>
        </div>

        <ExamAnalyticsContent examId={examId} />
      </div>
    </RequireRole>
  )
}
