import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { ChevronRight, ChevronDown, GripVertical, MoreHorizontal } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { type CategoryNode } from '@/api/categories'
import { CategoryTree } from './CategoryTree'

interface CategoryTreeNodeProps {
  node: CategoryNode
  depth: number
  canManage: boolean
  onAddChild: (node: CategoryNode) => void
  onEdit: (node: CategoryNode) => void
  onDelete: (node: CategoryNode) => void
}

export function CategoryTreeNode({
  node,
  depth,
  canManage,
  onAddChild,
  onEdit,
  onDelete,
}: CategoryTreeNodeProps) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(true)
  const [menuOpen, setMenuOpen] = useState(false)

  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: node.id,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  }

  const hasChildren = node.children?.length > 0

  return (
    <div ref={setNodeRef} style={style}>
      <div
        className="flex items-center gap-1 py-1.5 pr-2 rounded hover:bg-muted/50 group"
        style={{ paddingLeft: `${depth * 20 + 8}px` }}
      >
        {/* Drag handle */}
        {canManage && (
          <button
            {...attributes}
            {...listeners}
            className="cursor-grab text-muted-foreground opacity-0 group-hover:opacity-100 flex-shrink-0"
            tabIndex={-1}
          >
            <GripVertical size={14} />
          </button>
        )}

        {/* Expand/collapse toggle */}
        <button
          onClick={() => setExpanded((v) => !v)}
          className="flex-shrink-0 text-muted-foreground w-4 h-4 flex items-center justify-center"
          aria-label={t('common.toggleExpand')}
          aria-expanded={hasChildren ? expanded : undefined}
        >
          {hasChildren ? (
            expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />
          ) : (
            <span className="w-4" />
          )}
        </button>

        {/* Name */}
        <span className="flex-1 text-sm font-medium truncate">{node.name}</span>

        {/* Track badge */}
        {node.track && (
          <Badge variant="secondary" className="text-xs px-1.5 py-0 h-5 flex-shrink-0">
            {node.track}
          </Badge>
        )}

        {/* Child count */}
        {hasChildren && (
          <span className="text-xs text-muted-foreground flex-shrink-0">
            {node.children.length}
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
                <div className="absolute right-0 top-7 z-20 w-36 rounded-md border bg-background shadow-md py-1">
                  <button
                    className="w-full text-left px-3 py-1.5 text-sm hover:bg-muted"
                    onClick={() => { setMenuOpen(false); onAddChild(node) }}
                  >
                    {t('categories.actions.addChild')}
                  </button>
                  <button
                    className="w-full text-left px-3 py-1.5 text-sm hover:bg-muted"
                    onClick={() => { setMenuOpen(false); onEdit(node) }}
                  >
                    {t('categories.actions.edit')}
                  </button>
                  <button
                    className="w-full text-left px-3 py-1.5 text-sm text-red-600 hover:bg-muted"
                    onClick={() => { setMenuOpen(false); onDelete(node) }}
                  >
                    {t('categories.actions.delete')}
                  </button>
                </div>
              </>
            )}
          </div>
        )}
      </div>

      {/* Children */}
      {hasChildren && expanded && (
        <CategoryTree
          nodes={node.children}
          depth={depth + 1}
          canManage={canManage}
          onAddChild={onAddChild}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      )}
    </div>
  )
}
