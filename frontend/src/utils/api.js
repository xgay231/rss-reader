export async function fetchWithAuth(url, options = {}) {
  const accessToken = localStorage.getItem('accessToken')
  const refreshToken = localStorage.getItem('refreshToken')

  const headers = {
    ...options.headers,
    'Content-Type': 'application/json'
  }

  if (accessToken) {
    headers['Authorization'] = `Bearer ${accessToken}`
  }

  const response = await fetch(url, { ...options, headers })

  // If 401, try refreshing token and retry once
  if (response.status === 401 && refreshToken) {
    try {
      const refreshResponse = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refreshToken })
      })

      if (refreshResponse.ok) {
        const data = await refreshResponse.json()
        localStorage.setItem('accessToken', data.accessToken)
        localStorage.setItem('refreshToken', data.refreshToken)
        headers['Authorization'] = `Bearer ${data.accessToken}`
        return fetch(url, { ...options, headers })
      }
    } catch (error) {
      console.error('Failed to refresh token:', error)
    }
    // Refresh failed, clear tokens
    localStorage.removeItem('accessToken')
    localStorage.removeItem('refreshToken')
    // Reload to show login screen
    window.location.reload()
  }

  return response
}
