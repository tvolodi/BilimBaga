import { useMatches, Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'

interface BreadcrumbHandle {
  breadcrumb?: string
}

export function Breadcrumb() {
  const matches = useMatches()

  const crumbs = matches
    .filter((m) => Boolean((m.handle as BreadcrumbHandle | null)?.breadcrumb))
    .map((m) => ({
      pathname: m.pathname,
      label: (m.handle as BreadcrumbHandle).breadcrumb as string,
    }))

  if (crumbs.length === 0) return null

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
