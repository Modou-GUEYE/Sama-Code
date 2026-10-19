import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api, ApiError, onSessionExpired, session } from '../api/client'
import type { Me, Role, TokenPair } from '../api/types'

const STAFF_ROLES: Role[] = ['editor', 'validator', 'admin']

interface AuthState {
  me: Me | null
  loading: boolean
  login: (phone: string, pin: string) => Promise<void>
  logout: () => Promise<void>
  can: (...roles: Role[]) => boolean
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [me, setMe] = useState<Me | null>(null)
  const [loading, setLoading] = useState(true)

  const loadMe = useCallback(async () => {
    const profile = await api.get<Me>('/me')
    if (!profile.roles.some((r) => STAFF_ROLES.includes(r))) {
      session.clear()
      throw new ApiError(403, 'not_staff', "Ce compte n'a pas accès à l'administration. Demandez un rôle à un administrateur.")
    }
    setMe(profile)
  }, [])

  useEffect(() => {
    onSessionExpired(() => setMe(null))
    if (!session.refresh) {
      setLoading(false)
      return
    }
    loadMe()
      .catch(() => session.clear())
      .finally(() => setLoading(false))
  }, [loadMe])

  const login = useCallback(
    async (phone: string, pin: string) => {
      const pair = await api.post<TokenPair>('/auth/login', { phone_number: phone, pin })
      session.save(pair)
      await loadMe()
    },
    [loadMe],
  )

  const logout = useCallback(async () => {
    const refresh = session.refresh
    session.clear()
    setMe(null)
    if (refresh) await api.post('/auth/logout', { refresh_token: refresh }).catch(() => {})
  }, [])

  const can = useCallback((...roles: Role[]) => !!me && me.roles.some((r) => r === 'admin' || roles.includes(r)), [me])

  const value = useMemo(() => ({ me, loading, login, logout, can }), [me, loading, login, logout, can])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth hors de AuthProvider')
  return ctx
}
