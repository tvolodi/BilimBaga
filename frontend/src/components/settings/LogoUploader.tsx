import { useRef, useState, type DragEvent, type ChangeEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Upload } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Label } from '@/components/ui/label'

const MAX_BYTES = 1024 * 1024
const ACCEPTED_MIME = ['image/png', 'image/jpeg', 'image/svg+xml', 'image/webp']
const ACCEPT_ATTR = ACCEPTED_MIME.join(',')

interface LogoUploaderProps {
  currentLogoUrl: string
  onFile: (dataUrl: string) => void
}

export function LogoUploader({ currentLogoUrl, onFile }: LogoUploaderProps) {
  const { t } = useTranslation()
  const inputRef = useRef<HTMLInputElement>(null)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isDragging, setIsDragging] = useState(false)

  function validateAndRead(file: File) {
    if (!ACCEPTED_MIME.includes(file.type)) {
      setError(t('settings.branding.logoTypeError'))
      return
    }
    if (file.size > MAX_BYTES) {
      setError(t('settings.branding.logoSizeError'))
      return
    }

    const reader = new FileReader()
    reader.onload = () => {
      const dataUrl = String(reader.result ?? '')
      setPreviewUrl(dataUrl)
      setError(null)
      onFile(dataUrl)
    }
    reader.onerror = () => setError(t('settings.branding.logoReadError'))
    reader.readAsDataURL(file)
  }

  function handleFileInput(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (file) validateAndRead(file)
  }

  function handleDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setIsDragging(false)
    const file = e.dataTransfer.files?.[0]
    if (file) validateAndRead(file)
  }

  function handleDragOver(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setIsDragging(true)
  }

  function handleDragLeave() {
    setIsDragging(false)
  }

  function handleClick() {
    inputRef.current?.click()
  }

  const displayedSrc = previewUrl ?? currentLogoUrl

  return (
    <div className="space-y-2">
      <Label>{t('settings.branding.logo')}</Label>
      <div
        role="button"
        tabIndex={0}
        onClick={handleClick}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            handleClick()
          }
        }}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        aria-label={t('settings.branding.logoDragDrop')}
        className={cn(
          'flex flex-col items-center justify-center gap-3 rounded-md border-2 border-dashed p-6 cursor-pointer transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
          isDragging ? 'border-primary bg-primary/5' : 'border-input bg-background hover:bg-muted',
        )}
      >
        <img
          src={displayedSrc}
          alt={t('settings.branding.logo')}
          className="h-16 max-w-full object-contain"
          onError={(e) => {
            (e.currentTarget as HTMLImageElement).style.visibility = 'hidden'
          }}
        />
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Upload size={16} />
          <span>{t('settings.branding.logoDragDrop')}</span>
        </div>
        <span className="text-xs text-muted-foreground">{t('settings.branding.logoHint')}</span>

        <input
          ref={inputRef}
          type="file"
          accept={ACCEPT_ATTR}
          className="sr-only"
          onChange={handleFileInput}
          aria-label={t('settings.branding.logo')}
        />
      </div>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
    </div>
  )
}
