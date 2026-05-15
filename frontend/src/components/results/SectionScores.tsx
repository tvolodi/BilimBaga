import { useTranslation } from 'react-i18next'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import type { SectionScore } from '@/api/sessions'

interface SectionScoresProps {
  sections: SectionScore[]
}

export function SectionScores({ sections }: SectionScoresProps) {
  const { t } = useTranslation()

  if (sections.length === 0) return null

  return (
    <div className="w-full">
      <h3 className="mb-3 text-sm font-semibold text-muted-foreground uppercase tracking-wide">
        {t('result.section_scores')}
      </h3>
      <div className="flex flex-row flex-wrap gap-3">
        {sections.map((section) => (
          <Card key={section.section_id} className="min-w-[140px] flex-1">
            <CardHeader className="pb-1 pt-3 px-4">
              <p className="text-xs font-medium text-muted-foreground truncate">
                {section.title}
              </p>
            </CardHeader>
            <CardContent className="pb-3 px-4">
              <span className="text-2xl font-bold">{Math.round(section.score_pct)}%</span>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
