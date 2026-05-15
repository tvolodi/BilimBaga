import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface FlagButtonProps {
  isFlagged: boolean
  onToggle: () => void
}

export function FlagButton({ isFlagged, onToggle }: FlagButtonProps) {
  const { t } = useTranslation()

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={onToggle}
      className={cn(
        'shrink-0 text-xs',
        isFlagged ? 'text-yellow-600 hover:text-yellow-700' : 'text-muted-foreground',
      )}
      aria-pressed={isFlagged}
    >
      {isFlagged ? `⚑ ${t('exam.taking.unflagButton')}` : `⚐ ${t('exam.taking.flagButton')}`}
    </Button>
  )
}
