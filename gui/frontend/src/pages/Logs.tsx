import { useEffect, useRef, useState } from 'react'
import { RefreshCw, Download } from 'lucide-react'
import { GetLogs } from '../../wailsjs/go/main/App'

const LEVELS = ['all', 'INFO', 'WARN', 'ERROR'] as const
type Level = typeof LEVELS[number]

function lineColor(line: string): string {
  if (line.includes('ERROR') || line.includes('error')) return 'text-danger'
  if (line.includes('WARN')  || line.includes('warn'))  return 'text-warning'
  if (line.includes('INFO')  || line.includes('info'))  return 'text-white/70'
  if (line.includes('DEBUG') || line.includes('debug')) return 'text-white/30'
  return 'text-white/50'
}

export default function Logs() {
  const [lines, setLines]   = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [filter, setFilter] = useState<Level>('all')
  const [count, setCount]   = useState(200)
  const [autoRefresh, setAutoRefresh] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)

  const load = async () => {
    setLoading(true)
    const l = await GetLogs(count)
    setLines(l || [])
    setLoading(false)
    setTimeout(() => bottomRef.current?.scrollIntoView({ behavior: 'smooth' }), 50)
  }

  useEffect(() => { load() }, [count])

  useEffect(() => {
    if (!autoRefresh) return
    const id = setInterval(load, 3000)
    return () => clearInterval(id)
  }, [autoRefresh, count])

  const visible = filter === 'all'
    ? lines
    : lines.filter(l => l.toUpperCase().includes(filter))

  const exportLogs = () => {
    const blob = new Blob([lines.join('\n')], { type: 'text/plain' })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `backupsmc-${new Date().toISOString().slice(0, 10)}.log`
    a.click()
  }

  return (
    <div className="flex flex-col h-full p-8 gap-5 max-w-5xl">
      {/* Header */}
      <div className="flex items-center justify-between shrink-0">
        <div>
          <h1 className="text-2xl font-semibold text-white">Logs</h1>
          <p className="text-sm text-white/40 mt-0.5">{visible.length} lines shown</p>
        </div>
        <div className="flex items-center gap-2">
          {/* Level filter */}
          <div className="flex gap-1 p-1 rounded-lg bg-navy-800 border border-white/5">
            {LEVELS.map(l => (
              <button
                key={l}
                onClick={() => setFilter(l)}
                className={`px-3 py-1 rounded text-xs font-medium transition-all ${
                  filter === l ? 'bg-primary text-white' : 'text-white/40 hover:text-white'
                }`}
              >
                {l}
              </button>
            ))}
          </div>
          {/* Count */}
          <select
            value={count}
            onChange={e => setCount(Number(e.target.value))}
            className="px-3 py-1.5 rounded-lg bg-navy-800 border border-white/10 text-sm text-white/60"
          >
            {[100, 200, 500, 1000].map(n => <option key={n} value={n}>{n} lines</option>)}
          </select>
          {/* Auto-refresh toggle */}
          <button
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={`px-3 py-1.5 rounded-lg text-xs font-medium border transition-all ${
              autoRefresh
                ? 'bg-success/20 border-success/40 text-success'
                : 'bg-navy-800 border-white/10 text-white/40 hover:text-white'
            }`}
          >
            Live
          </button>
          <button onClick={load} className="p-2 rounded-lg bg-navy-800 border border-white/10 text-white/50 hover:text-white transition-all">
            <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          </button>
          <button onClick={exportLogs} className="p-2 rounded-lg bg-navy-800 border border-white/10 text-white/50 hover:text-white transition-all">
            <Download size={14} />
          </button>
        </div>
      </div>

      {/* Log viewer */}
      <div className="flex-1 overflow-y-auto rounded-xl border border-white/5 bg-navy-950 p-4 font-mono text-xs leading-relaxed min-h-0">
        {visible.length === 0 ? (
          <div className="text-white/20 text-center py-16">No log entries</div>
        ) : (
          visible.map((line, i) => (
            <div key={i} className={`${lineColor(line)} whitespace-pre-wrap break-all hover:bg-white/3 px-1 rounded`}>
              {line}
            </div>
          ))
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}
