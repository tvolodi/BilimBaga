import { Fragment, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ChevronDown, ChevronRight } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { AuditEntry } from '@/api/audit'

interface AuditLogTableProps {
  entries: AuditEntry[]
}

function formatLocalDatetime(isoString: string): string {
  return new Date(isoString).toLocaleString()
}

function truncateId(id: string | null): string {
  if (!id) return '—'
  return id.slice(0, 8)
}

export function AuditLogTable({ entries }: AuditLogTableProps) {
  const { t } = useTranslation()
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set())

  function toggleRow(id: string) {
    setExpandedRows((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  if (entries.length === 0) {
    return (
      <div className="rounded-md border p-8 text-center text-sm text-muted-foreground">
        —
      </div>
    )
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-8" />
            <TableHead>{t('audit.col_timestamp')}</TableHead>
            <TableHead>{t('audit.col_actor')}</TableHead>
            <TableHead>{t('audit.col_action')}</TableHead>
            <TableHead>{t('audit.col_entity_type')}</TableHead>
            <TableHead>{t('audit.col_entity_id')}</TableHead>
            <TableHead>{t('audit.col_ip')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {entries.map((entry) => {
            const isExpanded = expandedRows.has(entry.id)
            return (
              <Fragment key={entry.id}>
                <TableRow
                  className="cursor-pointer"
                  onClick={() => toggleRow(entry.id)}
                >
                  <TableCell className="pr-0">
                    {isExpanded ? (
                      <ChevronDown size={14} className="text-muted-foreground" />
                    ) : (
                      <ChevronRight size={14} className="text-muted-foreground" />
                    )}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-xs">
                    {formatLocalDatetime(entry.created_at)}
                  </TableCell>
                  <TableCell>
                    {entry.actor_id ? (
                      <Link
                        to={`/admin/users/${entry.actor_id}`}
                        className="text-primary hover:underline"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {entry.actor_name ?? entry.actor_id}
                      </Link>
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <code className="rounded bg-muted px-1 py-0.5 text-xs">{entry.action}</code>
                  </TableCell>
                  <TableCell className="text-xs">{entry.entity_type ?? '—'}</TableCell>
                  <TableCell>
                    {entry.entity_id ? (
                      <span
                        title={entry.entity_id}
                        className="cursor-help font-mono text-xs"
                      >
                        {truncateId(entry.entity_id)}…
                      </span>
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell className="text-xs">{entry.ip_address ?? '—'}</TableCell>
                </TableRow>

                {isExpanded && (
                  <TableRow>
                    <TableCell colSpan={7} className="bg-muted/30 py-2">
                      <pre className="overflow-auto rounded bg-muted p-2 text-xs">
                        {JSON.stringify(entry.metadata, null, 2)}
                      </pre>
                    </TableCell>
                  </TableRow>
                )}
              </Fragment>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}
