import { lazy, Suspense } from 'react'
import { useTranslation } from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { TenantProvider } from '@/components/TenantProvider'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { RequireAuth } from '@/components/RequireAuth'
import { RequireRole } from '@/components/RequireRole'
import { AUDIT_READ_ROLES, REPORTS_READ_ROLES } from '@/lib/routeRoles'
import { RequireSuperAdmin } from '@/components/RequireSuperAdmin'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import { useRefreshToken } from '@/api/auth'
import { useLocaleDirection } from '@/hooks/useLocaleDirection'
import { SkipLink } from '@/components/SkipLink'

// Non-admin routes — eagerly loaded (employee-facing, on the critical path)
import { LoginPage } from '@/pages/auth/LoginPage'
import { ChangePasswordPage } from '@/pages/auth/ChangePasswordPage'
import { ForgotPasswordPage } from '@/pages/auth/ForgotPasswordPage'
import { ResetPasswordPage } from '@/pages/auth/ResetPasswordPage'
import { EmployeePortal } from '@/pages/EmployeePortal'
import { MyResultsPage } from '@/pages/portal/MyResultsPage'
import { ExamResultRedirectPage } from '@/pages/portal/ExamResultRedirectPage'
import { ExamTakingPage } from '@/pages/ExamTaking'
import { ResultPage } from '@/pages/ResultPage'

// Layouts — eagerly loaded (shell chrome shared by all authenticated routes)
import { AdminLayout } from '@/layouts/AdminLayout'
import { PortalLayout } from '@/layouts/PortalLayout'

// AC-7: Admin-only routes lazy-loaded so the employee bundle stays lean
const UsersListPage = lazy(() =>
  import('@/pages/admin/users/UsersListPage').then((m) => ({ default: m.UsersListPage })),
)
const DepartmentsPage = lazy(() =>
  import('@/pages/admin/departments/DepartmentsPage').then((m) => ({ default: m.DepartmentsPage })),
)
const BrandingSettingsPage = lazy(() =>
  import('@/pages/admin/settings/BrandingSettingsPage').then((m) => ({
    default: m.BrandingSettingsPage,
  })),
)
const QuestionBankPage = lazy(() =>
  import('@/pages/admin/questions/QuestionBankPage').then((m) => ({ default: m.QuestionBankPage })),
)
const QuestionEditorPage = lazy(() =>
  import('@/pages/admin/questions/QuestionEditorPage').then((m) => ({
    default: m.QuestionEditorPage,
  })),
)
const CategoriesPage = lazy(() =>
  import('@/pages/admin/categories/CategoriesPage').then((m) => ({ default: m.CategoriesPage })),
)
const TagsPage = lazy(() =>
  import('@/pages/admin/tags/TagsPage').then((m) => ({ default: m.TagsPage })),
)
const ExamWizardCreatePage = lazy(() =>
  import('@/pages/ExamWizard').then((m) => ({ default: m.ExamWizardCreatePage })),
)
const ExamWizardEditPage = lazy(() =>
  import('@/pages/ExamWizard').then((m) => ({ default: m.ExamWizardEditPage })),
)
const GradingQueuePage = lazy(() =>
  import('@/pages/admin/GradingQueuePage').then((m) => ({ default: m.GradingQueuePage })),
)
const GradingDetailPage = lazy(() =>
  import('@/pages/admin/GradingDetailPage').then((m) => ({ default: m.GradingDetailPage })),
)
const EmployeeRecordPage = lazy(() =>
  import('@/pages/admin/EmployeeRecordPage').then((m) => ({ default: m.EmployeeRecordPage })),
)
const AuditLogPage = lazy(() =>
  import('@/pages/admin/AuditLogPage').then((m) => ({ default: m.AuditLogPage })),
)
const AdminDashboardPage = lazy(() =>
  import('@/pages/admin/AdminDashboardPage').then((m) => ({ default: m.AdminDashboardPage })),
)
const ExamsListPage = lazy(() =>
  import('@/pages/admin/ExamsListPage').then((m) => ({ default: m.ExamsListPage })),
)
const ExamAnalyticsPage = lazy(() =>
  import('@/pages/admin/ExamAnalyticsPage').then((m) => ({ default: m.ExamAnalyticsPage })),
)
const ReportsPage = lazy(() =>
  import('@/pages/admin/ReportsPage').then((m) => ({ default: m.ReportsPage })),
)

// FR-BB48: public certificate verification page (lazy, own small chunk)
const VerifyCertificatePage = lazy(() =>
  import('@/pages/VerifyCertificatePage').then((m) => ({ default: m.VerifyCertificatePage })),
)

const queryClient = new QueryClient()

