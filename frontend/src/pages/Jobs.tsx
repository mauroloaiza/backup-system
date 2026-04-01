import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchJobs } from '@/lib/api'
import { Badge } from '@/components/Badge'
import { fmtBytes, fmtDuration, fmtDate, fmtShortId } from '@/lib/utils'

const STATUS_OPTIONS = [
  { value: '', label: 'Todos' },
  { value: 'running', label: 'En ejecución' },
  { value: 'completed', label: 'Completados' },
  { value: 'failed', label: 'Fallidos' },
  { value: 'warning', label: 'Advertencia' },
]

export function Jobs() {
  const [status, setStatus] = useState('')

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
              <tr key={run.id} className="hover:bg-gray-50 transition-colors">
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
    </div>
  )
}
