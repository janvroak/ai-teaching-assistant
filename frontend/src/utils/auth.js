export function getTokenPayload(token) {
  if (!token || typeof token !== 'string') return null

  const parts = token.split('.')
  if (parts.length !== 3) return null

  try {
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4)
    const json = atob(padded)
    return JSON.parse(json)
  } catch {
    return null
  }
}

export function isTokenValid(token) {
  const payload = getTokenPayload(token)
  if (!payload) return false

  if (typeof payload.exp !== 'number') return true
  const nowInSeconds = Math.floor(Date.now() / 1000)
  return payload.exp > nowInSeconds
}

export function clearAuthSession() {
  localStorage.removeItem('token')
  localStorage.removeItem('user')
}
