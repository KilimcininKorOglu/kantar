import { NavLink } from 'react-router'
import { useTranslation } from 'react-i18next'
import {
  LayoutDashboard, Package, Database, Users, Shield,
  FileText, Settings, Scale
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

const navItems: { to: string; labelKey: string; icon: LucideIcon }[] = [
  { to: '/', labelKey: 'nav.overview', icon: LayoutDashboard },
  { to: '/packages', labelKey: 'nav.packages', icon: Package },
  { to: '/registries', labelKey: 'nav.registries', icon: Database },
  { to: '/users', labelKey: 'nav.users', icon: Users },
  { to: '/policies', labelKey: 'nav.policies', icon: Shield },
  { to: '/audit', labelKey: 'nav.auditLog', icon: FileText },
  { to: '/settings', labelKey: 'nav.settings', icon: Settings },
]

export default function Sidebar() {
  const { t } = useTranslation()

  return (
    <aside className="w-56 bg-surface border-r border-border flex flex-col min-h-screen">
      <div className="px-5 py-5 border-b border-border">
        <div className="flex items-center gap-2.5">
          <Scale className="w-5 h-5 text-accent" />
          <h1 className="text-lg font-bold text-text tracking-tight">Kantar</h1>
        </div>
        <p className="text-[11px] text-text-dim mt-1 tracking-wide uppercase">{t('app.subtitle')}</p>
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
            {t(item.labelKey)}
          </NavLink>
        ))}
      </nav>
      <div className="px-5 py-3 border-t border-border">
        <p className="text-[10px] text-text-dim font-mono">v0.1.0</p>
      </div>
    </aside>
  )
}
