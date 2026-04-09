import { useEffect, useState } from 'react'
import { Server, Key, Eye, EyeOff, Save, AlertCircle } from 'lucide-react'
import { GetConfig, SaveConfig } from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'
import { Row } from './Sources'

type Cfg = main.UIConfig

const LOG_LEVELS = ['debug', 'info', 'warn', 'error']

export default function ServerPage() {
  const [cfg, setCfg]     = useState<Cfg | null>(null)
  const [saved, setSaved] = useState(false)
  const [err, setErr]     = useState('')
  const [showToken, setShowToken] = useState(false)

  useEffect(() => { GetConfig().then(setCfg) }, [])

  const save = async () => {
    if (!cfg) return
    setErr('')
    const e = await SaveConfig(cfg)
    if (e) { setErr(e); return }
    setSaved(true)
    setTimeout(() => setSaved(false), 3000)
  }

  if (!cfg) return <div className="p-8 text-white/30 text-sm">Loading…</div>

  return (
    <div className="p-8 space-y-8 max-w-2xl">
      <div>
        <h1 className="text-2xl font-semibold text-white">API Server</h1>
        <p className="text-sm text-white/40 mt-0.5">Central server connection &amp; log settings</p>
      </div>

      <Card title="Server connection">
        <div className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-xs text-white/50 font-medium">Server URL</label>
            <div className="flex items-center gap-3 px-4 py-2.5 rounded-lg bg-navy-900 border border-white/10">
              <Server size={15} className="text-primary shrink-0" />
              <input
                value={cfg.server_url || ''}
                onChange={e => setCfg({ ...cfg, server_url: e.target.value })}
                placeholder="https://backup.smcsoluciones.com"
                className="flex-1 bg-transparent text-sm text-white/80 placeholder-white/20 outline-none"
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <label className="text-xs text-white/50 font-medium">API Token</label>
            <div className="flex items-center gap-3 px-4 py-2.5 rounded-lg bg-navy-900 border border-white/10">
              <Key size={15} className="text-primary shrink-0" />
              <input
                type={showToken ? 'text' : 'password'}
                value={cfg.api_token || ''}
                onChange={e => setCfg({ ...cfg, api_token: e.target.value })}
                placeholder="Bearer token"
                className="flex-1 bg-transparent text-sm text-white/80 placeholder-white/20 font-mono outline-none"
              />
              <button onClick={() => setShowToken(!showToken)} className="text-white/30 hover:text-white transition-colors">
                {showToken ? <EyeOff size={14} /> : <Eye size={14} />}
              </button>
            </div>
          </div>

          <p className="text-xs text-white/30">
            Leave empty to run offline — backups work without a central server.
          </p>
        </div>
      </Card>

      <Card title="Logging">
        <div className="space-y-3">
          <Row label="Log level">
            <div className="flex gap-1">
              {LOG_LEVELS.map(l => (
                <button
                  key={l}
                  onClick={() => setCfg({ ...cfg, log_level: l })}
                  className={`px-3 py-1 rounded-md text-xs font-medium transition-all ${
                    cfg.log_level === l ? 'bg-primary text-white' : 'bg-white/5 text-white/40 hover:text-white'
                  }`}
                >
                  {l}
                </button>
              ))}
            </div>
          </Row>
          <Row label="Log file path">
            <input
              value={cfg.log_path || ''}
              onChange={e => setCfg({ ...cfg, log_path: e.target.value })}
              placeholder="stdout or full file path"
              className="inp w-72 text-right text-xs font-mono"
            />
          </Row>
        </div>
      </Card>

      <Card title="Config file">
        <div className="flex items-center gap-3 px-4 py-2.5 rounded-lg bg-navy-900 border border-white/5">
          <span className="text-xs text-white/30">Path</span>
          <span className="text-xs text-white/60 font-mono ml-auto truncate">{cfg.config_path}</span>
        </div>
      </Card>

      <div className="flex items-center gap-3">
        <button onClick={save} className="flex items-center gap-2 px-5 py-2.5 rounded-lg bg-primary hover:bg-primary-hover text-white text-sm font-medium transition-all">
          <Save size={14} />Save changes
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

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-white/5 bg-navy-700 p-6 space-y-4">
      <h2 className="text-sm font-medium text-white/60 uppercase tracking-wide">{title}</h2>
      {children}
    </div>
  )
}