// FR-BB48 AC-2/AC-7: /verify/* is fully public — it is rendered outside the auth
// bootstrap so no token refresh is attempted and no RequireAuth/RequireRole wraps it.
function AppRoutes() {
  const { pathname } = useLocation()
  if (pathname.startsWith('/verify/')) {
    return (
      <Suspense fallback={<FullPageSpinner />}>
        <Routes>
          <Route path="/verify/:code" element={<VerifyCertificatePage />} />
        </Routes>
      </Suspense>
    )
  }
  // FR-BB115 AC-7: account-recovery pages are public too — no refresh bootstrap, no auth wrappers.
  if (pathname === '/forgot-password' || pathname === '/reset-password') {
    return (
      <Routes>
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />
        <Route path="/reset-password" element={<ResetPasswordPage />} />
      </Routes>
    )
  }
  return <AuthedRoutes />
}

function AuthedRoutes() {
  const { t } = useTranslation()
  const { isLoading } = useRefreshToken()
  if (isLoading)
    return (
      <main id="main-content" tabIndex={-1} className="outline-none">
        <h1 className="sr-only">{t('common.loading')}</h1>
        <FullPageSpinner />
      </main>
    )

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/change-password"
        element={
          <RequireAuth>
            <ChangePasswordPage />
          </RequireAuth>
        }
      />
      <Route
        path="/admin"
        element={
          <RequireRole roles={['super_admin', 'department_admin', 'examiner', 'hr_admin']}>
            <Suspense fallback={<FullPageSpinner />}>
              <AdminLayout />
            </Suspense>
          </RequireRole>
        }
      >
        <Route index element={<Navigate to="dashboard" replace />} />
        <Route
          path="dashboard"
          element={
            <RequireRole roles={['examiner', 'hr_admin', 'super_admin', 'department_admin']}>
              <AdminDashboardPage />
            </RequireRole>
          }
        />
        <Route path="users" element={<UsersListPage />} />
        <Route
          path="users/:userId/record"
          element={
            <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
              <EmployeeRecordPage />
            </RequireRole>
          }
        />
        <Route path="departments" element={<DepartmentsPage />} />
        <Route
          path="questions"
          element={
            <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
              <QuestionBankPage />
            </RequireRole>
          }
        />
        <Route
          path="questions/new"
          element={
            <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
              <QuestionEditorPage />
            </RequireRole>
          }
        />
        <Route
          path="questions/:id/edit"
          element={
            <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
              <QuestionEditorPage />
            </RequireRole>
          }
        />
        <Route
          path="categories"
          element={
            <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
              <CategoriesPage />
            </RequireRole>
          }
        />
        <Route
          path="tags"
          element={
            <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
              <TagsPage />
            </RequireRole>
          }
        />
        <Route
          path="exams"
          element={
            <RequireRole roles={['super_admin', 'department_admin', 'examiner', 'hr_admin']}>
              <ExamsListPage />
            </RequireRole>
          }
        />
        <Route
          path="exams/new"
          element={
            <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
              <ExamWizardCreatePage />
            </RequireRole>
          }
        />
        <Route
          path="exams/:id/edit"
          element={
            <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
              <ExamWizardEditPage />
            </RequireRole>
          }
        />
        <Route
          path="exams/:examId/analytics"
          element={<ExamAnalyticsPage />}
        />
        <Route
          path="settings/branding"
          element={
            <RequireSuperAdmin>
              <BrandingSettingsPage />
            </RequireSuperAdmin>
          }
        />
        <Route
          path="grading"
          element={
            <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
              <GradingQueuePage />
            </RequireRole>
          }
        />
        <Route
          path="grading/:sessionId"
          element={
            <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
              <GradingDetailPage />
            </RequireRole>
          }
        />
        <Route
          path="audit"
          element={
            <RequireRole roles={AUDIT_READ_ROLES} unauthorizedRedirect="/login">
              <AuditLogPage />
            </RequireRole>
          }
        />
        <Route
          path="reports"
          element={
            <RequireRole roles={REPORTS_READ_ROLES} unauthorizedRedirect="/login">
              <ReportsPage />
            </RequireRole>
          }
        />
      </Route>
      <Route
        path="/portal/exams/:examId/result"
        element={
          <RequireRole roles={['employee']}>
            <ExamResultRedirectPage />
          </RequireRole>
        }
      />
      <Route
        path="/portal/sessions/:sessionId/result"
        element={
          <RequireRole roles={['employee']}>
            <ResultPage />
          </RequireRole>
        }
      />
      <Route
        path="/portal/sessions/:sessionId"
        element={
          <RequireRole roles={['employee']}>
            <ExamTakingPage />
          </RequireRole>
        }
      />
      <Route
        path="/portal"
        element={
          <RequireRole roles={['employee']}>
            <PortalLayout />
          </RequireRole>
        }
      >
        <Route index element={<EmployeePortal />} />
        <Route path="results" element={<MyResultsPage />} />
      </Route>
      <Route path="/" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}

function AppRoot() {
  useLocaleDirection()
  return (
    <>
      <SkipLink />
      <ErrorBoundary>
        <AppRoutes />
      </ErrorBoundary>
    </>
  )
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TenantProvider>
        <BrowserRouter>
          <AppRoot />
        </BrowserRouter>
      </TenantProvider>
    </QueryClientProvider>
  )
}

export default App

