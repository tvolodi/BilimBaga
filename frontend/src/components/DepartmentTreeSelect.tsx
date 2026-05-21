import { useState, useRef, useCallback, useId } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronRight, ChevronDown, Loader2, AlertTriangle, X } from 'lucide-react'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { useDepartments, findDepartmentById, type Department } from '@/api/departments'

interface DepartmentTreeSelectProps {
  value: string | null
  onChange: (id: string | null) => void
  disabled?: boolean
  placeholder?: string
  clearable?: boolean
  className?: string
  'aria-label'?: string
}

function nodeMatchesSearch(node: Department, query: string): boolean {
  if (node.name.toLowerCase().includes(query.toLowerCase())) return true
  return node.children.some((child) => nodeMatchesSearch(child, query))
}

interface TreeNodeProps {
  node: Department
  depth: number
  selectedId: string | null
  query: string
  expandedIds: Set<string>
  onToggle: (id: string) => void
  onSelect: (id: string) => void
  focusedId: string | null
  nodeIds: string[]
}

function TreeNode({
  node,
  depth,
  selectedId,
  query,
  expandedIds,
  onToggle,
  onSelect,
  focusedId,
  nodeIds,
}: TreeNodeProps) {
  const hasChildren = node.children.length > 0
  const isExpanded = expandedIds.has(node.id)
  const isSelected = selectedId === node.id
  const isFocused = focusedId === node.id

  const isVisible = query === '' || nodeMatchesSearch(node, query)
  if (!isVisible) return null

  const autoExpand = query !== '' && hasChildren && node.children.some((c) => nodeMatchesSearch(c, query))
  const showExpanded = isExpanded || autoExpand

  return (
    <li role="none">
      <div
        id={`tree-node-${node.id}`}
        role="treeitem"
        aria-selected={isSelected}
        aria-expanded={hasChildren ? showExpanded : undefined}
        tabIndex={isFocused ? 0 : -1}
        data-nodeid={node.id}
        className={cn(
          'flex items-center gap-1 rounded px-2 py-1.5 text-sm cursor-pointer select-none outline-none',
          isSelected && 'bg-accent text-accent-foreground',
          !isSelected && 'hover:bg-muted',
          isFocused && 'ring-2 ring-ring ring-offset-1',
        )}
        style={{ paddingLeft: `${(depth + 1) * 0.75}rem` }}
        onClick={() => onSelect(node.id)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onSelect(node.id)
          } else if (e.key === 'ArrowRight' && hasChildren && !showExpanded) {
            e.preventDefault()
            onToggle(node.id)
          } else if (e.key === 'ArrowLeft' && hasChildren && showExpanded) {
            e.preventDefault()
            onToggle(node.id)
          } else if (e.key === 'ArrowDown') {
            e.preventDefault()
            const idx = nodeIds.indexOf(node.id)
            if (idx < nodeIds.length - 1) {
              const nextId = nodeIds[idx + 1]
              document.getElementById(`tree-node-${nextId}`)?.focus()
            }
          } else if (e.key === 'ArrowUp') {
            e.preventDefault()
            const idx = nodeIds.indexOf(node.id)
            if (idx > 0) {
              const prevId = nodeIds[idx - 1]
              document.getElementById(`tree-node-${prevId}`)?.focus()
            }
          }
        }}
      >
        {hasChildren ? (
          <span
            aria-hidden="true"
            className="shrink-0 text-muted-foreground"
            onClick={(e) => {
              e.stopPropagation()
              onToggle(node.id)
            }}
          >
            {showExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          </span>
        ) : (
          <span className="shrink-0 w-[14px]" aria-hidden="true" />
        )}
        <span className="truncate">{node.name}</span>
      </div>
      {hasChildren && showExpanded && (
        <ul role="group">
          {node.children.map((child) => (
            <TreeNode
              key={child.id}
              node={child}
              depth={depth + 1}
              selectedId={selectedId}
              query={query}
              expandedIds={expandedIds}
              onToggle={onToggle}
              onSelect={onSelect}
              focusedId={focusedId}
              nodeIds={nodeIds}
            />
          ))}
        </ul>
      )}
    </li>
  )
}

