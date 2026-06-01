import { createContext, useContext, useEffect, useState } from 'react'

const AuthContext = createContext()

const API_BASE    = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '')
const LOGIN_URL   = import.meta.env.VITE_LOGIN_URL || 'http://localhost:5175'
const SITE_URL    = import.meta.env.VITE_SITE_URL  || 'http://localhost:5173'

async function apiRequest(path, options = {}) {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    credentials: 'include',
    ...options,
  })
  const text = await response.text()
  let payload = null
  try { payload = JSON.parse(text) } catch { /* empty */ }
  if (!response.ok) throw new Error(payload?.error || 'Request failed')
  return payload
}

export function redirectToLogin() {
  const redirect = encodeURIComponent(window.location.href || SITE_URL)
  window.location.href = `${LOGIN_URL}/login?redirect=${redirect}`
}

export function AuthProvider({ children }) {
  const [user, setUser]       = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    apiRequest('/api/auth/me')
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  const logout = async () => {
    try {
      await apiRequest('/api/auth/logout', { method: 'POST' })
    } catch { /* ignore */ }
    setUser(null)
    redirectToLogin()
  }

  return (
    <AuthContext.Provider value={{ user, loading, isAuthenticated: !!user, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  return useContext(AuthContext)
}
