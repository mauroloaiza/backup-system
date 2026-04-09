import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  AreaChart, Area, BarChart, Bar,
  XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, Legend,
} from 'recharts'
import { format, parseISO } from 'date-fns'
import { es } from 'date-fns/locale'
import { fetchHistory } from '@/lib/api'
import { StatCard } from '@/components/StatCard'
import { fmtBytes, fmtDuration } from '@/lib/utils'

const RANGE_OPTIONS = [
  { label: '7 días',  value: 7  },
  { label: '14 días', value: 14 },
  { label: '30 días', value: 30 },
  { label: '90 días', value: 90 },
]

function shortDate(dateStr: string, days: number) {
  const d = parseISO(dateStr)
  if (days <= 14) return format(d, 'd MMM', { locale: es })
  if (days <= 30) return format(d, 'd MMM', { locale: es })
  return format(d, 'MMM', { locale: es })
}

export function History() {
  const [days, setDays] = useState(30)

  const { data, isLoading, error } = useQuery({
    queryKey: ['history', days],
    queryFn: () => fetchHistory(days),
    refetchInterval: 60_000,
  })

  if (isLoading) return <LoadingSkeleton />
  if (error || !data) return <ErrorState />

  const chartPoints: ChartPoint[] = data.points.map(p => ({
    ...p,
    label: shortDate(p.date, days),
    bytesFmt: fmtBytes(p.bytes),
  }))

  const displayPoints = days <= 30 ? chartPoints : aggregateByWeek(chartPoints)

  return (
    <div className="p-6 space-y-5">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-xl font-bold text-gray-800">Historial</h1>
          <p className="text-sm text-gray-400 mt-0.5">Tendencias y estadísticas</p>
        </div>
        <div className="flex items-center gap-1 bg-gray-100 rounded-lg p-1">
          {RANGE_OPTIONS.map(opt => (
            <button
              key={opt.value}
              onClick={() => setDays(opt.value)}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
                days === opt.value
                  ? 'bg-white text-gray-800 shadow-sm'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-4 gap-4">
        <StatCard
          label="Total ejecuciones"
          value={data.total_runs.toLocaleString()}
          iconBg="bg-blue-100"
          icon={<svg className="w-5 h-5 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
          </svg>}
        />
        <StatCard
          label="Tasa de éxito"
          value={`${data.success_rate}%`}
          sub={data.success_rate >= 95 ? 'Excelente' : data.success_rate >= 80 ? 'Normal' : 'Revisar'}
          subColor={data.success_rate >= 95 ? 'text-green-500' : data.success_rate >= 80 ? 'text-amber-500' : 'text-red-500'}
          iconBg="bg-green-100"
          icon={<svg className="w-5 h-5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>}
        />
        <StatCard
          label="Total respaldado"
          value={fmtBytes(data.total_bytes)}
          iconBg="bg-cyan-100"
          icon={<svg className="w-5 h-5 text-cyan-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
          </svg>}
        />
        <StatCard
          label="Duración promedio"
          value={fmtDuration(data.avg_duration)}
          iconBg="bg-violet-100"
          icon={<svg className="w-5 h-5 text-violet-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>}
        />
      </div>

      {/* Charts row */}
      <div className="grid grid-cols-2 gap-4">
        {/* Jobs per day — area */}
        <div className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
          <h3 className="text-sm font-semibold text-gray-700 mb-1">Ejecuciones por día</h3>
          <p className="text-xs text-gray-400 mb-4">Exitosas vs fallidas</p>
          <ResponsiveContainer width="100%" height={180}>
            <AreaChart data={displayPoints} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
              <defs>
                <linearGradient id="gradComp" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%"  stopColor="#22c55e" stopOpacity={0.15} />
                  <stop offset="95%" stopColor="#22c55e" stopOpacity={0}    />
                </linearGradient>
                <linearGradient id="gradFail" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%"  stopColor="#ef4444" stopOpacity={0.15} />
                  <stop offset="95%" stopColor="#ef4444" stopOpacity={0}    />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" vertical={false} />
              <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#94a3b8' }} axisLine={false} tickLine={false} interval="preserveStartEnd" />
              <YAxis tick={{ fontSize: 10, fill: '#94a3b8' }} axisLine={false} tickLine={false} allowDecimals={false} />
              <Tooltip
                contentStyle={{ fontSize: 12, borderRadius: 8, border: '1px solid #e2e8f0' }}
                labelStyle={{ fontWeight: 600, color: '#374151' }}
              />
              <Legend wrapperStyle={{ fontSize: 11, paddingTop: 8 }} />
              <Area type="monotone" dataKey="completed" name="Exitosos" stroke="#22c55e" strokeWidth={2}
                fill="url(#gradComp)" dot={false} activeDot={{ r: 4 }} />
              <Area type="monotone" dataKey="failed" name="Fallidos" stroke="#ef4444" strokeWidth={2}
                fill="url(#gradFail)" dot={false} activeDot={{ r: 4 }} />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Bytes per day — bar */}
        <div className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
          <h3 className="text-sm font-semibold text-gray-700 mb-1">Volumen respaldado</h3>
          <p className="text-xs text-gray-400 mb-4">Bytes procesados por período</p>
          <ResponsiveContainer width="100%" height={180}>
            <BarChart data={displayPoints} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" vertical={false} />
              <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#94a3b8' }} axisLine={false} tickLine={false} interval="preserveStartEnd" />
              <YAxis
                tick={{ fontSize: 10, fill: '#94a3b8' }}
                axisLine={false}
                tickLine={false}
                tickFormatter={v => fmtBytes(v)}
              />
              <Tooltip
                contentStyle={{ fontSize: 12, borderRadius: 8, border: '1px solid #e2e8f0' }}
                labelStyle={{ fontWeight: 600, color: '#374151' }}
                formatter={(v: number) => [fmtBytes(v), 'Datos']}
              />
              <Bar dataKey="bytes" name="Datos" fill="#4361ee" radius={[3, 3, 0, 0]} maxBarSize={24} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Duration trend */}
      <div className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
        <h3 className="text-sm font-semibold text-gray-700 mb-1">Duración promedio por día</h3>
        <p className="text-xs text-gray-400 mb-4">Segundos por ejecución</p>
        <ResponsiveContainer width="100%" height={120}>
          <AreaChart data={displayPoints} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
            <defs>
              <linearGradient id="gradDur" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%"  stopColor="#a855f7" stopOpacity={0.12} />
                <stop offset="95%" stopColor="#a855f7" stopOpacity={0}    />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" vertical={false} />
            <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#94a3b8' }} axisLine={false} tickLine={false} interval="preserveStartEnd" />
            <YAxis
              tick={{ fontSize: 10, fill: '#94a3b8' }}
              axisLine={false}
              tickLine={false}
              tickFormatter={v => fmtDuration(v)}
            />
            <Tooltip
              contentStyle={{ fontSize: 12, borderRadius: 8, border: '1px solid #e2e8f0' }}
              formatter={(v: number) => [fmtDuration(v), 'Duración']}
            />
            <Area type="monotone" dataKey="duration_avg" name="Duración" stroke="#a855f7" strokeWidth={2}
              fill="url(#gradDur)" dot={false} activeDot={{ r: 4 }} />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      {/* Zero-data notice */}
      {data.total_runs === 0 && (
        <div className="bg-blue-50 border border-blue-100 rounded-xl p-4 text-center">
          <p className="text-sm text-blue-600 font-medium">Sin datos en este período</p>
          <p className="text-xs text-blue-400 mt-1">Ejecuta el agente para ver estadísticas históricas.</p>
        </div>
      )}
    </div>
  )
}

// ── helpers ───────────────────────────────────────────────────────────────────

interface ChartPoint {
  date: string
  label: string
  total: number
  completed: number
  failed: number
  bytes: number
  duration_avg: number
  bytesFmt: string
}

function aggregateByWeek(points: ChartPoint[]): ChartPoint[] {
  const weeks: Record<string, ChartPoint> = {}
  points.forEach(p => {
    const d = parseISO(p.date)
    const day = d.getDay()
    const diff = day === 0 ? -6 : 1 - day
    const mon = new Date(d)
    mon.setDate(d.getDate() + diff)
    const key = format(mon, 'yyyy-MM-dd')
    if (!weeks[key]) {
      weeks[key] = { ...p, label: format(mon, 'd MMM', { locale: es }) }
    } else {
      weeks[key].total += p.total
      weeks[key].completed += p.completed
      weeks[key].failed += p.failed
      weeks[key].bytes += p.bytes
      const count = weeks[key].total
      weeks[key].duration_avg = count > 0
        ? (weeks[key].duration_avg * (count - 1) + p.duration_avg) / count
        : p.duration_avg
    }
  })
  return Object.values(weeks)
}

function LoadingSkeleton() {
  return (
    <div className="p-6 space-y-5">
      <div className="h-7 bg-gray-100 rounded w-32 animate-pulse" />
      <div className="grid grid-cols-4 gap-4">
        {[...Array(4)].map((_, i) => (
          <div key={i} className="bg-white border border-gray-200 rounded-xl p-4 h-20 animate-pulse">
            <div className="h-3 bg-gray-100 rounded w-1/2 mb-2" />
            <div className="h-6 bg-gray-100 rounded w-1/3" />
          </div>
        ))}
      </div>
      <div className="grid grid-cols-2 gap-4">
        {[...Array(2)].map((_, i) => (
          <div key={i} className="bg-white border border-gray-200 rounded-xl p-5 h-56 animate-pulse">
            <div className="h-4 bg-gray-100 rounded w-1/3 mb-3" />
            <div className="h-full bg-gray-50 rounded" />
          </div>
        ))}
      </div>
    </div>
  )
}

function ErrorState() {
  return (
    <div className="p-6 flex items-center justify-center min-h-96">
      <div className="text-center">
        <div className="w-12 h-12 bg-red-50 rounded-xl flex items-center justify-center mx-auto mb-3">
          <svg className="w-6 h-6 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <p className="text-sm font-medium text-gray-700">No se pueden cargar las estadísticas</p>
        <p className="text-xs text-gray-400 mt-1">Verifica que el servidor esté disponible</p>
      </div>
    </div>
  )
}
