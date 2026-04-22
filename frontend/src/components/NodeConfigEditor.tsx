import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchNodeConfig, updateNodeConfig, type Node,
  type NodeConfigPayload,
} from '@/lib/api'

/**
 * Remote agent config editor — opens as a right-side drawer.
 *
 * What's editable here reaches the agent within ~60s (the poll interval).
 * What's NOT editable from the UI (by design, never over the wire):
 *   - encryption passphrase
 *   - SFTP / SMTP passwords
 *   - server API token
 *   - destination type + target (change via agent.yaml locally)
 */
export function NodeConfigEditor({ node, onClose }: { node: Node; onClose: () => void }) {
  const qc = useQueryClient()
  const { data, isLoading, error } = useQuery({
    queryKey: ['node-config', node.id],
    queryFn: () => fetchNodeConfig(node.id),
    // Refresh every 10s so "Applied" indicator updates after the agent polls.
    refetchInterval: 10_000,
  })

  // Local draft state — initialized from server, writable by the user.
  const [draft, setDraft] = useState<NodeConfigPayload | null>(null)
  const [saveError, setSaveError] = useState('')

  useEffect(() => {
    if (data && !draft) {
      setDraft(structuredClone(data.payload))
    }
  }, [data, draft])

  const save = useMutation({
    mutationFn: (p: NodeConfigPayload) => updateNodeConfig(node.id, p),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['node-config', node.id] })
      qc.invalidateQueries({ queryKey: ['nodes'] })
      setSaveError('')
    },
    onError: (e: any) => {
      setSaveError(e?.response?.data?.detail || 'Error al guardar')
    },
  })

  // Patch helper — immutable update of a single field
  const patch = <K extends keyof NodeConfigPayload>(k: K, v: NodeConfigPayload[K]) => {
    if (!draft) return
    setDraft({ ...draft, [k]: v })
  }
  const patchEmail = <K extends keyof NodeConfigPayload['email']>(
    k: K, v: NodeConfigPayload['email'][K],
  ) => {
    if (!draft) return
    setDraft({ ...draft, email: { ...draft.email, [k]: v } })
  }

  return (
    <>
      <div className="fixed inset-0 bg-black/40 backdrop-blur-[2px] z-50" onClick={onClose} />
      <div className="fixed inset-y-0 right-0 w-[560px] max-w-full bg-white shadow-2xl z-50 flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-100">
          <div className="min-w-0">
            <p className="text-sm font-bold text-gray-800 truncate">Configuración remota</p>
            <p className="text-xs text-gray-400 font-mono mt-0.5 truncate">{node.hostname}</p>
          </div>
          <div className="flex items-center gap-3">
            {data && <SyncBadge inSync={data.in_sync} version={data.version}
                                lastPulledVersion={data.last_pulled_version} />}
            <button onClick={onClose}
                    className="text-gray-400 hover:text-gray-600 transition-colors p-1 rounded-lg hover:bg-gray-100">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-5 py-5 space-y-5">
          {isLoading ? (
            <LoadingBody />
          ) : error ? (
            <ErrorBody msg={String(error)} />
          ) : !draft ? null : (
            <>
              <Section title="Planificación">
                <NumberField label="Intervalo (minutos)" min={5} max={10080}
                  value={draft.schedule_interval_minutes}
                  onChange={v => patch('schedule_interval_minutes', v)}
                  help="Cada cuánto se ejecuta un backup. 1440 = 24h." />
                <div className="grid grid-cols-2 gap-3">
                  <BoolField label="VSS (Volume Shadow Copy)" value={draft.use_vss}
                    onChange={v => patch('use_vss', v)} />
                  <BoolField label="Incremental" value={draft.incremental}
                    onChange={v => patch('incremental', v)} />
                  <BoolField label="Verificar después de escribir" value={draft.verify_after_backup}
                    onChange={v => patch('verify_after_backup', v)} />
                  <NumberField label="Throttle (MB/s, 0 = ilimitado)" min={0}
                    value={draft.throttle_mbps}
                    onChange={v => patch('throttle_mbps', v)} step={0.1} />
                </div>
              </Section>

              <Section title="Carpetas respaldadas">
                <StringList
                  values={draft.source_paths}
                  onChange={v => patch('source_paths', v)}
                  placeholder="C:\Users\Datos"
                  mono
                />
              </Section>

              <Section title="Patrones de exclusión">
                <StringList
                  values={draft.exclude_patterns}
                  onChange={v => patch('exclude_patterns', v)}
                  placeholder="*.tmp"
                  mono
                />
              </Section>

              <Section title="Scripts (opcional)">
                <StringField label="Pre-backup" value={draft.pre_script ?? ''}
                  onChange={v => patch('pre_script', v || null)} mono
                  placeholder="C:\scripts\pre.bat" />
                <StringField label="Post-backup" value={draft.post_script ?? ''}
                  onChange={v => patch('post_script', v || null)} mono
                  placeholder="C:\scripts\post.bat" />
              </Section>

              <Section title="Retención">
                <NumberField label="Días (política simple, 0 = desactivada)" min={0}
                  value={draft.retention_days}
                  onChange={v => patch('retention_days', v)} />
                <BoolField label="GFS (Grandfather-Father-Son)" value={draft.gfs_enabled}
                  onChange={v => patch('gfs_enabled', v)} />
                {draft.gfs_enabled && (
                  <div className="grid grid-cols-3 gap-3">
                    <NumberField label="Diarios" min={0} value={draft.gfs_keep_daily}
                      onChange={v => patch('gfs_keep_daily', v)} />
                    <NumberField label="Semanales" min={0} value={draft.gfs_keep_weekly}
                      onChange={v => patch('gfs_keep_weekly', v)} />
                    <NumberField label="Mensuales" min={0} value={draft.gfs_keep_monthly}
                      onChange={v => patch('gfs_keep_monthly', v)} />
                  </div>
                )}
              </Section>

              <Section title="Reintentos">
                <div className="grid grid-cols-2 gap-3">
                  <NumberField label="Máx intentos" min={1} max={20}
                    value={draft.retry_max_attempts}
                    onChange={v => patch('retry_max_attempts', v)} />
                  <NumberField label="Delay inicial (seg)" min={1} max={300}
                    value={draft.retry_initial_delay_seconds}
                    onChange={v => patch('retry_initial_delay_seconds', v)} />
                </div>
              </Section>

              <Section title="Logging">
                <SelectField label="Nivel" value={draft.log_level}
                  options={['debug', 'info', 'warn', 'error']}
                  onChange={v => patch('log_level', v as any)} />
              </Section>

              <Section title="Email (notificaciones desde el agente)">
                <p className="text-[11px] text-gray-400 -mt-1 mb-2">
                  Las credenciales SMTP permanecen en el agente. Aquí sólo los toggles y destinatarios.
                </p>
                <BoolField label="Habilitado" value={draft.email.enabled}
                  onChange={v => patchEmail('enabled', v)} />
                {draft.email.enabled && (
                  <>
                    <div className="grid grid-cols-2 gap-3">
                      <BoolField label="Al fallar" value={draft.email.on_failure}
                        onChange={v => patchEmail('on_failure', v)} />
                      <BoolField label="Al completar" value={draft.email.on_success}
                        onChange={v => patchEmail('on_success', v)} />
                    </div>
                    <StringList values={draft.email.to}
                      onChange={v => patchEmail('to', v)}
                      placeholder="alguien@dominio.com"
                      label="Destinatarios"
                    />
                  </>
                )}
              </Section>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-gray-100 px-5 py-3 flex items-center gap-3 bg-gray-50">
          {saveError && (
            <span className="text-xs text-red-500 flex-1 truncate">{saveError}</span>
          )}
          {save.isSuccess && !saveError && (
            <span className="text-xs text-green-600 flex-1">
              Guardado — el agente aplicará en ~60s
            </span>
          )}
          <button
            onClick={onClose}
            className="text-xs text-gray-500 hover:text-gray-700 px-3 py-2 transition-colors"
          >
            Cancelar
          </button>
          <button
            onClick={() => draft && save.mutate(draft)}
            disabled={!draft || save.isPending}
            className="text-xs bg-primary hover:bg-[#3451d1] disabled:opacity-50 text-white
                       font-medium px-4 py-2 rounded-lg transition-colors"
          >
            {save.isPending ? 'Guardando...' : 'Guardar cambios'}
          </button>
        </div>
      </div>
    </>
  )
}

// ── UI primitives ────────────────────────────────────────────────────────────

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section>
      <h4 className="text-[11px] font-semibold uppercase tracking-widest text-gray-400 mb-2">
        {title}
      </h4>
      <div className="space-y-2.5">{children}</div>
    </section>
  )
}

