import { useState, useMemo } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  BarChart, Bar, Cell, XAxis, YAxis, Tooltip, ResponsiveContainer,
  LineChart, Line, CartesianGrid,
} from 'recharts'
import { format, parseISO } from 'date-fns'
import { fetchNodes, fetchJobs, updateNodeSourcePaths, type Node, type JobRun } from '@/lib/api'
import { Badge } from '@/components/Badge'
import { NodeConfigEditor } from '@/components/NodeConfigEditor'
import { fmtBytes, fmtDuration, fmtDate, timeAgo } from '@/lib/utils'

/**
 * Calcula estado visual de un nodo según su último heartbeat.
 * - online:  < 10 min
 * - stale:   10–30 min (agente pudo perder conexión)
 * - offline: > 30 min
 */
function computeNodeStatus(lastSeen: string | null | undefined): 'online' | 'stale' | 'offline' {
  if (!lastSeen) return 'offline'
  const minutes = (Date.now() - new Date(lastSeen).getTime()) / 60_000
  if (minutes < 10) return 'online'
  if (minutes < 30) return 'stale'
  return 'offline'
}

export function Nodes() {
  const qc = useQueryClient()
  const [drillNode, setDrillNode] = useState<Node | null>(null)
  const [configNode, setConfigNode] = useState<Node | null>(null)
  const { data = [], isLoading } = useQuery({
    queryKey: ['nodes'],
    queryFn: fetchNodes,
    refetchInterval: 15_000,
  })

  return (
    <div className="p-6 space-y-5">
      <div>
        <h1 className="text-xl font-bold text-gray-800">Nodos</h1>
        <p className="text-sm text-gray-400 mt-0.5">
          {data.length} nodo{data.length !== 1 ? 's' : ''} registrado{data.length !== 1 ? 's' : ''}
        </p>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {[...Array(2)].map((_, i) => (
            <div key={i} className="bg-white border border-gray-200 rounded-xl p-5 h-48 animate-pulse">
              <div className="h-4 bg-gray-100 rounded w-1/3 mb-3" />
              <div className="h-3 bg-gray-100 rounded w-1/2" />
            </div>
          ))}
        </div>
      ) : data.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {data.map(node => (
            <NodeCard
              key={node.id}
              node={node}
              onUpdate={() => qc.invalidateQueries({ queryKey: ['nodes'] })}
              onOpenRuns={() => setDrillNode(node)}
              onOpenConfig={() => setConfigNode(node)}
            />
          ))}
        </div>
      )}

      {drillNode && <NodeRunsDrawer node={drillNode} onClose={() => setDrillNode(null)} />}
      {configNode && <NodeConfigEditor node={configNode} onClose={() => setConfigNode(null)} />}
    </div>
  )
}

