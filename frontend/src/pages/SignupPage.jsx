import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import api from '../services/api'

function SignupPage() {
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('student')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const handleSubmit = async (event) => {
    event.preventDefault()
    setLoading(true)
    setError('')
    setSuccess('')
    try {
      await api.post('/signup', { name, email, password, role })
      setSuccess('Account created. Redirecting to login...')
      setTimeout(() => navigate('/login', { replace: true }), 700)
    } catch (err) {
      setError(
        err?.response?.data?.error ||
          err?.message ||
          'Signup failed',
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-dark-background px-4 py-12">
      <div className="mx-auto w-full max-w-md space-y-6">
        <header className="space-y-2 text-center">
          <div className="mx-auto flex h-11 w-11 items-center justify-center rounded-xl border border-cyan-400/30 bg-cyan-500/10 text-cyan-200">
            <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden="true">
              <path d="M12 3a5 5 0 1 0 0 10 5 5 0 0 0 0-10Zm-8 18a8 8 0 0 1 16 0H4Z" />
            </svg>
          </div>
          <h1 className="text-3xl font-semibold text-slate-100">Create Account</h1>
          <p className="text-sm text-slate-300">Join the AI Teaching Assistant platform.</p>
        </header>

        <section className="app-card p-6">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="mb-1 block text-xs uppercase tracking-wide text-slate-400">Full Name</label>
              <input
                type="text"
                value={name}
                onChange={(event) => setName(event.target.value)}
                className="input-field"
                required
              />
            </div>
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
            <div>
              <label className="mb-2 block text-xs uppercase tracking-wide text-slate-400">Role</label>
              <div className="grid grid-cols-2 gap-2">
                <button
                  type="button"
                  className={`rounded-xl border px-3 py-2 text-sm transition-all duration-300 ${
                    role === 'student'
                      ? 'border-blue-500/40 bg-blue-500/10 text-slate-100'
                      : 'border-white/10 bg-white/5 text-slate-300 hover:border-blue-500/30'
                  }`}
                  onClick={() => setRole('student')}
                >
                  Student
                </button>
                <button
                  type="button"
                  className={`rounded-xl border px-3 py-2 text-sm transition-all duration-300 ${
                    role === 'professor'
                      ? 'border-blue-500/40 bg-blue-500/10 text-slate-100'
                      : 'border-white/10 bg-white/5 text-slate-300 hover:border-blue-500/30'
                  }`}
                  onClick={() => setRole('professor')}
                >
                  Professor
                </button>
              </div>
            </div>

            {error ? <p className="text-sm text-red-300">{error}</p> : null}
            {success ? <p className="text-sm text-emerald-300">{success}</p> : null}

            <button type="submit" disabled={loading} className="btn-primary w-full">
              {loading ? 'Creating account...' : 'Create Account'}
            </button>
          </form>
        </section>

        <p className="text-center text-sm text-slate-400">
          Already have an account?{' '}
          <Link to="/login" className="text-slate-200 hover:text-blue-300">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  )
}

export default SignupPage