function NumberField({ label, value, onChange, min, max, step, help }:
  { label: string; value: number; onChange: (v: number) => void;
    min?: number; max?: number; step?: number; help?: string }) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-600 mb-1">{label}</label>
      <input type="number" value={value}
        min={min} max={max} step={step ?? 1}
        onChange={e => onChange(Number(e.target.value))}
        className="w-full text-sm bg-white border border-gray-200 rounded-lg px-3 py-2
                   outline-none focus:border-primary focus:ring-1 focus:ring-primary
                   transition-colors" />
      {help && <p className="text-[10px] text-gray-400 mt-1">{help}</p>}
    </div>
  )
}

function StringField({ label, value, onChange, placeholder, mono }:
  { label: string; value: string; onChange: (v: string) => void;
    placeholder?: string; mono?: boolean }) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-600 mb-1">{label}</label>
      <input type="text" value={value} onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className={`w-full text-sm bg-white border border-gray-200 rounded-lg px-3 py-2
                    outline-none focus:border-primary focus:ring-1 focus:ring-primary transition-colors
                    ${mono ? 'font-mono' : ''}`} />
    </div>
  )
}

function BoolField({ label, value, onChange }:
  { label: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="flex items-center gap-2.5 text-xs text-gray-700 cursor-pointer select-none
                      bg-white border border-gray-200 rounded-lg px-3 py-2 hover:border-gray-300 transition-colors">
      <input type="checkbox" checked={value} onChange={e => onChange(e.target.checked)}
        className="w-4 h-4 accent-primary cursor-pointer" />
      <span className="flex-1">{label}</span>
    </label>
  )
}