function NodeCard({ node, onUpdate, onOpenRuns, onOpenConfig }: {
  node: Node; onUpdate: () => void; onOpenRuns: () => void; onOpenConfig: () => void
}) {
  const [showAdd, setShowAdd] = useState(false)
  const [newPath, setNewPath] = useState('')
  const [saving, setSaving] = useState(false)

  const addPath = async () => {
    const p = newPath.trim()
    if (!p || node.source_paths.includes(p)) { setNewPath(''); setShowAdd(false); return }
    setSaving(true)
    try {
      await updateNodeSourcePaths(node.id, [...node.source_paths, p])
      setNewPath('')
      setShowAdd(false)
      onUpdate()
    } finally {
      setSaving(false)
    }
  }

  const removePath = async (path: string) => {
    await updateNodeSourcePaths(node.id, node.source_paths.filter(p => p !== path))
    onUpdate()
  }

  return (
    <div className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
      {/* Header */}
      <div className="flex items-start justify-between mb-4">
        <button
          onClick={onOpenRuns}
          className="flex items-center gap-3 text-left group/header"
          title="Ver backups recientes"
        >
          <div className="w-10 h-10 bg-orange-100 rounded-xl flex items-center justify-center flex-shrink-0">
            <svg className="w-5 h-5 text-orange-500" fill="none" viewBox="0 0 24 24"
              stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round"
                d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
            </svg>
          </div>
          <div>
            <p className="font-semibold text-gray-800 text-sm group-hover/header:text-primary transition-colors">
              {node.hostname}
            </p>
            <p className="text-xs text-gray-400 font-mono mt-0.5">{node.id}</p>
          </div>
        </button>
        <div className="flex items-center gap-2">
          <button
            onClick={onOpenConfig}
            title="Editar configuración remota"
            className="text-gray-400 hover:text-primary transition-colors p-1.5 rounded-lg hover:bg-gray-50"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </button>
          <Badge status={computeNodeStatus(node.last_seen)} />
        </div>
      </div>

      {/* Info row */}
      <div className="grid grid-cols-3 gap-2 mb-4">
        <InfoCell label="OS" value={node.os} />
        <InfoCell label="Versión" value={`v${node.agent_version}`} />
        <InfoCell label="Visto" value={timeAgo(node.last_seen)} />
      </div>

      {/* Source paths */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-widest">
            Carpetas respaldadas
          </p>
          <button
            onClick={() => setShowAdd(v => !v)}
            className="flex items-center gap-1 text-xs text-primary hover:text-[#3451d1] font-medium transition-colors"
          >
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24"
              stroke="currentColor" strokeWidth={2.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
            </svg>
            Agregar
          </button>
        </div>

        {node.source_paths.length === 0 && !showAdd ? (
          <p className="text-xs text-gray-400 italic py-1">
            Sin carpetas — el agente las sincroniza al conectarse.
          </p>
        ) : (
          <ul className="space-y-1">
            {node.source_paths.map(path => (
              <li key={path}
                className="group flex items-center gap-2 text-xs bg-gray-50 rounded-lg px-3 py-2">
                <svg className="w-3.5 h-3.5 text-gray-400 shrink-0" fill="none"
                  viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round"
                    d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V7z" />
                </svg>
                <span className="flex-1 font-mono text-gray-700 truncate">{path}</span>
                <button
                  onClick={() => removePath(path)}
                  title="Quitar"
                  className="opacity-0 group-hover:opacity-100 text-gray-400
                             hover:text-red-500 transition-all"
                >
                  <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24"
                    stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </li>
            ))}
          </ul>
        )}

        {showAdd && (
          <div className="flex gap-2 mt-2">
            <input
              type="text"
              value={newPath}
              onChange={e => setNewPath(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && addPath()}
              placeholder="C:\Users\Datos"
              autoFocus
              className="flex-1 text-xs bg-gray-50 border border-gray-200 rounded-lg px-3 py-2
                         font-mono outline-none focus:border-primary focus:ring-1 focus:ring-primary
                         transition-colors"
            />
            <button
              onClick={addPath}
              disabled={saving || !newPath.trim()}
              className="text-xs bg-primary hover:bg-[#3451d1] disabled:opacity-50 text-white
                         font-medium px-3 py-2 rounded-lg transition-colors"
            >
              {saving ? '...' : 'OK'}
            </button>
            <button
              onClick={() => { setShowAdd(false); setNewPath('') }}
              className="text-xs text-gray-400 hover:text-gray-600 px-2 transition-colors"
            >
              ✕
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

function InfoCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-center p-2 bg-gray-50 rounded-lg">
      <p className="text-[10px] text-gray-400 uppercase tracking-wide">{label}</p>
      <p className="text-xs font-semibold text-gray-700 mt-0.5 capitalize">{value}</p>
    </div>
  )
}

function NodeRunsDrawer({ node, onClose }: { node: Node; onClose: () => void }) {
  const { data: runs = [], isLoading } = useQuery({
    queryKey: ['node-runs', node.id],
    queryFn: () => fetchJobs({ node_id: node.id, limit: 20 }),
    refetchInterval: 10_000,
  })

  const stats = computeNodeStats(runs)
  const chartData = useMemo(() => buildRunChartData(runs), [runs])

  return (
    <>
      <div className="fixed inset-0 bg-black/30 backdrop-blur-[2px] z-40" onClick={onClose} />
      <div className="fixed inset-y-0 right-0 w-[480px] bg-white shadow-2xl z-50 flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-100">
          <div className="min-w-0">
            <p className="text-sm font-bold text-gray-800 truncate">{node.hostname}</p>
            <p className="text-xs font-mono text-gray-400 mt-0.5 truncate">{node.id}</p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors p-1 rounded-lg hover:bg-gray-100 flex-shrink-0"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-5 py-5 space-y-5">
          {/* Stats */}
          <div className="grid grid-cols-3 gap-2">
            <StatBox label="Tasa éxito" value={`${stats.successRate}%`}
              color={stats.successRate >= 90 ? 'green' : stats.successRate >= 70 ? 'amber' : 'red'} />
            <StatBox label="Runs" value={stats.total.toString()} color="blue" />
            <StatBox label="Total" value={fmtBytes(stats.totalBytes)} color="cyan" />
          </div>

          {/* Charts — bytes & duration trend */}
          {chartData.length >= 2 && (
            <div className="grid grid-cols-2 gap-3">
              <div className="bg-gray-50 rounded-xl p-3">
                <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-widest mb-2">
                  Volumen por run
                </p>
                <ResponsiveContainer width="100%" height={90}>
                  <BarChart data={chartData} margin={{ top: 2, right: 2, left: -28, bottom: 0 }}>
                    <XAxis dataKey="label" tick={{ fontSize: 9, fill: '#94a3b8' }}
                      axisLine={false} tickLine={false} interval="preserveStartEnd" />
                    <YAxis tick={{ fontSize: 9, fill: '#94a3b8' }} axisLine={false} tickLine={false}
                      tickFormatter={(v: number) => fmtBytes(v)} width={60} />
                    <Tooltip
                      cursor={{ fill: 'rgba(67,97,238,0.08)' }}
                      contentStyle={{ fontSize: 11, borderRadius: 8, border: '1px solid #e2e8f0' }}
                      formatter={(v: number) => [fmtBytes(v), 'Datos']}
                      labelFormatter={(l) => l}
                    />
                    <Bar dataKey="bytes" radius={[3, 3, 0, 0]}>
                      {chartData.map((d, i) => (
                        <Cell key={i} fill={
                          d.status === 'failed'  ? '#ef4444' :
                          d.status === 'warning' ? '#f59e0b' :
                                                   '#4361ee'
                        } />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </div>

              <div className="bg-gray-50 rounded-xl p-3">
                <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-widest mb-2">
                  Duración (min)
                </p>
                <ResponsiveContainer width="100%" height={90}>
                  <LineChart data={chartData} margin={{ top: 2, right: 2, left: -28, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#eef2f7" vertical={false} />
                    <XAxis dataKey="label" tick={{ fontSize: 9, fill: '#94a3b8' }}
                      axisLine={false} tickLine={false} interval="preserveStartEnd" />
                    <YAxis tick={{ fontSize: 9, fill: '#94a3b8' }} axisLine={false} tickLine={false}
                      tickFormatter={(v: number) => `${v}m`} width={32} />
                    <Tooltip
                      contentStyle={{ fontSize: 11, borderRadius: 8, border: '1px solid #e2e8f0' }}
                      formatter={(v: number) => [`${v} min`, 'Duración']}
                    />
                    <Line type="monotone" dataKey="minutes" stroke="#22c55e" strokeWidth={2}
                      dot={{ r: 2, fill: '#22c55e' }} activeDot={{ r: 4 }} />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            </div>
          )}

          {/* Runs list */}
          <div>
            <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-widest mb-2">
              Últimas ejecuciones
            </p>
            {isLoading ? (
              <div className="space-y-2">
                {[...Array(5)].map((_, i) => (
                  <div key={i} className="h-12 bg-gray-100 rounded-lg animate-pulse" />
                ))}
              </div>
            ) : runs.length === 0 ? (
              <p className="text-xs text-gray-400 italic py-4 text-center">
                Este nodo aún no ha ejecutado ningún backup.
              </p>
            ) : (
              <ul className="space-y-1.5">
                {runs.map(r => (
                  <li key={r.id} className="bg-gray-50 rounded-lg px-3 py-2 flex items-center gap-3">
                    <StatusDot status={r.status} />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-medium text-gray-700">
                          {fmtDate(r.started_at)}
                        </span>
                        <span className="text-[10px] text-gray-400 capitalize">{r.backup_type}</span>
                      </div>
                      <div className="flex items-center justify-between mt-0.5">
                        <span className="text-[10px] text-gray-500">
                          {r.files_done.toLocaleString()} archivos · {fmtBytes(r.bytes_done)}
                        </span>
                        <span className="text-[10px] text-gray-400">
                          {fmtDuration(r.duration_seconds)}
                        </span>
                      </div>
                      {r.error_message && (
                        <p className="text-[10px] text-red-500 mt-1 truncate font-mono">
                          {r.error_message}
                        </p>
                      )}
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </div>
    </>
  )
}

/**
 * Transform the runs list into chronological chart rows.
 * API returns newest first — reverse so the chart reads left→right in time.
 */
function buildRunChartData(runs: JobRun[]) {
  return [...runs].reverse().map(r => ({
    label: r.started_at ? format(parseISO(r.started_at), 'd/M HH:mm') : '—',
    bytes: r.bytes_done || 0,
    minutes: r.duration_seconds ? Math.round((r.duration_seconds / 60) * 10) / 10 : 0,
    status: r.status,
  }))
}

function computeNodeStats(runs: JobRun[]) {
  const total = runs.length
  const completed = runs.filter(r => r.status === 'completed').length
  const successRate = total === 0 ? 0 : Math.round((completed / total) * 100)
  const totalBytes = runs.reduce((acc, r) => acc + (r.bytes_done || 0), 0)
  return { total, successRate, totalBytes }
}

function StatBox({ label, value, color }: { label: string; value: string; color: 'green' | 'amber' | 'red' | 'blue' | 'cyan' }) {
  const cls = {
    green: 'bg-green-50 text-green-700',
    amber: 'bg-amber-50 text-amber-700',
    red:   'bg-red-50 text-red-600',
    blue:  'bg-blue-50 text-blue-700',
    cyan:  'bg-cyan-50 text-cyan-700',
  }[color]
  return (
    <div className={`rounded-xl p-3 text-center ${cls}`}>
      <p className="text-base font-bold">{value}</p>
      <p className="text-[10px] opacity-70 mt-0.5">{label}</p>
    </div>
  )
}

function StatusDot({ status }: { status: string }) {
  const cls =
    status === 'completed' ? 'bg-green-500' :
    status === 'failed'    ? 'bg-red-500'   :
    status === 'warning'   ? 'bg-amber-500' :
    status === 'running'   ? 'bg-blue-500 animate-pulse' :
                             'bg-gray-300'
  return <span className={`w-2 h-2 rounded-full flex-shrink-0 ${cls}`} />
}

function EmptyState() {
  return (
    <div className="bg-white border border-gray-200 rounded-xl p-12 text-center">
      <div className="w-12 h-12 bg-gray-50 rounded-xl flex items-center justify-center mx-auto mb-3">
        <svg className="w-6 h-6 text-gray-300" fill="none" viewBox="0 0 24 24"
          stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round"
            d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
        </svg>
      </div>
      <p className="text-sm text-gray-500 font-medium">Sin nodos registrados</p>
      <p className="text-xs text-gray-400 mt-1">
        El agente se registra automáticamente al arrancar el servicio.
      </p>
    </div>
  )
}
