import { createContext, useState, useCallback, useEffect, type ReactNode } from 'react'
import type { User } from '../api/types'
import { api } from '../api/client'
import { setTimezone } from '../utils/date'
import { setLocale } from '../i18n'

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  login: (user: User) => void
  logout: () => void
}

export const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => {
    const stored = localStorage.getItem('kantar_user')
    return stored ? JSON.parse(stored) : null
  })

  const login = useCallback((newUser: User) => {
    localStorage.setItem('kantar_user', JSON.stringify(newUser))
    if (newUser.timezone) {
      setTimezone(newUser.timezone)
    }
    if (newUser.locale) {
      setLocale(newUser.locale)
    }
    setUser(newUser)
  }, [])

  const logout = useCallback(() => {
    api.post('/auth/logout').catch(() => {})
    localStorage.removeItem('kantar_user')
    setUser(null)
  }, [])

  // Validate session on mount — if cookie expired, clear user state
  useEffect(() => {
    if (user) {
      api.get('/profile').catch(() => {
        localStorage.removeItem('kantar_user')
        setUser(null)
      })
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <AuthContext.Provider value={{ user, isAuthenticated: !!user, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
