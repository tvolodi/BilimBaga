import { type Department } from '@/api/departments'
import { DepartmentTreeNode } from './DepartmentTreeNode'

interface Props {
  nodes: Department[]
  depth?: number
  canManage: boolean
  onAddChild: (node: Department) => void
  onRename: (node: Department) => void
  onDelete: (node: Department) => void
}

export function DepartmentTree({ nodes, depth = 0, canManage, onAddChild, onRename, onDelete }: Props) {
  return (
    <div>
      {nodes.map((node) => (
        <DepartmentTreeNode
          key={node.id}
          node={node}
          depth={depth}
          canManage={canManage}
          onAddChild={onAddChild}
          onRename={onRename}
          onDelete={onDelete}
        />
      ))}
    </div>
  )
}
