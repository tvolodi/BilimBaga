import { useLocation, Navigate, Outlet } from 'react-router-dom'
import { useLocalStorage } from 'usehooks-ts'
import { useMe } from '@/api/users'
import { Sidebar } from '@/components/admin/Sidebar'
import { TopBar } from '@/components/admin/TopBar'
import { Breadcrumb } from '@/components/admin/Breadcrumb'

export function AdminLayout() {
  const location = useLocation()
  const { data: user, isLoading } = useMe()
  const [collapsed, setCollapsed] = useLocalStorage('sidebar-collapsed', false)

  if (isLoading) return null

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  if (user.role_name === 'employee') {
    return <Navigate to="/portal" replace />
  }

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar collapsed={collapsed} onToggle={() => setCollapsed(!collapsed)} />
      <div className="flex flex-1 flex-col overflow-hidden">
        <TopBar user={user} />
        <main id="main-content" tabIndex={-1} className="flex-1 overflow-y-auto p-6 outline-none">
          <Breadcrumb />
          <Outlet />
        </main>
      </div>
    </div>
  )
}
