import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { signInWithPopup } from 'firebase/auth'
import api from '../services/api'
import { auth, githubProvider, googleProvider } from '../utils/firebase'

function GoogleIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" aria-hidden="true">
      <path
        d="M21.35 11.1H12v2.98h5.36c-.23 1.52-1.68 4.46-5.36 4.46-3.22 0-5.84-2.67-5.84-5.96s2.62-5.96 5.84-5.96c1.84 0 3.08.79 3.79 1.46l2.58-2.5C16.73 4.05 14.57 3 12 3 7.03 3 3 7.03 3 12s4.03 9 9 9c5.2 0 8.64-3.65 8.64-8.8 0-.59-.06-1.03-.14-1.5Z"
        fill="#FABC05"
      />
      <path
        d="M6.56 14.56a5.39 5.39 0 0 1 0-5.12L3.64 7.17a8.98 8.98 0 0 0 0 9.66l2.92-2.27Z"
        fill="#EA4335"
      />
      <path
        d="M12 21c2.44 0 4.49-.8 5.99-2.17l-2.77-2.22c-.74.53-1.74.9-3.22.9-2.87 0-5.3-1.94-6.17-4.55l-2.98 2.3C4.33 18.53 7.89 21 12 21Z"
        fill="#34A853"
      />
      <path
        d="M12 6.49c1.61 0 2.71.69 3.34 1.27l2.43-2.37C16.48 4.18 14.44 3 12 3 7.89 3 4.33 5.47 2.85 9.04l2.98 2.31C6.7 8.43 9.13 6.49 12 6.49Z"
        fill="#4285F4"
      />
    </svg>
  )
}

function GithubIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current" aria-hidden="true">
      <path d="M12 .5a12 12 0 0 0-3.79 23.39c.6.11.82-.26.82-.58v-2.23c-3.34.73-4.04-1.42-4.04-1.42-.55-1.38-1.33-1.75-1.33-1.75-1.09-.75.08-.73.08-.73 1.2.09 1.84 1.26 1.84 1.26 1.08 1.86 2.84 1.32 3.53 1 .11-.79.42-1.33.76-1.64-2.67-.31-5.48-1.36-5.48-6.04 0-1.33.47-2.41 1.24-3.25-.12-.31-.54-1.56.12-3.25 0 0 1.01-.33 3.31 1.24a11.3 11.3 0 0 1 6.02 0c2.3-1.57 3.31-1.24 3.31-1.24.66 1.69.24 2.94.12 3.25.77.84 1.24 1.92 1.24 3.25 0 4.69-2.82 5.73-5.51 6.03.43.37.82 1.11.82 2.24v3.32c0 .32.22.69.83.58A12 12 0 0 0 12 .5Z" />
    </svg>
  )
}

function LoginPage() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const persistSession = (responseData) => {
    const token = responseData?.token
    if (!token) {
      setError('Invalid response from server')
      return false
    }
    localStorage.setItem('token', token)
    if (responseData?.user) localStorage.setItem('user', JSON.stringify(responseData.user))
    return true
  }

  const handleSubmit = async (event) => {
    event.preventDefault()
    setLoading(true)
    setError('')
    try {
      const response = await api.post('/login', { email, password })
      if (!persistSession(response?.data)) return
      navigate('/dashboard')
    } catch (err) {
      setError(err?.response?.data?.error || 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  const handleSocialLogin = async (provider) => {
    setLoading(true)
    setError('')
    try {
      const result = await signInWithPopup(auth, provider)
      const idToken = await result.user.getIdToken()
      const response = await api.post('/auth/social', { token: idToken, role: 'student' })
      if (!persistSession(response?.data)) return
      navigate('/dashboard')
    } catch (err) {
      setError(err?.message || 'Social login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-dark-background px-4 py-12">
      <div className="mx-auto w-full max-w-md space-y-6">
        <header className="space-y-2 text-center">
          <div className="mx-auto flex h-11 w-11 items-center justify-center rounded-xl border border-blue-400/30 bg-blue-500/10 text-blue-200">
            <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden="true">
              <path d="M12 2 3 6.5 12 11l7-3.5V14h2V6.5L12 2Zm-7 8.2V15c0 2.97 3.47 5 7 5s7-2.03 7-5v-4.8l-7 3.5-7-3.5Z" />
            </svg>
          </div>
          <h1 className="text-3xl font-semibold text-slate-100">AI Teaching Assistant</h1>
          <p className="text-sm text-slate-300">Sign in to continue to your learning workspace.</p>
        </header>

        <section className="app-card p-6">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="mb-1 block text-xs uppercase tracking-wide text-slate-400">Email</label>
              <input
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                className="input-field"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-xs uppercase tracking-wide text-slate-400">Password</label>
              <input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                className="input-field"
                required
              />
            </div>

            {error ? <p className="text-sm text-red-300">{error}</p> : null}

            <button type="submit" disabled={loading} className="btn-primary w-full">
              {loading ? 'Signing in...' : 'Sign In'}
            </button>
          </form>

          <div className="my-5 h-px bg-white/10" />
          <p className="mb-3 text-center text-xs uppercase tracking-[0.14em] text-slate-400">Continue With</p>

          <div className="grid grid-cols-2 gap-3">
            <button
              type="button"
              onClick={() => handleSocialLogin(googleProvider)}
              disabled={loading}
              className="btn-social app-card-hover"
            >
              <GoogleIcon />
              <span>Google</span>
            </button>
            <button
              type="button"
              onClick={() => handleSocialLogin(githubProvider)}
              disabled={loading}
              className="btn-social app-card-hover"
            >
              <GithubIcon />
              <span>GitHub</span>
            </button>
          </div>
        </section>

        <p className="text-center text-sm text-slate-400">
          Don&apos;t have an account?{' '}
          <Link to="/signup" className="text-slate-200 hover:text-blue-300">
            Create one
          </Link>
        </p>
      </div>
    </div>
  )
}

export default LoginPage