function SelectField({ label, value, options, onChange }:
  { label: string; value: string; options: string[]; onChange: (v: string) => void }) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-600 mb-1">{label}</label>
      <select value={value} onChange={e => onChange(e.target.value)}
        className="w-full text-sm bg-white border border-gray-200 rounded-lg px-3 py-2
                   outline-none focus:border-primary focus:ring-1 focus:ring-primary transition-colors">
        {options.map(o => <option key={o} value={o}>{o}</option>)}
      </select>
    </div>
  )
}

function StringList({ values, onChange, placeholder, mono, label }:
  { values: string[]; onChange: (v: string[]) => void;
    placeholder?: string; mono?: boolean; label?: string }) {
  const [draft, setDraft] = useState('')
  const add = () => {
    const v = draft.trim()
    if (!v || values.includes(v)) { setDraft(''); return }
    onChange([...values, v])
    setDraft('')
  }
  return (
    <div>
      {label && <label className="block text-xs font-medium text-gray-600 mb-1">{label}</label>}
      {values.length === 0 ? (
        <p className="text-[11px] text-gray-400 italic py-1">Vacío</p>
      ) : (
        <ul className="space-y-1 mb-2">
          {values.map(v => (
            <li key={v} className="group flex items-center gap-2 text-xs bg-gray-50 rounded-lg px-3 py-1.5">
              <span className={`flex-1 text-gray-700 truncate ${mono ? 'font-mono' : ''}`}>{v}</span>
              <button
                onClick={() => onChange(values.filter(x => x !== v))}
                className="opacity-0 group-hover:opacity-100 text-gray-400 hover:text-red-500 transition-all"
              >
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </li>
          ))}
        </ul>
      )}
      <div className="flex gap-2">
        <input type="text" value={draft} onChange={e => setDraft(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && (e.preventDefault(), add())}
          placeholder={placeholder}
          className={`flex-1 text-xs bg-white border border-gray-200 rounded-lg px-3 py-2
                      outline-none focus:border-primary focus:ring-1 focus:ring-primary transition-colors
                      ${mono ? 'font-mono' : ''}`} />
        <button onClick={add} disabled={!draft.trim()}
          className="text-xs bg-primary hover:bg-[#3451d1] disabled:opacity-50 text-white
                     font-medium px-3 py-2 rounded-lg transition-colors">
          Añadir
        </button>
      </div>
    </div>
  )
}

function SyncBadge({ inSync, version, lastPulledVersion }:
  { inSync: boolean; version: number; lastPulledVersion: number }) {
  return (
    <span
      title={`Servidor: v${version} · Agente: v${lastPulledVersion || 0}`}
      className={`inline-flex items-center gap-1.5 text-[10px] font-semibold px-2 py-1 rounded-full
        ${inSync
          ? 'bg-green-50 text-green-700'
          : 'bg-amber-50 text-amber-700'}`}>
      <span className={`w-1.5 h-1.5 rounded-full
        ${inSync ? 'bg-green-500' : 'bg-amber-500 animate-pulse'}`} />
      {inSync ? `Aplicada v${version}` : `Pendiente v${version}`}
    </span>
  )
}

function LoadingBody() {
  return (
    <div className="space-y-3">
      {[...Array(4)].map((_, i) => (
        <div key={i} className="h-20 bg-gray-50 rounded-xl animate-pulse" />
      ))}
    </div>
  )
}

function ErrorBody({ msg }: { msg: string }) {
  return (
    <div className="bg-red-50 border border-red-100 rounded-xl p-4 text-xs text-red-600">
      <p className="font-medium mb-1">No se pudo cargar la configuración</p>
      <p className="text-red-400 font-mono break-all">{msg}</p>
    </div>
  )
}
