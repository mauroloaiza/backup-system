import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchSettings, updateSettings,
  fetchTokens, createToken, revokeToken,
  fetchMailSettings, updateMailSettings, testMailSettings,
  sendDailySummaryTest,
  type AgentToken, type EmailSettings as EmailSettingsDTO,
} from '@/lib/api'
import { fmtDate } from '@/lib/utils'

export function Settings() {
  return (
    <div className="p-6 space-y-5">
      <div>
        <h1 className="text-xl font-bold text-gray-800">Configuración</h1>
        <p className="text-sm text-gray-400 mt-0.5">Ajustes del sistema BackupSMC</p>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-4">
          <ServerInfoCard />
          <NotificationsCard />
          <MailSettingsCard />
        </div>
        <AgentTokensCard />
      </div>
    </div>
  )
}

// ── Server info ───────────────────────────────────────────────────────────────

function ServerInfoCard() {
  const { data } = useQuery({ queryKey: ['settings'], queryFn: fetchSettings })

  return (
    <div className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
      <h3 className="text-sm font-semibold text-gray-700 mb-4 flex items-center gap-2">
        <svg className="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
        </svg>
        Servidor
      </h3>
      <div className="space-y-3">
        <InfoRow label="Versión API" value={data ? `v${data.server_version}` : '—'} />
        <InfoRow label="Estado" value="Activo" valueColor="text-green-600" />
      </div>
    </div>
  )
}

// ── Notifications ─────────────────────────────────────────────────────────────

function NotificationsCard() {
  const qc = useQueryClient()
  const { data } = useQuery({ queryKey: ['settings'], queryFn: fetchSettings })
  const mutation = useMutation({
    mutationFn: updateSettings,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
  })

  const [email, setEmail] = useState('')
  const [emailEditing, setEmailEditing] = useState(false)

  if (!data) return null

  const toggle = (key: keyof typeof data, value: boolean) => {
    mutation.mutate({ [key]: value })
  }

  const saveEmail = () => {
    mutation.mutate({ notify_email_to: email })
    setEmailEditing(false)
  }

  return (
    <div className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
      <h3 className="text-sm font-semibold text-gray-700 mb-4 flex items-center gap-2">
        <svg className="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
        </svg>
        Alertas
      </h3>

      {/* Email destination */}
      <div className="mb-4">
        <label className="block text-xs font-medium text-gray-500 mb-1">Correo destino</label>
        {emailEditing ? (
          <div className="flex gap-2">
            <input
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && saveEmail()}
              placeholder="admin@empresa.com"
              autoFocus
              className="flex-1 text-xs border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:border-primary"
            />
            <button onClick={saveEmail}
              className="text-xs bg-primary text-white px-3 py-2 rounded-lg hover:bg-[#3451d1]">
              Guardar
            </button>
            <button onClick={() => setEmailEditing(false)}
              className="text-xs text-gray-400 hover:text-gray-600 px-2">✕</button>
          </div>
        ) : (
          <div
            onClick={() => { setEmail(data.notify_email_to); setEmailEditing(true) }}
            className="flex items-center gap-2 text-xs border border-gray-200 rounded-lg px-3 py-2
                       cursor-pointer hover:border-primary transition-colors group"
          >
            <span className={`flex-1 ${data.notify_email_to ? 'text-gray-700' : 'text-gray-400 italic'}`}>
              {data.notify_email_to || 'Sin configurar — clic para editar'}
            </span>
            <svg className="w-3.5 h-3.5 text-gray-300 group-hover:text-primary transition-colors"
              fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </div>
        )}
      </div>

      {/* Toggle switches */}
      <div className="space-y-3">
        <ToggleRow
          label="Notificar en fallo"
          value={data.notify_on_failure}
          onChange={v => toggle('notify_on_failure', v)}
        />
        <ToggleRow
          label="Notificar en completado"
          value={data.notify_on_success}
          onChange={v => toggle('notify_on_success', v)}
        />
        <ToggleRow
          label="Resumen diario"
          value={data.notify_daily_summary}
          onChange={v => toggle('notify_daily_summary', v)}
        />
        <ToggleRow
          label="Email habilitado"
          value={data.notify_email_enabled}
          onChange={v => toggle('notify_email_enabled', v)}
        />
      </div>
    </div>
  )
}

// ── Agent Tokens ──────────────────────────────────────────────────────────────

