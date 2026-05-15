import { Card, CardContent, CardHeader } from '@/components/ui/card'

export function ExamCardSkeleton() {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div className="h-5 w-3/4 rounded bg-muted animate-pulse" />
          <div className="h-5 w-16 rounded-full bg-muted animate-pulse" />
        </div>
        <div className="h-4 w-full rounded bg-muted animate-pulse mt-2" />
        <div className="h-4 w-2/3 rounded bg-muted animate-pulse mt-1" />
      </CardHeader>
      <CardContent>
        <div className="flex gap-2 mb-3">
          <div className="h-3 w-14 rounded bg-muted animate-pulse" />
          <div className="h-3 w-14 rounded bg-muted animate-pulse" />
          <div className="h-3 w-20 rounded bg-muted animate-pulse" />
        </div>
        <div className="h-3 w-20 rounded bg-muted animate-pulse mb-4" />
        <div className="h-10 w-full rounded bg-muted animate-pulse" />
      </CardContent>
    </Card>
  )
}
