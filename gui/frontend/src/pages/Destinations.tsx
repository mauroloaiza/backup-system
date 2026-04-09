import { useEffect, useState } from 'react'
import { HardDrive, Cloud, Server, Save, AlertCircle } from 'lucide-react'
import { GetConfig, SaveConfig, BrowseFolder } from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'

type Cfg = main.UIConfig
type DestType = 'local' | 's3' | 'sftp'

const TABS: { id: DestType; label: string; icon: React.ReactNode }[] = [
  { id: 'local', label: 'Local',      icon: <HardDrive size={15} /> },
  { id: 's3',    label: 'Amazon S3',  icon: <Cloud size={15} /> },
  { id: 'sftp',  label: 'SFTP',       icon: <Server size={15} /> },
]

export default function Destinations() {
  const [cfg, setCfg]     = useState<Cfg | null>(null)
  const [saved, setSaved] = useState(false)
  const [err, setErr]     = useState('')

  useEffect(() => { GetConfig().then(setCfg) }, [])

  const save = async () => {
    if (!cfg) return
    setErr('')
    const e = await SaveConfig(cfg)
    if (e) { setErr(e); return }
    setSaved(true)
    setTimeout(() => setSaved(false), 3000)
  }

  const setDest = (t: DestType) => cfg && setCfg({ ...cfg, dest_type: t })

  const handleBrowse = async () => {
    const dir = await BrowseFolder()
    if (dir && cfg) setCfg({ ...cfg, local_path: dir })
  }

  if (!cfg) return <div className="p-8 text-white/30 text-sm">Loading…</div>

  const active = (cfg.dest_type || 'local') as DestType

  return (
    <div className="p-8 space-y-8 max-w-2xl">
      <div>
        <h1 className="text-2xl font-semibold text-white">Destinations</h1>
        <p className="text-sm text-white/40 mt-0.5">Where backups are stored</p>
      </div>

      {/* Type tabs */}
      <div className="flex gap-1 p-1 rounded-xl bg-navy-800 border border-white/5 w-fit">
        {TABS.map(t => (
          <button
            key={t.id}
            onClick={() => setDest(t.id)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              active === t.id ? 'bg-primary text-white' : 'text-white/40 hover:text-white'
            }`}
          >
            {t.icon}{t.label}
          </button>
        ))}
      </div>

      {/* Local */}
      {active === 'local' && (
        <Card title="Local destination">
          <div className="flex gap-2">
            <div className="flex-1 flex items-center gap-3 px-4 py-2.5 rounded-lg bg-navy-900 border border-white/10 text-sm text-white/70 min-w-0">
              <HardDrive size={15} className="text-primary shrink-0" />
              <span className="truncate">{cfg.local_path || 'No folder selected'}</span>
            </div>
            <button onClick={handleBrowse} className="px-4 py-2.5 rounded-lg bg-primary hover:bg-primary-hover text-white text-sm font-medium transition-all whitespace-nowrap">
              Browse…
            </button>
          </div>
          <p className="text-xs text-white/30 mt-2">Backups are stored encrypted in this folder.</p>
        </Card>
      )}

      {/* S3 */}
      {active === 's3' && (
        <Card title="Amazon S3">
          <div className="space-y-3">
            <Field label="Bucket name">
              <input value={cfg.s3_bucket || ''} onChange={e => setCfg({ ...cfg, s3_bucket: e.target.value })} placeholder="my-backup-bucket" className="inp" />
            </Field>
            <Field label="Region">
              <input value={cfg.s3_region || ''} onChange={e => setCfg({ ...cfg, s3_region: e.target.value })} placeholder="us-east-1" className="inp" />
            </Field>
            <Field label="Prefix (optional)">
              <input value={cfg.s3_prefix || ''} onChange={e => setCfg({ ...cfg, s3_prefix: e.target.value })} placeholder="backups/server01" className="inp" />
            </Field>
            <p className="text-xs text-white/30 pt-1">
              Credentials via: <span className="font-mono text-white/50">AWS_ACCESS_KEY_ID</span> · <span className="font-mono text-white/50">AWS_SECRET_ACCESS_KEY</span>
            </p>
          </div>
        </Card>
      )}

      {/* SFTP */}
      {active === 'sftp' && (
        <Card title="SFTP">
          <div className="space-y-3">
            <Field label="Host">
              <input value={cfg.sftp_host || ''} onChange={e => setCfg({ ...cfg, sftp_host: e.target.value })} placeholder="backup.example.com:22" className="inp" />
            </Field>
            <Field label="User">
              <input value={cfg.sftp_user || ''} onChange={e => setCfg({ ...cfg, sftp_user: e.target.value })} placeholder="backup" className="inp" />
            </Field>
            <Field label="Remote path">
              <input value={cfg.sftp_path || ''} onChange={e => setCfg({ ...cfg, sftp_path: e.target.value })} placeholder="/backups/server01" className="inp" />
            </Field>
            <p className="text-xs text-white/30 pt-1">Auth via SSH key: <span className="font-mono text-white/50">~/.ssh/id_rsa</span></p>
          </div>
        </Card>
      )}

      {/* Encryption */}
      <Card title="Encryption">
        <Field label="Passphrase">
          <input
            type="password"
            value={cfg.encryption_passphrase || ''}
            onChange={e => setCfg({ ...cfg, encryption_passphrase: e.target.value })}
            placeholder="Min. 16 characters"
            className="inp"
          />
        </Field>
        <p className="text-xs text-white/30 mt-2">AES-256-GCM. Keep this safe — without it backups cannot be restored.</p>
      </Card>

      {/* Save */}
      <div className="flex items-center gap-3">
        <button onClick={save} className="flex items-center gap-2 px-5 py-2.5 rounded-lg bg-primary hover:bg-primary-hover text-white text-sm font-medium transition-all">
          <Save size={14} />Save changes
        </button>
        {saved && <span className="text-sm text-success">Saved ✓</span>}
        {err && <div className="flex items-start gap-2 text-sm text-danger max-w-md"><AlertCircle size={14} className="shrink-0 mt-0.5" /><span className="break-all">{err}</span></div>}
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

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <label className="text-xs text-white/50 font-medium">{label}</label>
      {children}
    </div>
  )
}