function collectVisibleIds(
  nodes: Department[],
  expandedIds: Set<string>,
  query: string,
): string[] {
  const ids: string[] = []
  for (const node of nodes) {
    const isVisible = query === '' || nodeMatchesSearch(node, query)
    if (!isVisible) continue
    ids.push(node.id)
    const autoExpand = query !== '' && node.children.length > 0 && node.children.some((c) => nodeMatchesSearch(c, query))
    if (expandedIds.has(node.id) || autoExpand) {
      ids.push(...collectVisibleIds(node.children, expandedIds, query))
    }
  }
  return ids
}

export function DepartmentTreeSelect({
  value,
  onChange,
  disabled = false,
  placeholder,
  clearable = true,
  className,
  'aria-label': ariaLabel,
}: DepartmentTreeSelectProps) {
  const { t } = useTranslation()
  const { data: departments, isLoading, isError, refetch } = useDepartments()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())
  const [focusedId, setFocusedId] = useState<string | null>(null)
  const searchRef = useRef<HTMLInputElement>(null)
  const treeId = useId()

  const selectedDept = value && departments ? findDepartmentById(departments, value) : null
  const displayText = selectedDept ? selectedDept.name : (placeholder ?? t('departmentTree.placeholder'))

  const handleToggle = useCallback((id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  const handleSelect = useCallback((id: string) => {
    onChange(id)
    setOpen(false)
    setQuery('')
  }, [onChange])

  const handleClear = useCallback((e: React.MouseEvent) => {
    e.stopPropagation()
    onChange(null)
  }, [onChange])

  const visibleIds = departments ? collectVisibleIds(departments, expandedIds, query) : []

  return (
    <Popover open={open} onOpenChange={(v) => { setOpen(v); if (!v) setQuery('') }}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          role="combobox"
          aria-haspopup="tree"
          aria-expanded={open}
          aria-label={ariaLabel}
          aria-controls={open ? treeId : undefined}
          disabled={disabled || isLoading}
          className={cn('w-full justify-between font-normal', !selectedDept && 'text-muted-foreground', className)}
        >
          <span className="truncate">
            {isLoading ? (
              <span className="flex items-center gap-2">
                <Loader2 size={14} className="animate-spin" aria-hidden="true" />
                {t('departmentTree.loading')}
              </span>
            ) : (
              displayText
            )}
          </span>
          {clearable && value && !disabled && (
            <X
              size={14}
              className="ml-2 shrink-0 text-muted-foreground hover:text-foreground"
              aria-label={t('departmentTree.clearSelection')}
              onClick={handleClear}
              aria-hidden="false"
            />
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-72 p-0" onOpenAutoFocus={(e) => { e.preventDefault(); searchRef.current?.focus() }}>
        <div className="p-2 border-b">
          <Input
            ref={searchRef}
            placeholder={t('departmentTree.search')}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="h-8 text-sm"
            aria-label={t('departmentTree.search')}
          />
        </div>
        <div className="max-h-64 overflow-y-auto p-1">
          {isError && (
            <div className="flex flex-col items-center gap-2 py-4 text-sm text-muted-foreground">
              <AlertTriangle size={16} aria-hidden="true" className="text-destructive" />
              <span>{t('departmentTree.error')}</span>
              <Button size="sm" variant="outline" onClick={() => refetch()}>
                {t('departmentTree.retry')}
              </Button>
            </div>
          )}
          {isLoading && (
            <div className="space-y-1 p-1">
              <Skeleton className="h-7 w-full" />
              <Skeleton className="h-7 w-3/4 ml-4" />
              <Skeleton className="h-7 w-full" />
            </div>
          )}
          {!isLoading && !isError && departments?.length === 0 && (
            <p className="py-4 text-center text-sm text-muted-foreground">{t('departmentTree.empty')}</p>
          )}
          {!isLoading && !isError && departments && departments.length > 0 && (
            <ul
              id={treeId}
              role="tree"
              aria-label={ariaLabel ?? t('departmentTree.placeholder')}
              className="space-y-0.5"
              onFocus={(e) => {
                const nodeId = (e.target as HTMLElement).dataset.nodeid
                if (nodeId) setFocusedId(nodeId)
              }}
              onBlur={() => setFocusedId(null)}
            >
              {departments.map((node) => (
                <TreeNode
                  key={node.id}
                  node={node}
                  depth={0}
                  selectedId={value}
                  query={query}
                  expandedIds={expandedIds}
                  onToggle={handleToggle}
                  onSelect={handleSelect}
                  focusedId={focusedId}
                  nodeIds={visibleIds}
                />
              ))}
            </ul>
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}
