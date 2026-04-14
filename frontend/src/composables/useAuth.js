import { ref, provide, inject } from 'vue'

const AUTH_KEY = Symbol('auth')

export function provideAuth() {
  const user = ref(null)
  const accessToken = ref(localStorage.getItem('accessToken'))
  const refreshToken = ref(localStorage.getItem('refreshToken'))

  const setTokens = (access, refresh) => {
    accessToken.value = access
    refreshToken.value = refresh
    if (access) localStorage.setItem('accessToken', access)
    else localStorage.removeItem('accessToken')
    if (refresh) localStorage.setItem('refreshToken', refresh)
    else localStorage.removeItem('refreshToken')
  }

  const login = async (email, password) => {
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password })
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Login failed')
    }

    const data = await response.json()
    user.value = data.user
    setTokens(data.accessToken, data.refreshToken)
    return data
  }

  const register = async (email, username, password) => {
    const response = await fetch('/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, username, password })
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Registration failed')
    }

    return await response.json()
  }

  const logout = async () => {
    // Save token before clearing
    const token = accessToken.value
    user.value = null
    setTokens(null, null)
    // Don't await the logout API call, just clear state and reload
    if (token) {
      try {
        fetch('/api/auth/logout', {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        })
      } catch (e) {
        // Ignore logout errors
      }
    }
    // Force reload to clear all pending state
    window.location.reload()
  }

  const refreshAccessToken = async () => {
    if (!refreshToken.value) return false

    try {
      const response = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refreshToken: refreshToken.value })
      })

      if (response.ok) {
        const data = await response.json()
        setTokens(data.accessToken, data.refreshToken)
        return true
      }
    } catch (error) {
      console.error('Failed to refresh token:', error)
    }
    return false
  }

  const fetchCurrentUser = async () => {
    if (!accessToken.value) return null

    try {
      const response = await fetch('/api/auth/me', {
        headers: { 'Authorization': `Bearer ${accessToken.value}` }
      })

      if (response.ok) {
        user.value = await response.json()
        return user.value
      } else if (response.status === 401) {
        const refreshed = await refreshAccessToken()
        if (refreshed) {
          return fetchCurrentUser()
        }
        await logout()
      }
    } catch (error) {
      console.error('Failed to fetch current user:', error)
    }
    return null
  }

  const isAuthenticated = () => !!accessToken.value

  const auth = {
    user,
    accessToken,
    refreshToken,
    login,
    register,
    logout,
    fetchCurrentUser,
    refreshAccessToken,
    isAuthenticated
  }

  provide(AUTH_KEY, auth)
  return auth
}

export function useAuth() {
  const auth = inject(AUTH_KEY)
  if (!auth) {
    throw new Error('useAuth must be used within a component that calls provideAuth')
  }
  return auth
}
