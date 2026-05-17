import { useLocation, Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'

const ROUTE_LABELS: Record<string, string> = {
  dashboard: 'Dashboard',
  users: 'Users',
  departments: 'Departments',
  questions: 'Questions',
  categories: 'Categories',
  tags: 'Tags',
  exams: 'Exams',
  grading: 'Grading',
  audit: 'Audit Log',
  settings: 'Settings',
  branding: 'Branding',
  new: 'New',
  edit: 'Edit',
  record: 'Record',
  analytics: 'Analytics',
}

export function Breadcrumb() {
  const { pathname } = useLocation()
  const segments = pathname.replace(/^\/admin\/?/, '').split('/').filter(Boolean)

  if (segments.length === 0) return null

  const crumbs = segments.map((seg, i) => ({
    label: ROUTE_LABELS[seg] ?? seg,
    pathname: '/admin/' + segments.slice(0, i + 1).join('/'),
  }))

  return (
    <nav aria-label="breadcrumb" className="flex items-center gap-1 text-sm text-gray-500 mb-4">
      {crumbs.map((crumb, i) => (
        <span key={crumb.pathname} className="flex items-center gap-1">
          {i > 0 && <ChevronRight size={14} className="text-gray-400" />}
          {i < crumbs.length - 1 ? (
            <Link to={crumb.pathname} className="hover:text-gray-700 transition-colors">
              {crumb.label}
            </Link>
          ) : (
            <span className="text-gray-700 font-medium">{crumb.label}</span>
          )}
        </span>
      ))}
    </nav>
  )
}
