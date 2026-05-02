import { createContext, useContext, useEffect, useState } from 'react'

const AuthContext = createContext()
const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080').replace(/\/$/, '')

async function authRequest(path, options = {}) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    ...options,
  })
  const text = await response.text()
  let payload = null
  try { payload = JSON.parse(text) } catch { /* empty */ }
  if (!response.ok) throw new Error(payload?.error || 'Request failed')
  return payload
}

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    if (!token) {
      setLoading(false)
      return
    }
    authRequest('/api/auth/me', {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(setUser)
      .catch(() => {
        localStorage.removeItem('auth_token')
      })
      .finally(() => setLoading(false))
  }, [])

  const login = async (email, password) => {
    const data = await authRequest('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })
    localStorage.setItem('auth_token', data.token)
    setUser(data.user)
    return data
  }

  const logout = async () => {
    const token = localStorage.getItem('auth_token')
    if (token) {
      try {
        await authRequest('/api/auth/logout', {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}` },
        })
      } catch { /* ignore */ }
    }
    localStorage.removeItem('auth_token')
    setUser(null)
  }

  const register = async (name, email, password) => {
    return authRequest('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify({ name, email, password }),
    })
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        isAuthenticated: !!user,
        login,
        logout,
        register,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  return useContext(AuthContext)
}
