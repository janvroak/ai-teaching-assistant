import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import Layout from '../components/Layout'
import api from '../services/api'

function SignupPage() {
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('student')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (event) => {
    event.preventDefault()
    setError('')
    setSuccess('')
    setLoading(true)

    try {
      await api.post('/signup', {
        name,
        email,
        password,
        role,
      })
      setSuccess('Signup successful. Redirecting to login...')
      setTimeout(() => navigate('/login', { replace: true }), 700)
    } catch (err) {
      const message =
        err?.response?.data?.error ||
        (err?.request ? 'Cannot reach backend API. Check backend server and CORS settings.' : '') ||
        err?.message ||
        'Signup failed'
      setError(message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Layout>
      <div className="mx-auto mt-10 max-w-md rounded-xl bg-white p-6 shadow">
        <h1 className="text-2xl font-semibold text-slate-900">Signup</h1>
        <p className="mt-1 text-sm text-slate-500">Create your account</p>

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <div>
            <label htmlFor="name" className="mb-1 block text-sm font-medium text-slate-700">
              Name
            </label>
            <input
              id="name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="w-full rounded border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>

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

          <div>
            <label htmlFor="role" className="mb-1 block text-sm font-medium text-slate-700">
              Role
            </label>
            <select
              id="role"
              value={role}
              onChange={(e) => setRole(e.target.value)}
              className="w-full rounded border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400"
            >
              <option value="student">student</option>
              <option value="professor">professor</option>
            </select>
          </div>

          {error ? <p className="text-sm text-red-600">{error}</p> : null}
          {success ? <p className="text-sm text-green-600">{success}</p> : null}

          <button
            type="submit"
            disabled={loading}
            className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600 disabled:opacity-60"
          >
            {loading ? 'Creating account...' : 'Signup'}
          </button>
        </form>

        <p className="mt-4 text-sm text-slate-600">
          Already have an account?{' '}
          <Link className="font-medium text-slate-900 underline" to="/login">
            Login
          </Link>
        </p>
      </div>
    </Layout>
  )
}

export default SignupPage
