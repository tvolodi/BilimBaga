import { useLocation, Navigate, Outlet } from 'react-router-dom'
import { useState } from 'react'
import { useLocalStorage } from 'usehooks-ts'
import { useMe } from '@/api/users'
import { Sidebar } from '@/components/admin/Sidebar'
import { TopBar } from '@/components/admin/TopBar'
import { Breadcrumb } from '@/components/admin/Breadcrumb'
import { useSmUp } from '@/hooks/useSmUp'

export function AdminLayout() {
  const location = useLocation()
  const { data: user, isLoading } = useMe()
  // The desktop preference is stored. Below sm the sidebar starts as a rail and opens over the page,
  // so the top bar keeps its width on a phone (FR-BB321 AC-6, #472). That state is not stored.
  const [collapsed, setCollapsed] = useLocalStorage('sidebar-collapsed', false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const smUp = useSmUp()

  if (isLoading) return null

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  if (user.role_name === 'employee') {
    return <Navigate to="/portal" replace />
  }

  const sidebarCollapsed = smUp ? collapsed : !mobileOpen

  function toggleSidebar() {
    if (smUp) setCollapsed(!collapsed)
    else setMobileOpen(!mobileOpen)
  }

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar
        collapsed={sidebarCollapsed}
        overlay={!smUp && !sidebarCollapsed}
        onToggle={toggleSidebar}
        onClose={() => setMobileOpen(false)}
      />
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
