import { Skeleton } from '@/components/ui/skeleton'

export function EmployeeRecordSkeleton() {
  return (
    <div className="space-y-8">
      {/* Header skeleton */}
      <div className="flex items-start gap-4 p-6 rounded-lg border bg-card">
        <Skeleton className="w-16 h-16 rounded-full flex-shrink-0" />
        <div className="flex-1 space-y-3">
          <Skeleton className="h-7 w-52" />
          <div className="flex gap-2">
            <Skeleton className="h-5 w-24 rounded-full" />
            <Skeleton className="h-5 w-16 rounded-full" />
          </div>
          <div className="flex gap-4">
            <Skeleton className="h-4 w-32" />
            <Skeleton className="h-4 w-40" />
          </div>
        </div>
      </div>

      {/* Table skeleton */}
      <div className="space-y-3">
        <Skeleton className="h-6 w-36" />
        <div className="rounded-md border overflow-hidden">
          <div className="flex gap-4 px-4 py-3 border-b bg-muted/30">
            {[160, 90, 70, 80, 80, 90].map((w, i) => (
              <Skeleton key={i} className="h-4" style={{ width: w }} />
            ))}
          </div>
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex gap-4 px-4 py-3 border-b last:border-0">
              {[160, 90, 70, 80, 80, 90].map((w, j) => (
                <Skeleton key={j} className="h-4" style={{ width: w }} />
              ))}
            </div>
          ))}
        </div>
      </div>

      {/* Cards skeleton */}
      <div className="space-y-3">
        <Skeleton className="h-6 w-36" />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="rounded-lg border p-6 space-y-4">
              <Skeleton className="h-5 w-24" />
              <div className="space-y-1">
                <Skeleton className="h-3 w-36" />
                <Skeleton className="h-8 w-10" />
              </div>
              <div className="space-y-1">
                <Skeleton className="h-3 w-28" />
                <Skeleton className="h-4 w-32" />
              </div>
              <div className="space-y-1.5">
                <Skeleton className="h-3 w-28" />
                <Skeleton className="h-7 w-full rounded-full" />
                <Skeleton className="h-7 w-full rounded-full" />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
