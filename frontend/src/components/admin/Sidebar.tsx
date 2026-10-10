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
  ShieldCheck,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useMyPermissions } from '@/hooks/useMyPermissions'
import {
  AUDIT_READ_ROLES,
  REPORTS_READ_ROLES,
  ROLES_MANAGE_ROLES,
  ROUTE_PERMISSIONS as PERM,
  can,
} from '@/lib/routeRoles'

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
  /** Below sm the expanded sidebar floats over the page instead of taking width from it (#472). */
  overlay?: boolean
  /** Called when a link is followed, so an overlay closes. */
  onNavigate?: () => void
}

interface NavItem {
  key: string
  icon: typeof Users
  path: string
  labelKey: string
  end: boolean
  /** When set, the link is shown only to these roles (mirrors the route guards). */
  roles?: string[]
  /** Permission a custom (non-built-in) role needs to see the link (FR-BB117 AC-16). */
  permission: string
}

const NAV_ITEMS: NavItem[] = [
  { key: 'dashboard', icon: LayoutDashboard, path: '/admin', labelKey: 'nav.dashboard', end: true, permission: PERM.dashboard },
  { key: 'users', icon: Users, path: '/admin/users', labelKey: 'nav.users', end: false, permission: PERM.users },
  { key: 'roles', icon: ShieldCheck, path: '/admin/roles', labelKey: 'nav.roles', end: false, roles: ROLES_MANAGE_ROLES, permission: PERM.roles },
  { key: 'departments', icon: Building2, path: '/admin/departments', labelKey: 'nav.departments', end: false, permission: PERM.departments },
  { key: 'questions', icon: FileQuestion, path: '/admin/questions', labelKey: 'nav.questions', end: false, permission: PERM.questions },
  { key: 'categories', icon: FolderTree, path: '/admin/categories', labelKey: 'nav.categories', end: false, permission: PERM.categories },
  { key: 'tags', icon: Hash, path: '/admin/tags', labelKey: 'nav.tags', end: false, permission: PERM.tags },
  { key: 'exams', icon: ClipboardList, path: '/admin/exams', labelKey: 'nav.exams', end: false, permission: PERM.exams },
  { key: 'grading', icon: ClipboardCheck, path: '/admin/grading', labelKey: 'grading.nav', end: false, permission: PERM.grading },
  { key: 'reports', icon: BarChart2, path: '/admin/reports', labelKey: 'nav.reports', end: false, roles: REPORTS_READ_ROLES, permission: PERM.reports },
  { key: 'audit', icon: ScrollText, path: '/admin/audit', labelKey: 'nav.audit', end: false, roles: AUDIT_READ_ROLES, permission: PERM.audit },
  { key: 'settings', icon: Settings, path: '/admin/settings/branding', labelKey: 'nav.settings', end: false, permission: PERM.settings },
]

export function Sidebar({ collapsed, onToggle, overlay = false, onNavigate }: SidebarProps) {
  const { t } = useTranslation()
  const { data: tenantConfig } = useTenantConfig()
  const { role, isCustom, permissions } = useMyPermissions()
  // Built-in roles: unchanged role-list logic. Custom roles: only links whose permission they hold
  // (hidden while permissions are unknown). The API stays authoritative.
  const visibleItems = NAV_ITEMS.filter((item) =>
    isCustom
      ? can(permissions, item.permission)
      : !item.roles || (role !== undefined && item.roles.includes(role)),
  )

  return (
    <aside
      className={cn(
        'flex flex-col h-screen bg-bg-navy text-text-on-navy transition-all duration-200',
        overlay && 'fixed inset-y-0 start-0 z-40 shadow-xl',
        collapsed ? 'w-16' : 'w-56',
      )}
    >
      {/* Logo / Header */}
      <div className="flex items-center justify-between px-4 py-4 border-b border-text-on-navy/15">
        {!collapsed && (
          <span className="text-lg font-bold tracking-tight truncate">{tenantConfig?.app_name ?? 'BilimBaga'}</span>
        )}
        <button
          onClick={onToggle}
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          aria-expanded={!collapsed}
          className="inline-flex items-center justify-center p-1 max-sm:min-h-11 max-sm:min-w-11 rounded hover:bg-text-on-navy/15 text-text-on-navy/80 hover:text-text-on-navy transition-colors ml-auto"
        >
          {collapsed ? <ChevronRight size={18} /> : <ChevronLeft size={18} />}
        </button>
      </div>

      {/* Nav items */}
      <nav aria-label="Main navigation" className="flex-1 overflow-y-auto py-2">
        {visibleItems.map(({ key, icon: Icon, path, labelKey, end }) => (
          <NavLink
            key={key}
            to={path}
            end={end}
            aria-label={collapsed ? t(labelKey) : undefined}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 px-4 py-2.5 text-sm font-medium transition-colors max-sm:min-h-11',
                isActive
                  ? 'bg-text-on-navy/15 text-text-on-navy'
                  : 'text-text-on-navy/80 hover:bg-text-on-navy/10 hover:text-text-on-navy',
                collapsed && 'justify-center px-0',
              )
            }
            title={collapsed ? t(labelKey) : undefined}
            onClick={() => onNavigate?.()}
          >
            <Icon size={18} className="flex-shrink-0" aria-hidden="true" />
            {!collapsed && <span className="truncate">{t(labelKey)}</span>}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
