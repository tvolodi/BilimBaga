import { useTranslation } from 'react-i18next'
import { formatDistanceToNow } from 'date-fns'
import { Card, CardHeader, CardContent } from '@/components/ui/card'
import { ExamStatusChip } from './ExamStatusChip'
import type { TrackSummary } from '@/api/employees'

interface TrackProgressCardsProps {
  tracks: TrackSummary[]
}

const TRACK_NAME_KEY: Record<string, string> = {
  security: 'employee_record.track_security',
  safety: 'employee_record.track_safety',
  loyalty: 'employee_record.track_loyalty',
}

export function TrackProgressCards({ tracks }: TrackProgressCardsProps) {
  const { t } = useTranslation()

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      {tracks.map((track) => (
        <Card key={track.track}>
          <CardHeader>
            <h3 className="text-base font-semibold">
              {t(TRACK_NAME_KEY[track.track] ?? `employee_record.track_${track.track}`)}
            </h3>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-xs text-muted-foreground uppercase tracking-wide">
                {t('employee_record.questions_answered')}
              </p>
              <p className="text-2xl font-bold mt-0.5">{track.questions_answered}</p>
            </div>

            <div>
              <p className="text-xs text-muted-foreground uppercase tracking-wide">
                {t('employee_record.last_activity')}
              </p>
              <p className="text-sm mt-0.5">
                {track.last_activity
                  ? formatDistanceToNow(new Date(track.last_activity), { addSuffix: true })
                  : t('employee_record.never')}
              </p>
            </div>

            {track.required_exams.length > 0 && (
              <div>
                <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1.5">
                  {t('employee_record.required_exams')}
                </p>
                <div className="flex flex-col gap-1.5">
                  {track.required_exams.map((exam) => (
                    <ExamStatusChip
                      key={exam.exam_id}
                      title={exam.title}
                      passed={exam.passed}
                      attempts={exam.attempts}
                    />
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
