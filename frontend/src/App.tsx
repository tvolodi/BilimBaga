import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { TenantProvider } from '@/components/TenantProvider'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { RequireAuth } from '@/components/RequireAuth'
import { RequireRole } from '@/components/RequireRole'
import { LoginPage } from '@/pages/auth/LoginPage'
import { ChangePasswordPage } from '@/pages/auth/ChangePasswordPage'
import { AdminShell } from '@/pages/AdminShell'
import { EmployeePortal } from '@/pages/EmployeePortal'
import { UsersListPage } from '@/pages/users/UsersListPage'
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
        path="/admin/*"
        element={
          <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
            <AdminShell />
          </RequireRole>
        }
      />
      <Route
        path="/portal/*"
        element={
          <RequireRole roles={['employee']}>
            <EmployeePortal />
          </RequireRole>
        }
      />
      <Route
        path="/users"
        element={
          <RequireRole roles={['super_admin', 'department_admin', 'examiner']}>
            <UsersListPage />
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