function AgentTokensCard() {
  const qc = useQueryClient()
  const { data: tokens = [], isLoading } = useQuery({
    queryKey: ['tokens'],
    queryFn: fetchTokens,
  })

  const [newName, setNewName] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [copied, setCopied] = useState<number | null>(null)

  const createMutation = useMutation({
    mutationFn: () => createToken(newName || 'nuevo token'),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tokens'] })
      setNewName('')
      setShowCreate(false)
    },
  })

  const revokeMutation = useMutation({
    mutationFn: revokeToken,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tokens'] }),
  })

  const copy = async (token: AgentToken) => {
    await navigator.clipboard.writeText(token.token)
    setCopied(token.id)
    setTimeout(() => setCopied(null), 2000)
  }

  return (
    <div className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm h-full flex flex-col">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-gray-700 flex items-center gap-2">
          <svg className="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
          </svg>
          Tokens de Agente
        </h3>
        <button
          onClick={() => setShowCreate(v => !v)}
          className="flex items-center gap-1.5 text-xs bg-primary text-white px-3 py-1.5
                     rounded-lg hover:bg-[#3451d1] transition-colors"
        >
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
          </svg>
          Nuevo
        </button>
      </div>

      <p className="text-xs text-gray-400 mb-4">
        Los agentes usan estos tokens en el header <code className="bg-gray-100 px-1 rounded">X-Agent-Token</code> para autenticarse.
      </p>

      {showCreate && (
        <div className="flex gap-2 mb-4">
          <input
            type="text"
            value={newName}
            onChange={e => setNewName(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && createMutation.mutate()}
            placeholder="Nombre del token (ej: servidor-web)"
            autoFocus
            className="flex-1 text-xs border border-gray-200 rounded-lg px-3 py-2
                       focus:outline-none focus:border-primary"
          />
          <button
            onClick={() => createMutation.mutate()}
            disabled={createMutation.isPending}
            className="text-xs bg-primary text-white px-3 py-2 rounded-lg hover:bg-[#3451d1]
                       disabled:opacity-50 transition-colors"
          >
            {createMutation.isPending ? '...' : 'Crear'}
          </button>
          <button onClick={() => setShowCreate(false)}
            className="text-xs text-gray-400 hover:text-gray-600 px-2">✕</button>
        </div>
      )}

      <div className="flex-1 space-y-2 overflow-y-auto">
        {isLoading ? (
          [...Array(2)].map((_, i) => (
            <div key={i} className="h-16 bg-gray-50 rounded-xl animate-pulse" />
          ))
        ) : tokens.length === 0 ? (
          <div className="py-8 text-center text-gray-400 text-sm">Sin tokens</div>
        ) : tokens.map(t => (
          <div key={t.id}
            className={`border rounded-xl p-3 transition-colors ${
              t.is_active ? 'border-gray-100 bg-gray-50' : 'border-gray-100 bg-gray-50 opacity-50'
            }`}
          >
            <div className="flex items-center justify-between mb-1.5">
              <div className="flex items-center gap-2">
                <span className={`w-1.5 h-1.5 rounded-full ${t.is_active ? 'bg-green-400' : 'bg-gray-300'}`} />
                <span className="text-xs font-semibold text-gray-700">{t.name}</span>
              </div>
              <div className="flex items-center gap-1">
                <button
                  onClick={() => copy(t)}
                  title="Copiar token"
                  className="text-xs text-gray-400 hover:text-primary transition-colors px-2 py-1
                             rounded hover:bg-gray-100"
                >
                  {copied === t.id ? (
                    <svg className="w-3.5 h-3.5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                  ) : (
                    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                  )}
                </button>
                {t.is_active && (
                  <button
                    onClick={() => revokeMutation.mutate(t.id)}
                    title="Revocar"
                    className="text-xs text-gray-400 hover:text-red-500 transition-colors px-2 py-1
                               rounded hover:bg-red-50"
                  >
                    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                    </svg>
                  </button>
                )}
              </div>
            </div>
            <p className="text-[10px] font-mono text-gray-400 truncate px-0.5">{t.token}</p>
            <div className="flex justify-between mt-1.5">
              <span className="text-[10px] text-gray-400">
                {t.last_used_at ? `Último uso: ${fmtDate(t.last_used_at)}` : 'Nunca usado'}
              </span>
              <span className="text-[10px] text-gray-400">{fmtDate(t.created_at)}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

// ── SMTP / Mail Settings ──────────────────────────────────────────────────────

const PROVIDERS: { value: EmailSettingsDTO['outbound_provider']; label: string }[] = [
  { value: 'smtp',      label: 'SMTP genérico' },
  { value: 'office365', label: 'Office 365' },
]

function MailSettingsCard() {
  const qc = useQueryClient()
  const { data } = useQuery({ queryKey: ['mail-settings'], queryFn: fetchMailSettings })
  const [draft, setDraft] = useState<Partial<EmailSettingsDTO> | null>(null)
  const [testTo, setTestTo] = useState('')
  const [testMsg, setTestMsg] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null)

  const mutation = useMutation({
    mutationFn: updateMailSettings,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['mail-settings'] })
      setDraft(null)
    },
  })

  const testMutation = useMutation({
    mutationFn: (to: string) => testMailSettings(to || undefined),
    onSuccess: (r) => setTestMsg({ kind: 'ok', text: `Correo enviado a ${r.sent_to}` }),
    onError: (e: any) => setTestMsg({ kind: 'err', text: e.response?.data?.detail || e.message || 'Error' }),
  })

  const dailyMutation = useMutation({
    mutationFn: (to: string) => sendDailySummaryTest(to || undefined),
    onSuccess: (r) => setTestMsg({
      kind: 'ok',
      text: r.sent_to
        ? `Resumen diario enviado a ${r.sent_to} · ${r.summary?.runs ?? 0} runs / ${r.summary?.failed ?? 0} fallos`
        : 'Resumen diario enviado',
    }),
    onError: (e: any) => setTestMsg({ kind: 'err', text: e.response?.data?.detail || e.message || 'Error' }),
  })

  if (!data) return null

  const cur: Partial<EmailSettingsDTO> = { ...data, ...(draft ?? {}) }
  const dirty = draft !== null
  const isO365 = cur.outbound_provider === 'office365'

  const set = (patch: Partial<EmailSettingsDTO>) => setDraft(d => ({ ...(d ?? {}), ...patch }))

  return (
    <div className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-gray-700 flex items-center gap-2">
          <svg className="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
          Servidor SMTP
        </h3>
        <span className={`text-[10px] font-semibold px-2 py-0.5 rounded-full ${
          data.configured ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'
        }`}>
          {data.configured ? 'Configurado' : 'Sin configurar'}
        </span>
      </div>

      <div className="space-y-3">
        {/* Provider */}
        <FieldRow label="Proveedor">
          <select
            value={cur.outbound_provider ?? 'smtp'}
            onChange={e => set({ outbound_provider: e.target.value as EmailSettingsDTO['outbound_provider'] })}
            className="text-xs border border-gray-200 rounded-lg px-2 py-1.5 focus:outline-none focus:border-primary bg-white"
          >
            {PROVIDERS.map(p => <option key={p.value} value={p.value}>{p.label}</option>)}
          </select>
        </FieldRow>

        <FieldRow label="From (nombre)">
          <input type="text" value={cur.from_name ?? ''} onChange={e => set({ from_name: e.target.value })}
            placeholder="BackupSMC" className={inputCls} />
        </FieldRow>

        <FieldRow label="From (email)">
          <input type="email" value={cur.from_email ?? ''} onChange={e => set({ from_email: e.target.value })}
            placeholder="backups@empresa.com" className={inputCls} />
        </FieldRow>

        {!isO365 && (
          <>
            <FieldRow label="Host">
              <input type="text" value={cur.smtp_host ?? ''} onChange={e => set({ smtp_host: e.target.value })}
                placeholder="smtp.gmail.com" className={inputCls} />
            </FieldRow>
            <div className="grid grid-cols-2 gap-2">
              <FieldRow label="Puerto">
                <input type="number" value={cur.smtp_port ?? 587}
                  onChange={e => set({ smtp_port: parseInt(e.target.value) || 587 })}
                  className={inputCls} />
              </FieldRow>
              <FieldRow label="SSL (465)">
                <button
                  onClick={() => set({ smtp_secure: !cur.smtp_secure })}
                  className={`w-10 h-5 rounded-full transition-colors relative ${cur.smtp_secure ? 'bg-primary' : 'bg-gray-200'}`}
                >
                  <span className={`absolute top-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${
                    cur.smtp_secure ? 'translate-x-5' : 'translate-x-0.5'}`} />
                </button>
              </FieldRow>
            </div>
          </>
        )}

        <FieldRow label="Usuario">
          <input type="text" value={cur.smtp_user ?? ''} onChange={e => set({ smtp_user: e.target.value })}
            placeholder="backups@empresa.com" className={inputCls} />
        </FieldRow>

        <FieldRow label="Contraseña">
          <input type="password" value={cur.smtp_pass ?? ''}
            onChange={e => set({ smtp_pass: e.target.value })}
            placeholder={data.smtp_pass ? '•••••••• (oculto)' : 'app password'}
            className={inputCls} />
        </FieldRow>

        {/* Save bar */}
        <div className="flex items-center gap-2 pt-2">
          <button
            onClick={() => mutation.mutate(draft ?? {})}
            disabled={!dirty || mutation.isPending}
            className="text-xs bg-primary text-white px-3 py-1.5 rounded-lg hover:bg-[#3451d1] disabled:opacity-50 transition-colors"
          >
            {mutation.isPending ? 'Guardando…' : 'Guardar'}
          </button>
          {dirty && (
            <button onClick={() => setDraft(null)}
              className="text-xs text-gray-400 hover:text-gray-600 transition-colors">
              Cancelar
            </button>
          )}
          {mutation.isError && (
            <span className="text-xs text-red-500">Error al guardar</span>
          )}
        </div>

        {/* Test */}
        <div className="mt-4 pt-4 border-t border-gray-100">
          <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-widest mb-2">
            Probar configuración
          </p>
          <div className="flex gap-2">
            <input
              type="email"
              value={testTo}
              onChange={e => setTestTo(e.target.value)}
              placeholder="tu@correo.com (opcional)"
              className="flex-1 text-xs border border-gray-200 rounded-lg px-3 py-1.5 focus:outline-none focus:border-primary"
            />
            <button
              onClick={() => { setTestMsg(null); testMutation.mutate(testTo) }}
              disabled={!data.configured || testMutation.isPending}
              title={!data.configured ? 'Guarda primero la configuración' : 'Enviar correo de prueba'}
              className="text-xs bg-gray-700 text-white px-3 py-1.5 rounded-lg hover:bg-gray-800 disabled:opacity-50 transition-colors"
            >
              {testMutation.isPending ? 'Enviando…' : 'Enviar prueba'}
            </button>
            <button
              onClick={() => { setTestMsg(null); dailyMutation.mutate(testTo) }}
              disabled={!data.configured || dailyMutation.isPending}
              title={!data.configured ? 'Guarda primero la configuración' : 'Enviar resumen diario ahora'}
              className="text-xs bg-primary text-white px-3 py-1.5 rounded-lg hover:bg-[#3451d1] disabled:opacity-50 transition-colors whitespace-nowrap"
            >
              {dailyMutation.isPending ? 'Generando…' : 'Resumen diario'}
            </button>
          </div>
          <p className="text-[10px] text-gray-400 mt-1.5">
            El resumen diario se envía automáticamente cada día a las 07:00 UTC si la opción está activa.
          </p>
          {testMsg && (
            <p className={`text-[11px] mt-2 ${testMsg.kind === 'ok' ? 'text-green-600' : 'text-red-500'}`}>
              {testMsg.kind === 'ok' ? '✓ ' : '✕ '}{testMsg.text}
            </p>
          )}
        </div>
      </div>
    </div>
  )
}

const inputCls = 'w-full text-xs border border-gray-200 rounded-lg px-2 py-1.5 focus:outline-none focus:border-primary'

function FieldRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="block text-[10px] font-medium text-gray-500 uppercase tracking-wide mb-1">{label}</label>
      {children}
    </div>
  )
}


// ── helpers ───────────────────────────────────────────────────────────────────

function InfoRow({ label, value, valueColor = 'text-gray-700' }: {
  label: string; value: string; valueColor?: string
}) {
  return (
    <div className="flex justify-between py-1.5 border-b border-gray-50 last:border-0">
      <span className="text-xs text-gray-500">{label}</span>
      <span className={`text-xs font-medium ${valueColor}`}>{value}</span>
    </div>
  )
}

function ToggleRow({ label, value, onChange }: {
  label: string; value: boolean; onChange: (v: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-gray-600">{label}</span>
      <button
        onClick={() => onChange(!value)}
        className={`relative w-10 h-5 rounded-full transition-colors focus:outline-none ${
          value ? 'bg-primary' : 'bg-gray-200'
        }`}
      >
        <span
          className={`absolute top-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${
            value ? 'translate-x-5' : 'translate-x-0.5'
          }`}
        />
      </button>
    </div>
  )
}
