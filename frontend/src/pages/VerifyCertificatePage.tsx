import { useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { CheckCircle2, XCircle, AlertTriangle } from 'lucide-react'
import { useVerifyCertificate } from '@/api/verify'
import { useTenantConfig } from '@/api/useTenantConfig'
import { TenantLogo } from '@/components/TenantLogo'
import { LocaleSwitcher } from '@/components/LocaleSwitcher'
import { ThemeToggle } from '@/components/ThemeToggle'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

export function VerifyCertificatePage() {
  const { code = '' } = useParams<{ code: string }>()
  const { t, i18n } = useTranslation()
  const { data: tenant } = useTenantConfig()
  const { data, isPending, isError, refetch, isFetching } = useVerifyCertificate(code)

  // AC-7: keep verification pages out of search indexes while mounted.
  useEffect(() => {
    const meta = document.createElement('meta')
    meta.name = 'robots'
    meta.content = 'noindex'
    document.head.appendChild(meta)
    return () => {
      meta.remove()
    }
  }, [])

  const issuedOn =
    data?.valid && data.issued_at
      ? new Intl.DateTimeFormat(i18n.language, { dateStyle: 'long', timeZone: 'UTC' }).format(
          new Date(data.issued_at),
        )
      : ''

  return (
    <main
      id="main-content"
      tabIndex={-1}
      className="mx-auto flex min-h-screen w-full max-w-md flex-col gap-6 px-4 py-8 outline-none"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <TenantLogo appName={tenant?.app_name} />
        <div className="flex items-center gap-2">
          <ThemeToggle />
          <LocaleSwitcher />
        </div>
      </div>
      {tenant?.app_name && <p className="text-center text-sm font-semibold">{tenant.app_name}</p>}

      <Card>
        <CardHeader>
          <CardTitle>
            <h1 className="text-xl">{t('verify.title')}</h1>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {isPending && (
            <div data-testid="verify-skeleton" aria-busy="true" className="space-y-3">
              <span className="sr-only">{t('verify.loading')}</span>
              <Skeleton className="h-6 w-2/3" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-1/2" />
            </div>
          )}

          {isError && (
            <div className="space-y-3">
              <p className="flex items-center gap-2 font-medium text-amber-600">
                <AlertTriangle aria-hidden="true" className="h-5 w-5" />
                {t('verify.unavailable')}
              </p>
              <Button onClick={() => void refetch()} disabled={isFetching}>
                {t('verify.retry')}
              </Button>
            </div>
          )}

          {data && data.valid && (
            <div className="space-y-4">
              <p
                role="status"
                className="flex items-center gap-2 text-lg font-semibold text-green-600"
              >
                <CheckCircle2 aria-hidden="true" className="h-6 w-6" />
                {t('verify.valid')}
              </p>
              <dl className="space-y-2 text-sm">
                <div>
                  <dt className="text-muted-foreground">{t('verify.employee')}</dt>
                  <dd className="font-medium">{data.employee_name}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">{t('verify.exam')}</dt>
                  <dd className="font-medium">{data.exam_title}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">{t('verify.score')}</dt>
                  <dd className="font-medium">{(data.score_pct ?? 0).toFixed(1)}%</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">{t('verify.issuedOn')}</dt>
                  <dd className="font-medium">{issuedOn}</dd>
                </div>
              </dl>
            </div>
          )}

          {data && !data.valid && (
            <p role="alert" className="flex items-center gap-2 text-lg font-semibold text-red-600">
              <XCircle aria-hidden="true" className="h-6 w-6" />
              {t('verify.invalid')}
            </p>
          )}
        </CardContent>
      </Card>

      <p className="text-center text-xs text-muted-foreground">{t('verify.footer')}</p>
    </main>
  )
}
