import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchJobs, type JobRun } from '@/lib/api'
import { Badge } from '@/components/Badge'
import { fmtBytes, fmtDuration, fmtDate, fmtShortId } from '@/lib/utils'

const STATUS_OPTIONS = [
  { value: '',           label: 'Todos'        },
  { value: 'running',   label: 'En ejecución' },
  { value: 'completed', label: 'Completados'  },
  { value: 'failed',    label: 'Fallidos'     },
  { value: 'warning',   label: 'Advertencia'  },
]

export function Jobs() {
  const [status, setStatus] = useState('')
  const [selected, setSelected] = useState<JobRun | null>(null)

  const { data = [], isLoading } = useQuery({
    queryKey: ['jobs', status],
    queryFn: () => fetchJobs({ status: status || undefined, limit: 100 }),
    refetchInterval: 10_000,
  })

  return (
    <div className="p-6 space-y-5">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-xl font-bold text-gray-800">Jobs</h1>
          <p className="text-sm text-gray-400 mt-0.5">{data.length} ejecuciones</p>
        </div>
      </div>

      <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
        {/* Filters */}
        <div className="px-5 py-3.5 border-b border-gray-100 flex items-center gap-2">
          {STATUS_OPTIONS.map(opt => (
            <button
              key={opt.value}
              onClick={() => setStatus(opt.value)}
              className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors ${
                status === opt.value
                  ? 'bg-primary text-white'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>

        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-[11px] text-gray-400 uppercase tracking-wide">
            <tr>
              <th className="text-left px-5 py-3 font-medium">Job ID</th>
              <th className="text-left px-5 py-3 font-medium">Nodo</th>
              <th className="text-left px-5 py-3 font-medium">Tipo</th>
              <th className="text-left px-5 py-3 font-medium">Estado</th>
              <th className="text-right px-5 py-3 font-medium">Progreso</th>
              <th className="text-right px-5 py-3 font-medium">Archivos</th>
              <th className="text-right px-5 py-3 font-medium">Datos</th>
              <th className="text-right px-5 py-3 font-medium">Duración</th>
              <th className="text-left px-5 py-3 font-medium">Inicio</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {isLoading ? (
              [...Array(5)].map((_, i) => (
                <tr key={i}>
                  {[...Array(9)].map((_, j) => (
                    <td key={j} className="px-5 py-3">
                      <div className="h-3 bg-gray-100 rounded animate-pulse w-3/4" />
                    </td>
                  ))}
                </tr>
              ))
            ) : data.length === 0 ? (
              <tr>
                <td colSpan={9} className="px-5 py-12 text-center text-gray-400 text-sm">
                  No hay jobs con este filtro.
                </td>
              </tr>
            ) : data.map(run => (
              <tr
                key={run.id}
                onClick={() => setSelected(run)}
                className="hover:bg-gray-50 transition-colors cursor-pointer"
              >
                <td className="px-5 py-3 font-mono text-xs text-gray-600">{fmtShortId(run.job_id)}</td>
                <td className="px-5 py-3 text-xs text-gray-700">{run.node_id ?? '—'}</td>
                <td className="px-5 py-3 text-xs text-gray-500 capitalize">{run.backup_type}</td>
                <td className="px-5 py-3"><Badge status={run.status} /></td>
                <td className="px-5 py-3">
                  <div className="flex items-center gap-2 justify-end">
                    <div className="w-16 bg-gray-100 rounded-full h-1.5 overflow-hidden">
                      <div
                        className={`h-full rounded-full ${run.status === 'completed' ? 'bg-green-500' : 'bg-primary'}`}
                        style={{ width: `${Math.min(run.progress_pct, 100)}%` }}
                      />
                    </div>
                    <span className="text-[10px] text-gray-400 w-8 text-right">{run.progress_pct}%</span>
                  </div>
                </td>
                <td className="px-5 py-3 text-right text-xs text-gray-600">
                  {run.files_done.toLocaleString()} / {run.files_total.toLocaleString()}
                </td>
                <td className="px-5 py-3 text-right text-xs text-gray-600">{fmtBytes(run.bytes_done)}</td>
                <td className="px-5 py-3 text-right text-xs text-gray-500">{fmtDuration(run.duration_seconds)}</td>
                <td className="px-5 py-3 text-xs text-gray-400">{fmtDate(run.started_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {selected && <JobDetailModal run={selected} onClose={() => setSelected(null)} />}
    </div>
  )
}

// ── Job Detail Modal ──────────────────────────────────────────────────────────

function JobDetailModal({ run, onClose }: { run: JobRun; onClose: () => void }) {
  return (
    <>
      {/* backdrop */}
      <div
        className="fixed inset-0 bg-black/30 backdrop-blur-[2px] z-40"
        onClick={onClose}
      />
      {/* panel */}
      <div className="fixed inset-y-0 right-0 w-[420px] bg-white shadow-2xl z-50 flex flex-col overflow-hidden">
        {/* header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-100">
          <div>
            <p className="text-sm font-bold text-gray-800">Detalle del job</p>
            <p className="text-xs font-mono text-gray-400 mt-0.5">{run.job_id}</p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors p-1 rounded-lg hover:bg-gray-100"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-5 py-5 space-y-5">
          {/* Status + progress */}
          <div className="bg-gray-50 rounded-xl p-4">
            <div className="flex items-center justify-between mb-3">
              <Badge status={run.status} />
              <span className="text-sm font-bold text-gray-700">{run.progress_pct}%</span>
            </div>
            <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${
                  run.status === 'completed' ? 'bg-green-500' :
                  run.status === 'failed'    ? 'bg-red-500'   : 'bg-primary'
                }`}
                style={{ width: `${Math.min(run.progress_pct, 100)}%` }}
              />
            </div>
            {run.current_file && (
              <p className="text-[10px] text-gray-400 mt-2 truncate font-mono">{run.current_file}</p>
            )}
          </div>

          {/* Key data */}
          <div>
            <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-widest mb-2">Información</p>
            <div className="space-y-0 divide-y divide-gray-50">
              {[
                ['Nodo',         run.node_id ?? '—'],
                ['Tipo',         run.backup_type || '—'],
                ['Inicio',       fmtDate(run.started_at)],
                ['Fin',          run.finished_at ? fmtDate(run.finished_at) : '—'],
                ['Duración',     run.duration_seconds ? fmtDuration(run.duration_seconds) : '—'],
              ].map(([label, value]) => (
                <div key={label} className="flex justify-between py-2.5">
                  <span className="text-xs text-gray-500">{label}</span>
                  <span className="text-xs font-medium text-gray-700">{value}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Volume */}
          <div>
            <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-widest mb-2">Volumen</p>
            <div className="grid grid-cols-2 gap-3">
              <div className="bg-blue-50 rounded-xl p-3 text-center">
                <p className="text-lg font-bold text-blue-600">{run.files_done.toLocaleString()}</p>
                <p className="text-[10px] text-blue-400 mt-0.5">archivos copiados</p>
              </div>
              <div className="bg-cyan-50 rounded-xl p-3 text-center">
                <p className="text-lg font-bold text-cyan-600">{fmtBytes(run.bytes_done)}</p>
                <p className="text-[10px] text-cyan-400 mt-0.5">datos copiados</p>
              </div>
              <div className="bg-gray-50 rounded-xl p-3 text-center">
                <p className="text-base font-bold text-gray-600">{run.files_total.toLocaleString()}</p>
                <p className="text-[10px] text-gray-400 mt-0.5">archivos totales</p>
              </div>
              <div className="bg-gray-50 rounded-xl p-3 text-center">
                <p className="text-base font-bold text-gray-600">{fmtBytes(run.bytes_total)}</p>
                <p className="text-[10px] text-gray-400 mt-0.5">bytes totales</p>
              </div>
            </div>
          </div>

          {/* Error message if any */}
          {run.error_message && (
            <div>
              <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-widest mb-2">Error</p>
              <div className="bg-red-50 border border-red-100 rounded-xl p-3">
                <p className="text-xs text-red-600 font-mono break-words">{run.error_message}</p>
              </div>
            </div>
          )}

          {/* Job ID full */}
          <div>
            <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-widest mb-2">Job ID</p>
            <p className="text-xs font-mono text-gray-600 bg-gray-50 rounded-lg px-3 py-2 break-all">
              {run.job_id}
            </p>
          </div>
        </div>
      </div>
    </>
  )
}
