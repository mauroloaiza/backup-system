import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Cloud, Server, HardDrive, Database } from 'lucide-react'
import { fetchDestinations, type DestinationAggregate } from '@/lib/api'
import { fmtDate, fmtBytes } from '@/lib/utils'

// ── Type styling ────────────────────────────────────────────────────────────

const TYPE_META: Record<DestinationAggregate['type'], {
  label: string
  icon: React.ComponentType<{ className?: string }>
  iconBg: string
  iconColor: string
}> = {
  s3:    { label: 'S3',    icon: Cloud,     iconBg: 'bg-blue-100',  iconColor: 'text-blue-500'  },
  sftp:  { label: 'SFTP',  icon: Server,    iconBg: 'bg-green-100', iconColor: 'text-green-600' },
  local: { label: 'Local', icon: HardDrive, iconBg: 'bg-amber-100', iconColor: 'text-amber-600' },
}

const STATUS_COLOR: Record<DestinationAggregate['node_status'], string> = {
  online:  'bg-emerald-400',
  stale:   'bg-amber-400',
  offline: 'bg-red-400',
}

const STATUS_LABEL: Record<DestinationAggregate['node_status'], string> = {
  online:  'En línea',
  stale:   'Intermitente',
  offline: 'Sin contacto',
}

const LAST_STATUS_COLOR: Record<string, string> = {
  completed: 'text-emerald-600',
  failed:    'text-red-500',
  warning:   'text-amber-500',
  running:   'text-blue-500',
}

// ── Page ────────────────────────────────────────────────────────────────────

