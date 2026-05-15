import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { TenantProvider } from '@/components/TenantProvider'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { RequireAuth } from '@/components/RequireAuth'
import { RequireRole } from '@/components/RequireRole'
import { LoginPage } from '@/pages/auth/LoginPage'
import { ChangePasswordPage } from '@/pages/auth/ChangePasswordPage'
import { AdminLayout } from '@/layouts/AdminLayout'
import { EmployeePortal } from '@/pages/EmployeePortal'
import { UsersListPage } from '@/pages/admin/users/UsersListPage'
import { DepartmentsPage } from '@/pages/admin/departments/DepartmentsPage'
import { BrandingSettingsPage } from '@/pages/admin/settings/BrandingSettingsPage'
import { QuestionBankPage } from '@/pages/admin/questions/QuestionBankPage'
import { QuestionEditorPage } from '@/pages/admin/questions/QuestionEditorPage'
import { CategoriesPage } from '@/pages/admin/categories/CategoriesPage'
import { TagsPage } from '@/pages/admin/tags/TagsPage'
import { RequireSuperAdmin } from '@/components/RequireSuperAdmin'
import { useRefreshToken } from '@/api/auth'

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
          <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
            <AdminLayout />
          </RequireRole>
        }
      >
        <Route index element={<Navigate to="users" replace />} />
        <Route path="users" element={<UsersListPage />} />
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
          path="settings/branding"
          element={
            <RequireSuperAdmin>
              <BrandingSettingsPage />
            </RequireSuperAdmin>
          }
        />
      </Route>
      <Route
        path="/portal/*"
        element={
          <RequireRole roles={['employee']}>
            <EmployeePortal />
          </RequireRole>
        }
      />
      <Route path="/" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TenantProvider>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
      </TenantProvider>
    </QueryClientProvider>
  )
}

export default App

