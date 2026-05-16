import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { TenantProvider } from '@/components/TenantProvider'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { RequireAuth } from '@/components/RequireAuth'
import { RequireRole } from '@/components/RequireRole'
import { LoginPage } from '@/pages/auth/LoginPage'
import { ChangePasswordPage } from '@/pages/auth/ChangePasswordPage'
import { AdminLayout } from '@/layouts/AdminLayout'
import { PortalLayout } from '@/layouts/PortalLayout'
import { EmployeePortal } from '@/pages/EmployeePortal'
import { MyResultsPage } from '@/pages/portal/MyResultsPage'
import { UsersListPage } from '@/pages/admin/users/UsersListPage'
import { DepartmentsPage } from '@/pages/admin/departments/DepartmentsPage'
import { BrandingSettingsPage } from '@/pages/admin/settings/BrandingSettingsPage'
import { QuestionBankPage } from '@/pages/admin/questions/QuestionBankPage'
import { QuestionEditorPage } from '@/pages/admin/questions/QuestionEditorPage'
import { CategoriesPage } from '@/pages/admin/categories/CategoriesPage'
import { TagsPage } from '@/pages/admin/tags/TagsPage'
import { RequireSuperAdmin } from '@/components/RequireSuperAdmin'
import { useRefreshToken } from '@/api/auth'
import { useLocaleDirection } from '@/hooks/useLocaleDirection'
import { SkipLink } from '@/components/SkipLink'
import { ExamWizardCreatePage, ExamWizardEditPage } from '@/pages/ExamWizard'
import { ExamTakingPage } from '@/pages/ExamTaking'
import { ResultPage } from '@/pages/ResultPage'
import { GradingQueuePage } from '@/pages/admin/GradingQueuePage'
import { GradingDetailPage } from '@/pages/admin/GradingDetailPage'
import { EmployeeRecordPage } from '@/pages/admin/EmployeeRecordPage'
import { AuditLogPage } from '@/pages/admin/AuditLogPage'
import { AdminDashboardPage } from '@/pages/admin/AdminDashboardPage'
import { ExamsListPage } from '@/pages/admin/ExamsListPage'
import { ExamAnalyticsPage } from '@/pages/admin/ExamAnalyticsPage'

const queryClient = new QueryClient()

function AppRoutes() {
  const { isLoading } = useRefreshToken()
  if (isLoading) return <FullPageSpinner />

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
            <AdminLayout />
          </RequireRole>
        }
      >
        <Route index element={<Navigate to="dashboard" replace />} />
        <Route
          path="dashboard"
          element={
            <RequireRole roles={['examiner', 'hr_admin', 'super_admin']}>
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
            <RequireRole roles={['super_admin', 'hr_admin', 'examiner']}>
              <AuditLogPage />
            </RequireRole>
          }
        />
      </Route>
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
      <AppRoutes />
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