export function Destinations() {
  const { data = [], isLoading } = useQuery({
    queryKey: ['destinations'],
    queryFn: fetchDestinations,
    refetchInterval: 15000,
  })

  // Summary cards: counts by type + aggregated bytes
  const summary = useMemo(() => {
    const acc: Record<string, { count: number; bytes: number; nodes: Set<string> }> = {}
    data.forEach(d => {
      const row = (acc[d.type] ??= { count: 0, bytes: 0, nodes: new Set() })
      row.count += 1
      row.bytes += d.bytes_backed_up
      row.nodes.add(d.node_id)
    })
    return Object.entries(acc).map(([type, v]) => ({
      type: type as DestinationAggregate['type'],
      destinations: v.count,
      nodes: v.nodes.size,
      bytes: v.bytes,
    }))
  }, [data])

  // Group by type+target to deduplicate destinations shared across nodes
  const grouped = useMemo(() => {
    const map = new Map<string, { type: string; target: string; details: any; rows: DestinationAggregate[] }>()
    for (const d of data) {
      const key = `${d.type}::${d.target}`
      const g = map.get(key)
      if (g) g.rows.push(d)
      else map.set(key, { type: d.type, target: d.target, details: d.details, rows: [d] })
    }
    return Array.from(map.values()).sort(
      (a, b) =>
        b.rows.reduce((s, r) => s + r.bytes_backed_up, 0) -
        a.rows.reduce((s, r) => s + r.bytes_backed_up, 0),
    )
  }, [data])

  return (
    <div className="p-6 space-y-5">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-xl font-bold text-gray-800">Destinos</h1>
          <p className="text-sm text-gray-400 mt-0.5">
            Almacenamiento reportado por los agentes (desde <code className="bg-gray-100 px-1 rounded">agent.yaml</code>)
          </p>
        </div>
      </div>

      {/* Summary cards */}
      {summary.length > 0 && (
        <div className="grid grid-cols-3 gap-4">
          {(['s3', 'sftp', 'local'] as const).map(t => {
            const s = summary.find(x => x.type === t)
            const meta = TYPE_META[t]
            const Icon = meta.icon
            return (
              <div key={t} className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
                <div className="flex items-center gap-3">
                  <div className={`w-10 h-10 rounded-xl flex items-center justify-center ${meta.iconBg}`}>
                    <Icon className={`w-5 h-5 ${meta.iconColor}`} />
                  </div>
                  <div>
                    <p className="text-xs text-gray-400 uppercase tracking-wide font-medium">{meta.label}</p>
                    <p className="text-lg font-bold text-gray-800">
                      {s ? `${s.destinations} destino${s.destinations !== 1 ? 's' : ''}` : '—'}
                    </p>
                  </div>
                </div>
                {s && (
                  <div className="mt-3 flex justify-between text-xs text-gray-500">
                    <span>{s.nodes} {s.nodes === 1 ? 'nodo' : 'nodos'}</span>
                    <span className="font-medium text-gray-700">{fmtBytes(s.bytes)}</span>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {/* Destinations list */}
      {isLoading ? (
        <div className="space-y-2">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="h-24 bg-gray-50 border border-gray-100 rounded-xl animate-pulse" />
          ))}
        </div>
      ) : grouped.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="space-y-3">
          {grouped.map(g => (
            <DestinationCard key={`${g.type}::${g.target}`} type={g.type} target={g.target} details={g.details} rows={g.rows} />
          ))}
        </div>
      )}
    </div>
  )
}

// ── Destination card ────────────────────────────────────────────────────────

function DestinationCard({
  type, target, details, rows,
}: {
  type: string
  target: string
  details: any
  rows: DestinationAggregate[]
}) {
  const meta = TYPE_META[type as DestinationAggregate['type']] ?? TYPE_META.local
  const Icon = meta.icon
  const totalBytes = rows.reduce((s, r) => s + r.bytes_backed_up, 0)
  const totalRuns = rows.reduce((s, r) => s + r.runs_count, 0)
  const lastBackup = rows.reduce<string | null>((acc, r) => {
    if (!r.last_backup_at) return acc
    if (!acc || new Date(r.last_backup_at) > new Date(acc)) return r.last_backup_at
    return acc
  }, null)

  return (
    <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
      {/* Header */}
      <div className="flex items-start justify-between px-5 py-4 border-b border-gray-100">
        <div className="flex items-center gap-3 min-w-0">
          <div className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${meta.iconBg}`}>
            <Icon className={`w-5 h-5 ${meta.iconColor}`} />
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-2 mb-0.5">
              <span className="text-[10px] bg-gray-100 text-gray-500 px-1.5 py-0.5 rounded font-medium uppercase tracking-wide">
                {meta.label}
              </span>
              <span className="text-xs text-gray-400">·</span>
              <span className="text-xs text-gray-500">
                {rows.length} {rows.length === 1 ? 'nodo' : 'nodos'}
              </span>
            </div>
            <p className="font-semibold text-gray-800 text-sm truncate font-mono" title={target}>
              {target}
            </p>
            {details && <DetailsLine details={details} type={type} />}
          </div>
        </div>
        <div className="text-right shrink-0 ml-4">
          <p className="text-sm font-bold text-gray-800">{fmtBytes(totalBytes)}</p>
          <p className="text-[11px] text-gray-400">{totalRuns} {totalRuns === 1 ? 'run' : 'runs'}</p>
          {lastBackup && (
            <p className="text-[10px] text-gray-400 mt-0.5">Últ: {fmtDate(lastBackup)}</p>
          )}
        </div>
      </div>

      {/* Per-node rows */}
      <div className="divide-y divide-gray-50">
        {rows.map(r => (
          <NodeRow key={r.node_id} row={r} />
        ))}
      </div>
    </div>
  )
}

function DetailsLine({ details, type }: { details: Record<string, string>; type: string }) {
  if (type === 's3') {
    return (
      <p className="text-[11px] text-gray-400 mt-0.5">
        {details.region && <span>región {details.region}</span>}
        {details.prefix && <span> · prefix {details.prefix}</span>}
      </p>
    )
  }
  if (type === 'sftp') {
    return (
      <p className="text-[11px] text-gray-400 mt-0.5">
        {details.user && <span>{details.user}@</span>}
        {details.host}{details.port && details.port !== '22' && <span>:{details.port}</span>}
      </p>
    )
  }
  return null
}

function NodeRow({ row }: { row: DestinationAggregate }) {
  const lastStatusClass = row.last_status ? (LAST_STATUS_COLOR[row.last_status] ?? 'text-gray-400') : 'text-gray-400'
  return (
    <div className="px-5 py-2.5 flex items-center gap-3 hover:bg-gray-50/50 transition-colors">
      <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${STATUS_COLOR[row.node_status]}`} title={STATUS_LABEL[row.node_status]} />
      <Database className="w-3.5 h-3.5 text-gray-300 shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="text-xs font-semibold text-gray-700 truncate">{row.node_name}</p>
        <p className="text-[10px] text-gray-400 truncate font-mono">{row.hostname}</p>
      </div>
      <div className="text-right shrink-0">
        <p className="text-xs font-medium text-gray-600">{fmtBytes(row.bytes_backed_up)}</p>
        <p className="text-[10px] text-gray-400">
          {row.runs_count} runs
          {row.last_status && (
            <span className={`ml-1.5 ${lastStatusClass}`}>· {row.last_status}</span>
          )}
        </p>
      </div>
    </div>
  )
}

// ── Empty state ─────────────────────────────────────────────────────────────

function EmptyState() {
  return (
    <div className="bg-white border border-dashed border-gray-200 rounded-xl p-10 text-center shadow-sm">
      <div className="w-12 h-12 mx-auto rounded-xl bg-gray-50 flex items-center justify-center mb-3">
        <Cloud className="w-5 h-5 text-gray-300" />
      </div>
      <p className="text-sm font-medium text-gray-700">Ningún agente ha reportado destinos aún</p>
      <p className="text-xs text-gray-400 mt-1 max-w-md mx-auto">
        Los destinos se configuran en el archivo <code className="bg-gray-100 px-1 rounded">agent.yaml</code> de cada nodo
        (sección <code className="bg-gray-100 px-1 rounded">destination</code>) y se reportan al registrarse.
      </p>
      <p className="text-[11px] text-gray-400 mt-3">
        Requiere <strong>agent v0.2.0+</strong> — los agentes más antiguos no envían esta información todavía.
      </p>
    </div>
  )
}
