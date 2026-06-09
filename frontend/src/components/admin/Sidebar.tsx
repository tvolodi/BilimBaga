import { useTranslation } from 'react-i18next'
import { NavLink } from 'react-router-dom'
import { useTenantConfig } from '@/api/useTenantConfig'
import {
  LayoutDashboard,
  Users,
  Building2,
  FileQuestion,
  FolderTree,
  Hash,
  ClipboardList,
  BarChart2,
  Settings,
  ChevronLeft,
  ChevronRight,
  ClipboardCheck,
  ScrollText,
} from 'lucide-react'
import { cn } from '@/lib/utils'

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
}

const NAV_ITEMS = [
  { key: 'dashboard', icon: LayoutDashboard, path: '/admin', labelKey: 'nav.dashboard', end: true },
  { key: 'users', icon: Users, path: '/admin/users', labelKey: 'nav.users', end: false },
  { key: 'departments', icon: Building2, path: '/admin/departments', labelKey: 'nav.departments', end: false },
  { key: 'questions', icon: FileQuestion, path: '/admin/questions', labelKey: 'nav.questions', end: false },
  { key: 'categories', icon: FolderTree, path: '/admin/categories', labelKey: 'nav.categories', end: false },
  { key: 'tags', icon: Hash, path: '/admin/tags', labelKey: 'nav.tags', end: false },
  { key: 'exams', icon: ClipboardList, path: '/admin/exams', labelKey: 'nav.exams', end: false },
  { key: 'grading', icon: ClipboardCheck, path: '/admin/grading', labelKey: 'grading.nav', end: false },
  { key: 'reports', icon: BarChart2, path: '/admin/reports', labelKey: 'nav.reports', end: false },
  { key: 'audit', icon: ScrollText, path: '/admin/audit', labelKey: 'nav.audit', end: false },
  { key: 'settings', icon: Settings, path: '/admin/settings/branding', labelKey: 'nav.settings', end: false },
]

export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const { t } = useTranslation()
  const { data: tenantConfig } = useTenantConfig()

  return (
    <aside
      className={cn(
        'flex flex-col h-screen bg-gray-900 text-gray-100 transition-all duration-200',
        collapsed ? 'w-16' : 'w-56',
      )}
    >
      {/* Logo / Header */}
      <div className="flex items-center justify-between px-4 py-4 border-b border-gray-700">
        {!collapsed && (
          <span className="text-lg font-bold tracking-tight truncate">{tenantConfig?.app_name ?? 'BilimBaga'}</span>
        )}
        <button
          onClick={onToggle}
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          className="p-1 rounded hover:bg-gray-700 text-gray-400 hover:text-white transition-colors ml-auto"
        >
          {collapsed ? <ChevronRight size={18} /> : <ChevronLeft size={18} />}
        </button>
      </div>

      {/* Nav items */}
      <nav aria-label="Main navigation" className="flex-1 overflow-y-auto py-2">
        {NAV_ITEMS.map(({ key, icon: Icon, path, labelKey, end }) => (
          <NavLink
            key={key}
            to={path}
            end={end}
            aria-label={collapsed ? t(labelKey) : undefined}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 px-4 py-2.5 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-gray-700 text-white'
                  : 'text-gray-400 hover:bg-gray-800 hover:text-white',
                collapsed && 'justify-center px-0',
              )
            }
            title={collapsed ? t(labelKey) : undefined}
          >
            <Icon size={18} className="flex-shrink-0" aria-hidden="true" />
            {!collapsed && <span className="truncate">{t(labelKey)}</span>}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
