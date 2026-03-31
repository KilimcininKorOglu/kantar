import { useAuth } from '../../hooks/useAuth'
import { useNavigate, useLocation } from 'react-router'
import { useTranslation } from 'react-i18next'
import { LogOut, User } from 'lucide-react'

const pageTitleKeys: Record<string, string> = {
  '/': 'nav.overview',
  '/packages': 'nav.packages',
  '/registries': 'nav.registries',
  '/users': 'nav.users',
  '/policies': 'nav.policies',
  '/audit': 'nav.auditLog',
  '/settings': 'nav.settings',
}

export default function Header() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const { t } = useTranslation()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const titleKey = pageTitleKeys[location.pathname] || 'nav.overview'

  return (
    <header className="h-12 bg-surface border-b border-border flex items-center justify-between px-6">
      <h2 className="text-sm font-semibold text-text">{t(titleKey)}</h2>
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
