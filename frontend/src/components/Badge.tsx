interface BadgeProps { status: string }

const config: Record<string, { label: string; cls: string; dot?: string }> = {
  running:   { label: 'Ejecutando',   cls: 'bg-blue-50 text-blue-700 border border-blue-200',   dot: 'bg-blue-500 pulse' },
  completed: { label: 'Completado',   cls: 'bg-green-50 text-green-700 border border-green-200' },
  failed:    { label: 'Fallido',      cls: 'bg-red-50 text-red-600 border border-red-200' },
  warning:   { label: 'Advertencia', cls: 'bg-amber-50 text-amber-700 border border-amber-200' },
  online:    { label: 'Online',       cls: 'bg-emerald-50 text-emerald-700 border border-emerald-200', dot: 'bg-emerald-500' },
  offline:   { label: 'Offline',      cls: 'bg-slate-100 text-slate-500 border border-slate-200' },
}

export function Badge({ status }: BadgeProps) {
  const c = config[status] ?? { label: status, cls: 'bg-slate-100 text-slate-500' }
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium ${c.cls}`}>
      {c.dot && <span className={`w-1.5 h-1.5 rounded-full inline-block ${c.dot}`} />}
      {c.label}
    </span>
  )
}
