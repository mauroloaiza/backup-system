import { useQuery } from '@tanstack/react-query'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { format, subDays } from 'date-fns'
import { fetchDashboard, type JobRun } from '@/lib/api'
import { StatCard } from '@/components/StatCard'
import { Badge } from '@/components/Badge'
import { fmtBytes, fmtDuration, fmtDate, fmtShortId } from '@/lib/utils'

// Generate placeholder chart data from actual run history
function buildChartData(runs: JobRun[]) {
  return Array.from({ length: 14 }, (_, i) => {
    const d = subDays(new Date(), 13 - i)
    const day = format(d, 'yyyy-MM-dd')
    const dayRuns = runs.filter(r => r.started_at.startsWith(day))
    return {
      date: format(d, 'd MMM', { locale: undefined }),
      total: dayRuns.length,
      exitosos: dayRuns.filter(r => r.status === 'completed').length,
      fallidos: dayRuns.filter(r => r.status === 'failed').length,
    }
  })
}

export function Dashboard() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['dashboard'],
    queryFn: fetchDashboard,
    refetchInterval: 10_000,
  })

  if (isLoading) return <Loading />
  if (error || !data) return <Error />

  const chartData = buildChartData(data.recent_runs)

  return (
    <div className="p-6 space-y-5">
      {/* Page title */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-xl font-bold text-gray-800">Dashboard</h1>
          <p className="text-sm text-gray-400 mt-0.5">Resumen general del sistema</p>
        </div>
        <button className="flex items-center gap-1.5 text-sm text-gray-500 border border-gray-200
          rounded-lg px-3 py-1.5 bg-white hover:bg-gray-50 transition-colors">
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round"
              d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
          </svg>
          Personalizar
        </button>
      </div>

      {/* Stat cards row 1 */}
      <div className="grid grid-cols-4 gap-4">
        <StatCard label="Total Backups" value={data.total_runs.toLocaleString()}
          iconBg="bg-blue-100"
          icon={<svg className="w-5 h-5 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
          </svg>} />
        <StatCard label="En ejecución" value={data.runs_running}
          sub={data.runs_running > 0 ? 'Activos ahora' : undefined}
          subColor="text-amber-500"
          iconBg="bg-amber-100"
          icon={<svg className="w-5 h-5 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>} />
        <StatCard label="Exitosos" value={data.runs_completed.toLocaleString()}
          iconBg="bg-green-100"
          icon={<svg className="w-5 h-5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>} />
        <StatCard label="Almacenado hoy" value={fmtBytes(data.bytes_backed_up_today)}
          sub="últimas 24h"
          iconBg="bg-cyan-100"
          icon={<svg className="w-5 h-5 text-cyan-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
          </svg>} />
      </div>

      {/* Stat cards row 2 */}
      <div className="grid grid-cols-4 gap-4">
        <StatCard label="Fallidos activos" value={data.runs_failed}
          sub={data.runs_failed > 0 ? 'Alta prioridad' : undefined}
          subColor="text-red-500"
          iconBg="bg-red-100"
          icon={<svg className="w-5 h-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>} />
        <StatCard label="Nodos activos" value={data.nodes_online}
          sub={`${data.total_nodes} registrados`}
          iconBg="bg-orange-100"
          icon={<svg className="w-5 h-5 text-orange-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
          </svg>} />
        <StatCard label="Jobs hoy" value={data.runs_today}
          iconBg="bg-violet-100"
          icon={<svg className="w-5 h-5 text-violet-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>} />
        <StatCard label="Total almacenado" value={fmtBytes(data.bytes_backed_up_total)}
          iconBg="bg-indigo-100"
          icon={<svg className="w-5 h-5 text-indigo-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
          </svg>} />
      </div>

      {/* Chart + Running now */}
      <div className="grid grid-cols-3 gap-4">
        {/* Area chart */}
        <div className="col-span-2 bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
          <h3 className="text-sm font-semibold text-gray-700 mb-4">Tendencia de backups — últimos 14 días</h3>
          <ResponsiveContainer width="100%" height={160}>
            <AreaChart data={chartData} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
              <defs>
                <linearGradient id="gradTotal" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#4361ee" stopOpacity={0.12} />
                  <stop offset="95%" stopColor="#4361ee" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" vertical={false} />
              <XAxis dataKey="date" tick={{ fontSize: 10, fill: '#94a3b8' }} axisLine={false} tickLine={false} />
              <YAxis tick={{ fontSize: 10, fill: '#94a3b8' }} axisLine={false} tickLine={false} allowDecimals={false} />
              <Tooltip
                contentStyle={{ fontSize: 12, borderRadius: 8, border: '1px solid #e2e8f0', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
                labelStyle={{ fontWeight: 600, color: '#374151' }}
              />
              <Area type="monotone" dataKey="total" name="Total" stroke="#4361ee" strokeWidth={2.5}
                fill="url(#gradTotal)" dot={false} activeDot={{ r: 4, fill: '#4361ee' }} />
              <Area type="monotone" dataKey="exitosos" name="Exitosos" stroke="#22c55e" strokeWidth={1.5}
                fill="none" dot={false} strokeDasharray="4 2" />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Running now */}
        <div className="bg-white border border-gray-200 rounded-xl p-4 shadow-sm flex flex-col">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">En ejecución ahora</h3>
          <div className="space-y-3 flex-1">
            {data.running_now.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full py-6 text-center">
                <svg className="w-8 h-8 text-gray-200 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <p className="text-xs text-gray-400">Sin jobs activos</p>
              </div>
            ) : data.running_now.map(run => (
              <div key={run.job_id} className="bg-gray-50 border border-gray-100 rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs font-semibold text-gray-700 font-mono">{fmtShortId(run.job_id)}</span>
                  <span className="inline-flex items-center gap-1 text-xs font-medium text-primary">
                    <span className="w-1.5 h-1.5 rounded-full bg-primary pulse inline-block" />Activo
                  </span>
                </div>
                <div className="h-1.5 bg-gray-200 rounded-full overflow-hidden mb-1.5">
                  <div className="h-full bg-primary rounded-full bar-fill" style={{ width: `${run.progress_pct}%` }} />
                </div>
                <div className="flex justify-between text-[10px] text-gray-400">
                  <span>{run.progress_pct}%</span>
                  <span>{run.node_id ?? '—'}</span>
                </div>
                {run.current_file && (
                  <p className="text-[10px] text-gray-400 mt-1 truncate">{run.current_file}</p>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Recent runs table */}
      <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
        <div className="px-5 py-3.5 border-b border-gray-100 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-gray-700">Últimas ejecuciones</h3>
          <a href="/jobs" className="text-xs text-primary hover:underline">Ver todas →</a>
        </div>
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-[11px] text-gray-400 uppercase tracking-wide">
            <tr>
              <th className="text-left px-5 py-3 font-medium">Job ID</th>
              <th className="text-left px-5 py-3 font-medium">Nodo</th>
              <th className="text-left px-5 py-3 font-medium">Tipo</th>
              <th className="text-left px-5 py-3 font-medium">Estado</th>
              <th className="text-right px-5 py-3 font-medium">Archivos</th>
              <th className="text-right px-5 py-3 font-medium">Datos</th>
              <th className="text-right px-5 py-3 font-medium">Duración</th>
              <th className="text-left px-5 py-3 font-medium">Inicio</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {data.recent_runs.length === 0 ? (
              <tr>
                <td colSpan={8} className="px-5 py-10 text-center text-gray-400 text-sm">
                  Sin ejecuciones aún. Corre el agente para ver datos aquí.
                </td>
              </tr>
            ) : data.recent_runs.map(run => (
              <tr key={run.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-5 py-3 font-mono text-xs text-gray-600">{fmtShortId(run.job_id)}</td>
                <td className="px-5 py-3 text-gray-700 text-xs">{run.node_id ?? '—'}</td>
                <td className="px-5 py-3 text-gray-500 text-xs capitalize">{run.backup_type}</td>
                <td className="px-5 py-3"><Badge status={run.status} /></td>
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

function Loading() {
  return (
    <div className="p-6">
      <div className="grid grid-cols-4 gap-4 mb-5">
        {[...Array(8)].map((_, i) => (
          <div key={i} className="bg-white border border-gray-200 rounded-xl p-4 h-20 animate-pulse">
            <div className="h-3 bg-gray-100 rounded w-1/2 mb-2" />
            <div className="h-6 bg-gray-100 rounded w-1/3" />
          </div>
        ))}
      </div>
    </div>
  )
}

function Error() {
  return (
    <div className="p-6 flex items-center justify-center min-h-96">
      <div className="text-center">
        <div className="w-12 h-12 bg-red-50 rounded-xl flex items-center justify-center mx-auto mb-3">
          <svg className="w-6 h-6 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <p className="text-sm font-medium text-gray-700">No se puede conectar al servidor</p>
        <p className="text-xs text-gray-400 mt-1">Verifica que el servidor FastAPI esté corriendo en localhost:8000</p>
      </div>
    </div>
  )
}
