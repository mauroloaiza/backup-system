/**
 * Ant mascot — the brand character of SMC Backup.
 *
 * Three variants:
 *   <AntIcon />    — monochrome silhouette for dark badges (uses currentColor).
 *                    Use inside the blue rounded container in the sidebar/login header.
 *   <AntMascot />  — full color, detailed, carries a backup disk. Use in login card
 *                    and empty states.
 *   <AntSpot />    — tiny decorative ant for section headers and subtle accents.
 */

interface CommonProps {
  className?: string
  title?: string
}

// ── Monochrome silhouette (uses currentColor) ────────────────────────────────

export function AntIcon({ className = '', title = 'Ant' }: CommonProps) {
  return (
    <svg viewBox="0 0 64 64" className={className} fill="currentColor" role="img" aria-label={title}>
      {/* Legs */}
      <g stroke="currentColor" strokeWidth="2.8" strokeLinecap="round" fill="none">
        <path d="M26 34 L18 30" />
        <path d="M26 36 L18 42" />
        <path d="M32 34 L32 24" />
        <path d="M32 36 L32 46" />
        <path d="M38 34 L46 30" />
        <path d="M38 36 L46 42" />
      </g>
      {/* Three body segments (defining ant trait) */}
      <ellipse cx="44" cy="35" rx="9" ry="7.5" />
      <circle cx="32" cy="35" r="5.5" />
      <circle cx="22" cy="33" r="6.5" />
      {/* Antennae */}
      <g stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" fill="none">
        <path d="M19 28 Q17 21 21 18" />
        <path d="M23 27 Q23 20 27 18" />
      </g>
      <circle cx="21" cy="18" r="1.6" />
      <circle cx="27" cy="18" r="1.6" />
    </svg>
  )
}

// ── Full color mascot carrying a backup disk ─────────────────────────────────

export function AntMascot({ className = '', title = 'Tita, the SMC Backup ant' }: CommonProps) {
  return (
    <svg viewBox="0 0 120 100" className={className} fill="none" role="img" aria-label={title}>
      {/* Ground shadow */}
      <ellipse cx="60" cy="90" rx="34" ry="3" fill="#000" opacity="0.08" />

      {/* Legs (behind body) */}
      <g stroke="#1e2a5c" strokeWidth="3" strokeLinecap="round" fill="none">
        <path d="M48 62 L36 52" />
        <path d="M48 66 L36 82" />
        <path d="M60 62 L60 44" />
        <path d="M60 66 L60 86" />
        <path d="M72 62 L86 52" />
        <path d="M72 66 L86 82" />
      </g>

      {/* Abdomen — largest segment, back */}
      <ellipse cx="86" cy="62" rx="16" ry="14" fill="#1e2a5c" />
      <ellipse cx="82" cy="58" rx="6" ry="4" fill="#2d3e7a" opacity="0.6" />

      {/* Thorax — middle, narrower (the "waist" of the ant) */}
      <ellipse cx="60" cy="62" rx="10" ry="9" fill="#1e2a5c" />

      {/* Head — round, front */}
      <circle cx="38" cy="58" r="13" fill="#1e2a5c" />
      <circle cx="34" cy="54" r="4" fill="#2d3e7a" opacity="0.6" />

      {/* Antennae with rounded tips */}
      <g stroke="#1e2a5c" strokeWidth="2.8" strokeLinecap="round" fill="none">
        <path d="M32 48 Q26 32 32 24" />
        <path d="M40 46 Q42 30 48 24" />
      </g>
      <circle cx="32" cy="24" r="3" fill="#4361ee" />
      <circle cx="48" cy="24" r="3" fill="#4361ee" />

      {/* Friendly eye */}
      <circle cx="42" cy="55" r="3.2" fill="#ffffff" />
      <circle cx="43" cy="55.5" r="1.8" fill="#1e2a5c" />
      <circle cx="43.5" cy="55" r="0.6" fill="#ffffff" />

      {/* Smile */}
      <path d="M36 64 Q40 67 44 64" stroke="#ffffff" strokeWidth="1.4" strokeLinecap="round" fill="none" opacity="0.8" />

      {/* Backup disk strapped to the abdomen */}
      <g transform="translate(74 38)">
        {/* Strap around the abdomen */}
        <path d="M-2 16 Q12 8 26 16" stroke="#0b1538" strokeWidth="1.5" fill="none" opacity="0.5" />
        {/* Box/disk body */}
        <rect x="0" y="0" width="24" height="16" rx="2.4" fill="#ffffff" stroke="#4361ee" strokeWidth="1.5" />
        <rect x="0" y="0" width="24" height="5" rx="2.4" fill="#4361ee" />
        <rect x="0" y="3" width="24" height="2" fill="#4361ee" />
        {/* Disk indicator light */}
        <circle cx="19" cy="10" r="1.6" fill="#22c55e">
          <animate attributeName="opacity" values="1;0.3;1" dur="2s" repeatCount="indefinite" />
        </circle>
        {/* Label lines */}
        <rect x="3" y="8" width="10" height="1.2" rx="0.4" fill="#cbd5e1" />
        <rect x="3" y="11" width="7" height="1.2" rx="0.4" fill="#cbd5e1" />
      </g>
    </svg>
  )
}

// ── Tiny decorative spot (for section headers) ───────────────────────────────

export function AntSpot({ className = '', title = 'Ant' }: CommonProps) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="currentColor" role="img" aria-label={title}>
      <ellipse cx="17" cy="13" rx="4" ry="3.2" />
      <circle cx="12" cy="13" r="2.4" />
      <circle cx="7.5" cy="12" r="2.8" />
      <g stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" fill="none">
        <path d="M6 9 Q5 5 7 4" />
        <path d="M9 9 Q9 5 11 4" />
        <path d="M10 13 L10 8" />
        <path d="M10 14 L10 18" />
        <path d="M6 13 L3 11" />
        <path d="M6 14 L3 17" />
        <path d="M14 13 L17 11" />
        <path d="M14 14 L17 17" />
      </g>
    </svg>
  )
}
