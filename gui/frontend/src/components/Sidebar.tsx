import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard, FolderOpen, Database, Server, ScrollText, Shield,
} from 'lucide-react'

const links = [
  { to: '/dashboard',    label: 'Dashboard',    icon: LayoutDashboard },
  { to: '/sources',      label: 'Sources',      icon: FolderOpen },
  { to: '/destinations', label: 'Destinations', icon: Database },
  { to: '/server',       label: 'API Server',   icon: Server },
  { to: '/logs',         label: 'Logs',         icon: ScrollText },
]

export default function Sidebar() {
  return (
    <aside
      className="flex flex-col w-56 shrink-0 bg-navy-800 border-r border-white/5"
      style={{ height: '100vh' }}
    >
      {/* Logo */}
      <div className="flex items-center gap-3 px-5 py-5 border-b border-white/5">
        <div className="flex items-center justify-center w-8 h-8 rounded-lg bg-primary">
          <Shield size={16} className="text-white" />
        </div>
        <div>
          <div className="text-sm font-semibold text-white leading-tight">BackupSMC</div>
          <div className="text-[10px] text-white/40 leading-tight">v0.1.0</div>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 px-3 py-4 space-y-0.5 overflow-y-auto">
        {links.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-all ${
                isActive
                  ? 'bg-primary text-white font-medium'
                  : 'text-white/50 hover:text-white hover:bg-white/5'
              }`
            }
          >
            <Icon size={16} />
            {label}
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      <div className="px-5 py-4 border-t border-white/5">
        <div className="text-[10px] text-white/25 leading-relaxed">
          SMC Soluciones<br />
          smcsoluciones.com
        </div>
      </div>
    </aside>
  )
}
