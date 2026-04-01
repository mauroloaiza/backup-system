import { Outlet, useLocation } from 'react-router-dom'
import { Sidebar } from './Sidebar'

const titles: Record<string, string> = {
  '/': 'Dashboard',
  '/jobs': 'Jobs',
  '/nodes': 'Nodos',
  '/history': 'Historial',
  '/destinations': 'Destinos',
  '/settings': 'Configuración',
}

export function Layout() {
  const { pathname } = useLocation()
  const title = titles[pathname] ?? 'BackupSMC'

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
            <button className="relative p-1.5 rounded-lg hover:bg-gray-100 transition-colors">
              <svg className="w-5 h-5 text-gray-500" fill="none" viewBox="0 0 24 24"
                stroke="currentColor" strokeWidth={1.8}>
                <path strokeLinecap="round" strokeLinejoin="round"
                  d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
              </svg>
              <span className="absolute top-0.5 right-0.5 w-2 h-2 bg-red-500 rounded-full border-2 border-white pulse" />
            </button>

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
