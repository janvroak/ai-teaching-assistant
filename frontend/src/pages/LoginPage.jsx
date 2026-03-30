import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import Layout from '../components/Layout'
import api from '../services/api'

function LoginPage() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (event) => {
    event.preventDefault()
    setError('')
    setLoading(true)

    try {
      const response = await api.post('/login', { email, password })
      const token = response?.data?.token

      if (!token) {
        setError('Token missing in response')
        return
      }

      localStorage.setItem('token', token)
      if (response?.data?.user) {
        localStorage.setItem('user', JSON.stringify(response.data.user))
      }

      navigate('/dashboard', { replace: true })
    } catch (err) {
      const message = err?.response?.data?.error || 'Login failed'
      setError(message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Layout>
      <div className="mx-auto mt-10 max-w-md rounded-xl bg-white p-6 shadow">
        <h1 className="text-2xl font-semibold text-slate-900">Login</h1>
        <p className="mt-1 text-sm text-slate-500">Welcome back</p>

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <div>
            <label htmlFor="email" className="mb-1 block text-sm font-medium text-slate-700">
              Email
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              className="w-full rounded border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>

          <div>
            <label htmlFor="password" className="mb-1 block text-sm font-medium text-slate-700">
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              className="w-full rounded border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>

          {error ? <p className="text-sm text-red-600">{error}</p> : null}

          <button
            type="submit"
            disabled={loading}
            className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600 disabled:opacity-60"
          >
            {loading ? 'Logging in...' : 'Login'}
          </button>
        </form>

        <p className="mt-4 text-sm text-slate-600">
          Don&apos;t have an account?{' '}
          <Link className="font-medium text-slate-900 underline" to="/signup">
            Signup
          </Link>
        </p>
      </div>
    </Layout>
  )
}

export default LoginPage
