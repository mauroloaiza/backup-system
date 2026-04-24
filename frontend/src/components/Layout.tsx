import { useEffect, useState, useRef } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { useAuth } from '@/store/auth'
import { useNotifications } from '@/store/notifications'
import { timeAgo } from '@/lib/utils'
import type { AppNotification } from '@/lib/api'

const titles: Record<string, string> = {
  '/': 'Dashboard',
  '/jobs': 'Jobs',
  '/nodes': 'Nodos',
  '/history': 'Historial',
  '/destinations': 'Destinos',
  '/restore': 'Restauraciones',
  '/settings': 'Configuración',
}

export function Layout() {
  const { pathname } = useLocation()
  const title = titles[pathname] ?? 'BackupSMC'
  const token = useAuth(s => s.token)
  const { load } = useNotifications()

  // Poll every 30s while authenticated.
  useEffect(() => {
    if (!token) return
    load()
    const t = setInterval(load, 30_000)
    return () => clearInterval(t)
  }, [token, load])

  return (
    <div className="flex w-full min-h-screen bg-gray-50">
      <Sidebar />

      <main className="ml-52 flex-1 min-h-screen flex flex-col">
        {/* Top bar */}
        <header className="sticky top-0 z-30 bg-white border-b border-gray-200 px-6 py-3 flex items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            {/* Search */}
            <div className="relative">
              <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400"
                fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round"
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              <input
                placeholder="Buscar..."
                className="border border-gray-200 rounded-md pl-8 pr-16 py-1.5 text-sm text-gray-600
                  placeholder-gray-400 w-48 focus:outline-none focus:border-primary bg-white"
              />
              <span className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-gray-400
                bg-gray-100 px-1.5 py-0.5 rounded font-medium">
                ctrl K
              </span>
            </div>

            {/* Page title badge */}
            <span className="text-sm font-semibold text-gray-700 ml-2">{title}</span>
          </div>

          {/* Right: bell + user */}
          <div className="flex items-center gap-3">
            <NotificationBell />

            <div className="flex items-center gap-2 cursor-pointer hover:bg-gray-50 rounded-lg px-2 py-1 transition-colors">
              <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center text-xs font-bold text-white">
                A
              </div>
              <div className="text-right">
                <div className="text-sm font-semibold text-gray-800 leading-tight">Admin</div>
                <div className="text-xs text-gray-400 leading-tight">Administrador</div>
              </div>
            </div>
          </div>
        </header>

        {/* Page content */}
        <div className="flex-1">
          <Outlet />
        </div>
      </main>
    </div>
  )
}

// ── Notification Bell ────────────────────────────────────────────────────────

function NotificationBell() {
  const [open, setOpen] = useState(false)
  const { notifications, unread, markRead, markAllRead } = useNotifications()
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [open])

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen(v => !v)}
        className="relative p-1.5 rounded-lg hover:bg-gray-100 transition-colors"
        aria-label="Notificaciones"
      >
        <svg className="w-5 h-5 text-gray-500" fill="none" viewBox="0 0 24 24"
          stroke="currentColor" strokeWidth={1.8}>
          <path strokeLinecap="round" strokeLinejoin="round"
            d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
        </svg>
        {unread > 0 && (
          <span className="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 px-1 bg-red-500 text-white text-[9px] font-bold rounded-full flex items-center justify-center border-2 border-white">
            {unread > 99 ? '99+' : unread}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 top-full mt-1 w-[360px] bg-white rounded-xl shadow-xl border border-gray-200 overflow-hidden z-50">
          <div className="flex items-center justify-between px-4 py-3 border-b border-gray-100">
            <div className="flex items-center gap-2">
              <span className="text-sm font-semibold text-gray-800">Notificaciones</span>
              {unread > 0 && (
                <span className="text-[10px] font-bold bg-red-500 text-white px-1.5 py-0.5 rounded-full">
                  {unread}
                </span>
              )}
            </div>
            {unread > 0 && (
              <button
                onClick={markAllRead}
                className="text-xs text-primary hover:text-[#3451d1] font-medium transition-colors"
              >
                Marcar todo
              </button>
            )}
          </div>

          <div className="max-h-[420px] overflow-y-auto">
            {notifications.length === 0 ? (
              <div className="py-10 text-center">
                <svg className="w-8 h-8 text-gray-200 mx-auto mb-2" fill="none" viewBox="0 0 24 24"
                  stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round"
                    d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
                </svg>
                <p className="text-xs text-gray-400">Sin notificaciones</p>
              </div>
            ) : (
              notifications.map(n => (
                <NotificationItem
                  key={n.id}
                  notif={n}
                  onClick={() => {
                    markRead(n.id)
                    setOpen(false)
                  }}
                />
              ))
            )}
          </div>
        </div>
      )}
    </div>
  )
}

const TYPE_STYLES: Record<string, { bg: string; icon: string; path: string }> = {
  backup_completed: { bg: 'bg-green-100',  icon: 'text-green-600',
    path: 'M5 13l4 4L19 7' },
  backup_failed:    { bg: 'bg-red-100',    icon: 'text-red-600',
    path: 'M12 9v2m0 4h.01M4.93 19h14.14c1.54 0 2.5-1.67 1.73-3L13.73 4a2 2 0 00-3.46 0L3.2 16c-.77 1.33.19 3 1.73 3z' },
  backup_warning:   { bg: 'bg-amber-100',  icon: 'text-amber-600',
    path: 'M12 9v2m0 4h.01M4.93 19h14.14c1.54 0 2.5-1.67 1.73-3L13.73 4a2 2 0 00-3.46 0L3.2 16c-.77 1.33.19 3 1.73 3z' },
  node_offline:     { bg: 'bg-slate-100',  icon: 'text-slate-500',
    path: 'M18.364 5.636a9 9 0 11-12.728 0M12 3v9' },
  general:          { bg: 'bg-blue-100',   icon: 'text-blue-600',
    path: 'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z' },
}

function NotificationItem({ notif, onClick }: { notif: AppNotification; onClick: () => void }) {
  const navigate = useNavigate()
  const style = TYPE_STYLES[notif.type] ?? TYPE_STYLES.general

  const handleClick = () => {
    onClick()
    if (notif.entity_type === 'job_run') {
      navigate('/jobs')
    }
  }

  return (
    <div
      onClick={handleClick}
      className={`flex gap-3 px-4 py-3 border-b border-gray-50 last:border-0 cursor-pointer transition-colors ${
        notif.read ? 'bg-white hover:bg-gray-50' : 'bg-blue-50/40 hover:bg-blue-50/70'
      }`}
    >
      <div className={`w-7 h-7 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5 ${
        notif.read ? 'bg-gray-100' : style.bg
      }`}>
        <svg className={`w-3.5 h-3.5 ${notif.read ? 'text-gray-400' : style.icon}`}
          fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d={style.path} />
        </svg>
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-start justify-between gap-2">
          <p className={`text-xs font-semibold leading-tight ${notif.read ? 'text-gray-600' : 'text-gray-900'}`}>
            {notif.title}
          </p>
          {!notif.read && (
            <span className="mt-1 w-1.5 h-1.5 bg-primary rounded-full flex-shrink-0" />
          )}
        </div>
        {notif.body && (
          <p className="text-[11px] text-gray-500 mt-0.5 line-clamp-2">{notif.body}</p>
        )}
        <p className="text-[10px] text-gray-400 mt-1">{timeAgo(notif.created_at)}</p>
      </div>
    </div>
  )
}
