import { createContext, useState, useCallback, useEffect, type ReactNode } from 'react'
import type { User } from '../api/types'
import { setTimezone } from '../utils/date'

interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  login: (token: string, user: User) => void
  logout: () => void
}

export const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('kantar_token'))
  const [user, setUser] = useState<User | null>(() => {
    const stored = localStorage.getItem('kantar_user')
    return stored ? JSON.parse(stored) : null
  })

  const login = useCallback((newToken: string, newUser: User) => {
    localStorage.setItem('kantar_token', newToken)
    localStorage.setItem('kantar_user', JSON.stringify(newUser))
    if (newUser.timezone) {
      setTimezone(newUser.timezone)
    }
    setToken(newToken)
    setUser(newUser)
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem('kantar_token')
    localStorage.removeItem('kantar_user')
    setToken(null)
    setUser(null)
  }, [])

  useEffect(() => {
    if (token && !user) {
      logout()
    }
  }, [token, user, logout])

  return (
    <AuthContext.Provider value={{ user, token, isAuthenticated: !!token && !!user, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
