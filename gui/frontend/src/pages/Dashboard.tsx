import { useEffect, useState, useCallback } from 'react'
import {
  Play, Square, RefreshCw, CheckCircle2, XCircle,
  Clock, HardDrive, Files, Zap,
} from 'lucide-react'
import { GetServiceStatus, GetStats, StartService, StopService, RunBackupNow } from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'
import { EventsOn } from '../../wailsjs/runtime/runtime'

type SvcStatus = main.ServiceStatus
type Stats     = main.BackupStats

function fmtBytes(b: number): string {
  if (b === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(b) / Math.log(1024))
  return (b / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}

export default function Dashboard() {
  const [svc, setSvc]       = useState<SvcStatus | null>(null)
  const [stats, setStats]   = useState<Stats | null>(null)
  const [loading, setLoading] = useState(false)
  const [backupMsg, setBackupMsg] = useState('')
  const [running, setRunning] = useState(false)

  const refresh = useCallback(async () => {
    setLoading(true)
    const [s, st] = await Promise.all([GetServiceStatus(), GetStats()])
    setSvc(s)
    setStats(st)
    setLoading(false)
  }, [])

  useEffect(() => {
    refresh()
    const id = setInterval(refresh, 10_000)
    EventsOn('backup:done', (msg: string) => {
      setRunning(false)
      setBackupMsg(msg.includes('Error') ? '❌ ' + msg.split('\n')[0] : '✅ Backup completed')
      refresh()
      setTimeout(() => setBackupMsg(''), 6000)
    })
    return () => clearInterval(id)
  }, [refresh])

  const handleStart = async () => { await StartService(); refresh() }
  const handleStop  = async () => { await StopService();  refresh() }
  const handleBackup = async () => {
    setRunning(true)
    setBackupMsg('Running backup...')
    await RunBackupNow()
  }

  const isRunning = svc?.running ?? false

  return (
    <div className="p-8 space-y-8 max-w-5xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-white">Dashboard</h1>
          <p className="text-sm text-white/40 mt-0.5">Agent status &amp; last backup overview</p>
        </div>
        <button
          onClick={refresh}
          className="flex items-center gap-2 px-3 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-white/60 hover:text-white text-sm transition-all"
        >
          <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>

      {/* Service status card */}
      <div className="rounded-xl border border-white/5 bg-navy-700 p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className={`w-3 h-3 rounded-full ${isRunning ? 'bg-success animate-pulse' : 'bg-white/20'}`} />
            <div>
              <div className="text-base font-medium text-white">Windows Service</div>
              <div className="text-sm text-white/40 mt-0.5">
                {svc ? svc.status : 'Checking...'}
                {svc && !svc.installed && ' — not installed'}
              </div>
            </div>
          </div>
          <div className="flex gap-2">
            <button
              onClick={handleBackup}
              disabled={running}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary hover:bg-primary-hover text-white text-sm font-medium transition-all disabled:opacity-50"
            >
              <Zap size={14} />
              {running ? 'Running...' : 'Backup Now'}
            </button>
            {isRunning ? (
              <button
                onClick={handleStop}
                className="flex items-center gap-2 px-4 py-2 rounded-lg bg-danger/20 hover:bg-danger/30 text-danger text-sm font-medium transition-all"
              >
                <Square size={14} />
                Stop
              </button>
            ) : (
              <button
                onClick={handleStart}
                className="flex items-center gap-2 px-4 py-2 rounded-lg bg-success/20 hover:bg-success/30 text-success text-sm font-medium transition-all"
              >
                <Play size={14} />
                Start
              </button>
            )}
          </div>
        </div>
        {backupMsg && (
          <div className="mt-4 px-4 py-2.5 rounded-lg bg-white/5 text-sm text-white/70">
            {backupMsg}
          </div>
        )}
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard
          icon={<Clock size={18} className="text-primary" />}
          label="Last Backup"
          value={stats?.last_run || '—'}
        />
        <StatCard
          icon={<Clock size={18} className="text-warning" />}
          label="Next Backup"
          value={stats?.next_run || '—'}
        />
        <StatCard
          icon={<Files size={18} className="text-success" />}
          label="Files (last run)"
          value={stats?.last_files?.toString() || '—'}
        />
        <StatCard
          icon={<HardDrive size={18} className="text-primary-light" />}
          label="Size (last run)"
          value={stats?.last_bytes ? fmtBytes(stats.last_bytes) : '—'}
        />
      </div>

      {/* Summary */}
      <div className="grid grid-cols-2 gap-4">
        <InfoRow icon={<CheckCircle2 size={15} className="text-success" />} label="Total backup jobs" value={stats?.total_jobs?.toString() || '0'} />
        <InfoRow icon={<XCircle size={15} className="text-white/30" />}     label="Last job ID"      value={stats?.last_job_id || '—'} />
      </div>
    </div>
  )
}

function StatCard({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="rounded-xl border border-white/5 bg-navy-700 p-5">
      <div className="flex items-center gap-2 mb-3">
        {icon}
        <span className="text-xs text-white/40 font-medium uppercase tracking-wide">{label}</span>
      </div>
      <div className="text-xl font-semibold text-white">{value}</div>
    </div>
  )
}

function InfoRow({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="flex items-center gap-3 px-4 py-3 rounded-lg bg-navy-700 border border-white/5">
      {icon}
      <span className="text-sm text-white/40">{label}</span>
      <span className="ml-auto text-sm text-white font-medium">{value}</span>
    </div>
  )
}
