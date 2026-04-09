import { useEffect, useState } from 'react'
import { FolderOpen, Plus, Trash2, Save, AlertCircle } from 'lucide-react'
import { GetConfig, SaveConfig, BrowseFolder } from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'

type Cfg = main.UIConfig

export default function Sources() {
  const [cfg, setCfg]     = useState<Cfg | null>(null)
  const [saved, setSaved] = useState(false)
  const [err, setErr]     = useState('')
  const [newExclude, setNewExclude] = useState('')

  useEffect(() => { GetConfig().then(setCfg) }, [])

  const handleBrowseAdd = async () => {
    const dir = await BrowseFolder()
    if (dir && cfg) {
      const paths = [...(cfg.source_paths || [])]
      if (!paths.includes(dir)) paths.push(dir)
      setCfg({ ...cfg, source_paths: paths })
    }
  }

  const removeSource = (i: number) => {
    if (!cfg) return
    const paths = [...(cfg.source_paths || [])]
    paths.splice(i, 1)
    setCfg({ ...cfg, source_paths: paths })
  }

  const addExclude = () => {
    if (!newExclude.trim() || !cfg) return
    setCfg({ ...cfg, exclude_patterns: [...(cfg.exclude_patterns || []), newExclude.trim()] })
    setNewExclude('')
  }

  const removeExclude = (i: number) => {
    if (!cfg) return
    const p = [...(cfg.exclude_patterns || [])]
    p.splice(i, 1)
    setCfg({ ...cfg, exclude_patterns: p })
  }

  const save = async () => {
    if (!cfg) return
    setErr('')
    const e = await SaveConfig(cfg)
    if (e) { setErr(e); return }
    setSaved(true)
    setTimeout(() => setSaved(false), 3000)
  }

  if (!cfg) return <div className="p-8 text-white/30 text-sm">Loading…</div>

  const sources = cfg.source_paths || []

  return (
    <div className="p-8 space-y-8 max-w-2xl">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold text-white">Sources</h1>
        <p className="text-sm text-white/40 mt-0.5">Configure which folders to back up</p>
      </div>

      {/* Source folders */}
      <div className="rounded-xl border border-white/5 bg-navy-700 p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-medium text-white/60 uppercase tracking-wide">Backup folders</h2>
          <button
            onClick={handleBrowseAdd}
            className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-primary hover:bg-primary-hover text-white text-xs font-medium transition-all"
          >
            <Plus size={13} />
            Add folder
          </button>
        </div>

        {sources.length === 0 ? (
          <button
            onClick={handleBrowseAdd}
            className="w-full flex flex-col items-center justify-center gap-2 py-10 rounded-lg border-2 border-dashed border-white/10 hover:border-primary/40 hover:bg-primary/5 text-white/30 hover:text-white/60 transition-all"
          >
            <FolderOpen size={24} />
            <span className="text-sm">Click to add a folder</span>
          </button>
        ) : (
          <div className="space-y-2">
            {sources.map((p, i) => (
              <div key={i} className="flex items-center gap-3 px-4 py-2.5 rounded-lg bg-navy-900 border border-white/5 group">
                <FolderOpen size={15} className="text-primary shrink-0" />
                <span className="text-sm text-white/70 truncate flex-1 font-mono">{p}</span>
                <button
                  onClick={() => removeSource(i)}
                  className="text-white/20 hover:text-danger transition-colors opacity-0 group-hover:opacity-100"
                >
                  <Trash2 size={13} />
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="border-t border-white/5 pt-4 space-y-3">
          <Row label="Incremental backup">
            <Toggle value={cfg.incremental} onChange={v => setCfg({ ...cfg, incremental: v })} />
          </Row>
          <Row label="Use VSS (Volume Shadow Copy)">
            <Toggle value={cfg.use_vss} onChange={v => setCfg({ ...cfg, use_vss: v })} />
          </Row>
          <Row label="Schedule interval">
            <input
              value={cfg.schedule_interval || '24h'}
              onChange={e => setCfg({ ...cfg, schedule_interval: e.target.value })}
              className="w-28 px-3 py-1.5 rounded-lg bg-navy-900 border border-white/10 text-sm text-white/80 text-right focus:border-primary/50 transition-all"
            />
          </Row>
          <Row label="Retention (days)">
            <input
              type="number"
              value={cfg.retention_days || 30}
              onChange={e => setCfg({ ...cfg, retention_days: Number(e.target.value) })}
              className="w-20 px-3 py-1.5 rounded-lg bg-navy-900 border border-white/10 text-sm text-white/80 text-right focus:border-primary/50 transition-all"
              min={0}
            />
          </Row>
        </div>
      </div>

      {/* Exclusions */}
      <div className="rounded-xl border border-white/5 bg-navy-700 p-6 space-y-4">
        <h2 className="text-sm font-medium text-white/60 uppercase tracking-wide">Exclude patterns</h2>
        <div className="space-y-1.5">
          {(cfg.exclude_patterns || []).map((p, i) => (
            <div key={i} className="flex items-center justify-between px-3 py-2 rounded-lg bg-navy-900 border border-white/5 group">
              <span className="text-sm text-white/70 font-mono">{p}</span>
              <button onClick={() => removeExclude(i)} className="text-white/20 hover:text-danger transition-colors ml-2 opacity-0 group-hover:opacity-100">
                <Trash2 size={13} />
              </button>
            </div>
          ))}
        </div>
        <div className="flex gap-2">
          <input
            value={newExclude}
            onChange={e => setNewExclude(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && addExclude()}
            placeholder="e.g. *.log or node_modules"
            className="flex-1 px-3 py-2 rounded-lg bg-navy-900 border border-white/10 text-sm text-white/80 placeholder-white/20 focus:border-primary/50 transition-all"
          />
          <button onClick={addExclude} className="p-2 rounded-lg bg-primary hover:bg-primary-hover text-white transition-all">
            <Plus size={16} />
          </button>
        </div>
      </div>

      {/* Save bar */}
      <div className="flex items-center gap-3">
        <button onClick={save} className="flex items-center gap-2 px-5 py-2.5 rounded-lg bg-primary hover:bg-primary-hover text-white text-sm font-medium transition-all">
          <Save size={14} />
          Save changes
        </button>
        {saved && <span className="text-sm text-success">Saved ✓</span>}
        {err && (
          <div className="flex items-start gap-2 text-sm text-danger max-w-md">
            <AlertCircle size={14} className="shrink-0 mt-0.5" />
            <span className="break-all">{err}</span>
          </div>
        )}
      </div>
    </div>
  )
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between py-1">
      <span className="text-sm text-white/60">{label}</span>
      {children}
    </div>
  )
}

function Toggle({ value, onChange }: { value: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      onClick={() => onChange(!value)}
      className={`relative w-10 h-5 rounded-full transition-colors ${value ? 'bg-primary' : 'bg-white/10'}`}
    >
      <span className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow transition-transform ${value ? 'translate-x-5' : ''}`} />
    </button>
  )
}

export { Row, Toggle }
