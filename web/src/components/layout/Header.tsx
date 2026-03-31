import { useAuth } from '../../hooks/useAuth'
import { useNavigate, useLocation } from 'react-router'
import { LogOut, User } from 'lucide-react'

const pageTitles: Record<string, string> = {
  '/': 'Overview',
  '/packages': 'Packages',
  '/registries': 'Registries',
  '/users': 'Users',
  '/policies': 'Policies',
  '/audit': 'Audit Log',
  '/settings': 'Settings',
}

export default function Header() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const title = pageTitles[location.pathname] || 'Kantar'

  return (
    <header className="h-12 bg-surface border-b border-border flex items-center justify-between px-6">
      <h2 className="text-sm font-semibold text-text">{title}</h2>
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2 text-text-muted">
          <User className="w-3.5 h-3.5" />
          <span className="text-xs font-medium">{user?.username}</span>
        </div>
        <button
          onClick={handleLogout}
          className="flex items-center gap-1.5 text-xs text-text-dim hover:text-danger cursor-pointer"
        >
          <LogOut className="w-3.5 h-3.5" />
        </button>
      </div>
    </header>
  )
}
