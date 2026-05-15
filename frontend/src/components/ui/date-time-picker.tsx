import * as React from 'react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

/**
 * DateTimePicker — a simple composite using two native inputs (date + time).
 * Returns value as an ISO 8601 string (UTC), or null when cleared.
 */

interface DateTimePickerProps {
  id?: string
  label?: string
  value: string | null
  onChange: (value: string | null) => void
  disabled?: boolean
  className?: string
}

function toDateTimeLocal(iso: string | null): { date: string; time: string } {
  if (!iso) return { date: '', time: '' }
  try {
    const d = new Date(iso)
    const date = d.toISOString().slice(0, 10)
    const time = d.toISOString().slice(11, 16)
    return { date, time }
  } catch {
    return { date: '', time: '' }
  }
}

function fromDateAndTime(date: string, time: string): string | null {
  if (!date) return null
  const timeVal = time || '00:00'
  return new Date(`${date}T${timeVal}:00.000Z`).toISOString()
}

export function DateTimePicker({ id, label, value, onChange, disabled, className }: DateTimePickerProps) {
  const { date, time } = toDateTimeLocal(value)
  const [localDate, setLocalDate] = React.useState(date)
  const [localTime, setLocalTime] = React.useState(time)

  React.useEffect(() => {
    const { date: d, time: t } = toDateTimeLocal(value)
    setLocalDate(d)
    setLocalTime(t)
  }, [value])

  function handleDateChange(e: React.ChangeEvent<HTMLInputElement>) {
    const newDate = e.target.value
    setLocalDate(newDate)
    onChange(fromDateAndTime(newDate, localTime))
  }

  function handleTimeChange(e: React.ChangeEvent<HTMLInputElement>) {
    const newTime = e.target.value
    setLocalTime(newTime)
    onChange(fromDateAndTime(localDate, newTime))
  }

  return (
    <div className={cn('space-y-1', className)}>
      {label && <Label htmlFor={id}>{label}</Label>}
      <div className="flex gap-2">
        <Input
          id={id}
          type="date"
          value={localDate}
          onChange={handleDateChange}
          disabled={disabled}
          className="flex-1"
        />
        <Input
          type="time"
          value={localTime}
          onChange={handleTimeChange}
          disabled={disabled || !localDate}
          className="w-32"
        />
      </div>
    </div>
  )
}
