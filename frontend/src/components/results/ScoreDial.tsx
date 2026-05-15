interface ScoreDialProps {
  scorePct: number
  primaryColor: string
}

export function ScoreDial({ scorePct, primaryColor }: ScoreDialProps) {
  const radius = 54
  const circumference = 2 * Math.PI * radius
  const clampedPct = Math.min(100, Math.max(0, scorePct))
  const offset = circumference - (clampedPct / 100) * circumference

  return (
    <svg
      width="140"
      height="140"
      viewBox="0 0 140 140"
      aria-label={`${Math.round(clampedPct)}%`}
      role="img"
    >
      {/* Background track */}
      <circle
        cx="70"
        cy="70"
        r={radius}
        fill="none"
        stroke="#e5e7eb"
        strokeWidth="12"
      />
      {/* Progress arc */}
      <circle
        cx="70"
        cy="70"
        r={radius}
        fill="none"
        stroke={primaryColor}
        strokeWidth="12"
        strokeLinecap="round"
        strokeDasharray={circumference}
        strokeDashoffset={offset}
        transform="rotate(-90 70 70)"
        style={{ transition: 'stroke-dashoffset 0.6s ease' }}
      />
      {/* Percentage text */}
      <text
        x="70"
        y="70"
        textAnchor="middle"
        dominantBaseline="central"
        fontSize="22"
        fontWeight="700"
        fill="currentColor"
      >
        {Math.round(clampedPct)}%
      </text>
    </svg>
  )
}
