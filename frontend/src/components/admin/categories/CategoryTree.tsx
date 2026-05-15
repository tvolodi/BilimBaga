import { useMemo } from 'react'
import {
  DndContext,
  closestCenter,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { type CategoryNode } from '@/api/categories'
import { CategoryTreeNode } from './CategoryTreeNode'

interface CategoryTreeProps {
  nodes: CategoryNode[]
  depth?: number
  canManage: boolean
  onAddChild: (node: CategoryNode) => void
  onEdit: (node: CategoryNode) => void
  onDelete: (node: CategoryNode) => void
  onReorder?: (parentId: string | null, nodeId: string, newSortOrder: number) => void
}

function sortNodes(nodes: CategoryNode[]): CategoryNode[] {
  return [...nodes].sort((a, b) => {
    if (a.sort_order !== b.sort_order) return a.sort_order - b.sort_order
    return a.name.localeCompare(b.name)
  })
}

export function CategoryTree({
  nodes,
  depth = 0,
  canManage,
  onAddChild,
  onEdit,
  onDelete,
  onReorder,
}: CategoryTreeProps) {
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const sorted = useMemo(() => sortNodes(nodes), [nodes])

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over || active.id === over.id || !onReorder) return

    const activeIdx = sorted.findIndex((n) => n.id === active.id)
    const overIdx = sorted.findIndex((n) => n.id === over.id)
    if (activeIdx === -1 || overIdx === -1) return

    const parentId = sorted[0]?.parent_id ?? null

    // Compute new sort_order as the average of the neighbours at the target position
    const newSorted = [...sorted]
    const [moved] = newSorted.splice(activeIdx, 1)
    newSorted.splice(overIdx, 0, moved)

    const prev = newSorted[overIdx - 1]
    const next = newSorted[overIdx + 1]

    let newOrder: number
    if (prev && next) {
      newOrder = Math.round((prev.sort_order + next.sort_order) / 2)
    } else if (prev) {
      newOrder = prev.sort_order + 1
    } else if (next) {
      newOrder = next.sort_order - 1
    } else {
      newOrder = 0
    }

    // If collision (same value), fall back to sequential renumber — caller handles this
    if (newOrder === prev?.sort_order || newOrder === next?.sort_order) {
      newOrder = overIdx
    }

    onReorder(parentId, String(active.id), newOrder)
  }

  if (!canManage) {
    return (
      <div>
        {sorted.map((node) => (
          <CategoryTreeNode
            key={node.id}
            node={node}
            depth={depth}
            canManage={false}
            onAddChild={onAddChild}
            onEdit={onEdit}
            onDelete={onDelete}
          />
        ))}
      </div>
    )
  }

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
      <SortableContext items={sorted.map((n) => n.id)} strategy={verticalListSortingStrategy}>
        <div>
          {sorted.map((node) => (
            <CategoryTreeNode
              key={node.id}
              node={node}
              depth={depth}
              canManage={canManage}
              onAddChild={onAddChild}
              onEdit={onEdit}
              onDelete={onDelete}
            />
          ))}
        </div>
      </SortableContext>
    </DndContext>
  )
}
