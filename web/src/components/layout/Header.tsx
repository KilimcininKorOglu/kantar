import { useAuth } from '../../hooks/useAuth'
import { useNavigate } from 'react-router'

export default function Header() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <header className="h-14 bg-slate-900 border-b border-slate-800 flex items-center justify-between px-6">
      <div className="text-sm text-slate-400">
        Dashboard
      </div>
      <div className="flex items-center gap-4">
        <span className="text-sm text-slate-300">{user?.username}</span>
        <button
          onClick={handleLogout}
          className="text-sm text-slate-400 hover:text-white transition-colors cursor-pointer"
        >
          Logout
        </button>
      </div>
    </header>
  )
}
