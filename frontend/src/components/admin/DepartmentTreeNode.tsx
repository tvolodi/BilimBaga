import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronRight, ChevronDown, MoreHorizontal } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { type Department } from '@/api/departments'
import { DepartmentTree } from './DepartmentTree'

interface Props {
  node: Department
  depth: number
  canManage: boolean
  onAddChild: (node: Department) => void
  onRename: (node: Department) => void
  onDelete: (node: Department) => void
}

export function DepartmentTreeNode({ node, depth, canManage, onAddChild, onRename, onDelete }: Props) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(true)
  const [menuOpen, setMenuOpen] = useState(false)

  const hasChildren = node.children?.length > 0

  return (
    <div>
      <div
        className="flex items-center gap-1 py-1.5 pr-2 rounded hover:bg-muted/50 group"
        style={{ paddingLeft: `${depth * 20 + 8}px` }}
      >
        {/* Expand/collapse toggle */}
        <button
          onClick={() => setExpanded((v) => !v)}
          className="flex-shrink-0 text-muted-foreground w-5 h-5 flex items-center justify-center"
          aria-label={t('common.toggleExpand')}
        >
          {hasChildren ? (
            expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />
          ) : (
            <span className="w-5" />
          )}
        </button>

        {/* Name */}
        <span className="flex-1 text-sm font-medium truncate">{node.name}</span>

        {/* Child count badge */}
        {hasChildren && (
          <span className="text-xs text-muted-foreground flex-shrink-0 mr-1">
            ({node.children.length} {t('departments.children')})
          </span>
        )}

        {/* Actions menu */}
        {canManage && (
          <div className="relative flex-shrink-0">
            <Button
              variant="ghost"
              size="sm"
              className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100"
              onClick={(e) => {
                e.stopPropagation()
                setMenuOpen((v) => !v)
              }}
              aria-label={t('common.actions')}
            >
              <MoreHorizontal size={14} />
            </Button>
            {menuOpen && (
              <>
                <div
                  aria-hidden="true"
                  className="fixed inset-0 z-10"
                  onClick={() => setMenuOpen(false)}
                />
                <div className="absolute right-0 top-7 z-20 w-40 rounded-md border bg-background shadow-md py-1">
                  <button
                    className="w-full text-left px-3 py-1.5 text-sm hover:bg-muted"
                    onClick={() => { setMenuOpen(false); onAddChild(node) }}
                  >
                    {t('departments.actions.addChild')}
                  </button>
                  <button
                    className="w-full text-left px-3 py-1.5 text-sm hover:bg-muted"
                    onClick={() => { setMenuOpen(false); onRename(node) }}
                  >
                    {t('departments.actions.rename')}
                  </button>
                  <button
                    className="w-full text-left px-3 py-1.5 text-sm text-danger hover:bg-muted"
                    onClick={() => { setMenuOpen(false); onDelete(node) }}
                  >
                    {t('departments.actions.delete')}
                  </button>
                </div>
              </>
            )}
          </div>
        )}
      </div>

      {/* Recursive children */}
      {hasChildren && expanded && (
        <DepartmentTree
          nodes={node.children}
          depth={depth + 1}
          canManage={canManage}
          onAddChild={onAddChild}
          onRename={onRename}
          onDelete={onDelete}
        />
      )}
    </div>
  )
}
