import { NavLink } from 'react-router-dom'

const nav = [
  {
    to: '/', label: 'Dashboard',
    icon: <path strokeLinecap="round" strokeLinejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />,
  },
  {
    to: '/jobs', label: 'Jobs',
    icon: <path strokeLinecap="round" strokeLinejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />,
  },
  {
    to: '/nodes', label: 'Nodos',
    icon: <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />,
  },
  {
    to: '/history', label: 'Historial',
    icon: <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />,
  },
  {
    to: '/destinations', label: 'Destinos',
    icon: <path strokeLinecap="round" strokeLinejoin="round" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />,
  },
]

const navBottom = [
  {
    to: '/settings', label: 'Configuración',
    icon: <><path strokeLinecap="round" strokeLinejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></>,
  },
]

function Icon({ d }: { d: React.ReactNode }) {
  return (
    <svg className="w-[17px] h-[17px] flex-shrink-0" fill="none" viewBox="0 0 24 24"
      stroke="currentColor" strokeWidth={2}>
      {d}
    </svg>
  )
}

export function Sidebar() {
  return (
    <aside className="w-52 min-h-screen bg-sidebar flex flex-col fixed top-0 left-0 bottom-0 z-40 select-none">

      {/* Logo */}
      <div className="px-4 py-5 flex items-center gap-2.5">
        <div className="w-8 h-8 bg-primary rounded-lg flex items-center justify-center flex-shrink-0">
          <svg className="w-[18px] h-[18px] text-white" fill="none" viewBox="0 0 24 24"
            stroke="currentColor" strokeWidth={2.2}>
            <path strokeLinecap="round" strokeLinejoin="round"
              d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10" />
          </svg>
        </div>
        <div>
          <div className="font-bold text-white text-sm leading-tight">BackupSMC</div>
          <div className="text-[10px] text-blue-400/70">Soporte IT</div>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 px-3 pt-1 pb-4 space-y-0.5 overflow-y-auto">
        {nav.map(item => (
          <NavLink key={item.to} to={item.to} end={item.to === '/'}
            className={({ isActive }) => `nav-item ${isActive ? 'nav-active' : 'nav-inactive'}`}>
            <Icon d={item.icon} />
            {item.label}
            {item.to === '/' && (
              <svg className="w-3.5 h-3.5 ml-auto" fill="none" viewBox="0 0 24 24"
                stroke="currentColor" strokeWidth={2.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
              </svg>
            )}
          </NavLink>
        ))}

        <div className="pt-4 pb-1">
          <p className="text-[10px] font-semibold uppercase tracking-widest text-slate-600 px-3">
            Sistema
          </p>
        </div>

        {navBottom.map(item => (
          <NavLink key={item.to} to={item.to}
            className={({ isActive }) => `nav-item ${isActive ? 'nav-active' : 'nav-inactive'}`}>
            <Icon d={item.icon} />
            {item.label}
          </NavLink>
        ))}
      </nav>

      {/* Version footer */}
      <div className="px-4 py-3 border-t border-white/5">
        <p className="text-[10px] text-slate-600">BackupSMC v0.1.0</p>
      </div>
    </aside>
  )
}
