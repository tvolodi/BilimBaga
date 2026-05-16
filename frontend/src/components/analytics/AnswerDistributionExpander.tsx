import { useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { AnswerDistributionItem } from '@/api/analytics'

interface AnswerDistributionExpanderProps {
  distribution: AnswerDistributionItem[]
}

export function AnswerDistributionExpander({ distribution }: AnswerDistributionExpanderProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  // Hidden for short_text questions (no distribution items)
  if (!distribution || distribution.length === 0) return null

  const total = distribution.reduce((sum, d) => sum + d.select_count, 0)

  return (
    <div className="mt-1">
      <button
        type="button"
        className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
        onClick={() => setOpen(!open)}
        aria-expanded={open}
      >
        {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        {t('exam_analytics.answer_distribution')}
      </button>
      {open && (
        <div className="mt-2 space-y-1.5 pl-2">
          {distribution.map((item) => {
            const pct = total > 0 ? (item.select_count / total) * 100 : 0
            return (
              <div key={item.option_id} className="text-xs">
                <div className="flex justify-between mb-0.5">
                  <span className="text-foreground truncate max-w-[240px]">
                    {item.option_text}
                  </span>
                  <span className="text-muted-foreground ml-2 flex-shrink-0">
                    {item.select_count} ({pct.toFixed(0)}%)
                  </span>
                </div>
                <div className="h-1.5 rounded-full bg-muted overflow-hidden">
                  <div
                    className="h-full rounded-full bg-primary"
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
