import { NavLink } from 'react-router'
import {
  LayoutDashboard, Package, Database, Users, Shield,
  FileText, Settings, Scale
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

const navItems: { to: string; label: string; icon: LucideIcon }[] = [
  { to: '/', label: 'Overview', icon: LayoutDashboard },
  { to: '/packages', label: 'Packages', icon: Package },
  { to: '/registries', label: 'Registries', icon: Database },
  { to: '/users', label: 'Users', icon: Users },
  { to: '/policies', label: 'Policies', icon: Shield },
  { to: '/audit', label: 'Audit Log', icon: FileText },
  { to: '/settings', label: 'Settings', icon: Settings },
]

export default function Sidebar() {
  return (
    <aside className="w-56 bg-surface border-r border-border flex flex-col min-h-screen">
      <div className="px-5 py-5 border-b border-border">
        <div className="flex items-center gap-2.5">
          <Scale className="w-5 h-5 text-accent" />
          <h1 className="text-lg font-bold text-text tracking-tight">Kantar</h1>
        </div>
        <p className="text-[11px] text-text-dim mt-1 tracking-wide uppercase">Package Registry</p>
      </div>
      <nav className="flex-1 py-3">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            className={({ isActive }) =>
              `flex items-center gap-3 px-5 py-2 text-[13px] font-medium ${
                isActive
                  ? 'bg-accent/10 text-accent border-r-2 border-accent'
                  : 'text-text-muted hover:text-text hover:bg-surface-2'
              }`
            }
          >
            <item.icon className="w-4 h-4" />
            {item.label}
          </NavLink>
        ))}
      </nav>
      <div className="px-5 py-3 border-t border-border">
        <p className="text-[10px] text-text-dim font-mono">v1.0.0</p>
      </div>
    </aside>
  )
}
