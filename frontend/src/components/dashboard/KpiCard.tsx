import type { ReactNode } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'

interface KpiCardProps {
  icon: ReactNode
  label: string
  value: string | number
  description?: string
  className?: string
}

export function KpiCard({ icon, label, value, description, className }: KpiCardProps) {
  return (
    <Card className={cn('', className)}>
      <CardContent className="p-6">
        <div className="flex items-start gap-4">
          <div className="flex-shrink-0 rounded-lg bg-primary/10 p-2.5 text-primary">
            {icon}
          </div>
          <div className="min-w-0">
            <p className="text-sm font-medium text-muted-foreground truncate">{label}</p>
            <p className="mt-1 text-2xl font-bold tracking-tight">{value}</p>
            {description && (
              <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
