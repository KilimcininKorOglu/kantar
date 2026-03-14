import { NavLink } from 'react-router'

const navItems = [
  { to: '/', label: 'Overview', icon: '◈' },
  { to: '/packages', label: 'Packages', icon: '◻' },
  { to: '/registries', label: 'Registries', icon: '◎' },
  { to: '/users', label: 'Users', icon: '◉' },
  { to: '/policies', label: 'Policies', icon: '◇' },
  { to: '/audit', label: 'Audit Log', icon: '◆' },
  { to: '/settings', label: 'Settings', icon: '⚙' },
]

export default function Sidebar() {
  return (
    <aside className="w-60 bg-slate-900 border-r border-slate-800 flex flex-col min-h-screen">
      <div className="p-5 border-b border-slate-800">
        <h1 className="text-xl font-bold">
          <span className="text-blue-400">Kantar</span>
        </h1>
        <p className="text-xs text-slate-500 mt-1">Package Registry</p>
      </div>
      <nav className="flex-1 py-4">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            className={({ isActive }) =>
              `flex items-center gap-3 px-5 py-2.5 text-sm transition-colors ${
                isActive
                  ? 'bg-slate-800 text-white border-r-2 border-blue-400'
                  : 'text-slate-400 hover:text-white hover:bg-slate-800/50'
              }`
            }
          >
            <span className="text-base">{item.icon}</span>
            {item.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
