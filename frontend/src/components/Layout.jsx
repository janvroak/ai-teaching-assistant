import { useNavigate } from 'react-router-dom'

function Layout({ children }) {
  const navigate = useNavigate()

  const handleLogout = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    navigate('/login', { replace: true })
  }

  return (
    <main className="min-h-screen">
      <div className="mx-auto max-w-6xl space-y-6 p-6">
        <nav className="flex items-center justify-between rounded-xl bg-white p-6 shadow">
          <h1 className="text-2xl font-semibold text-slate-900">AI Teaching Assistant</h1>
          <button
            onClick={handleLogout}
            className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600"
          >
            Logout
          </button>
        </nav>

        <section className="rounded-xl bg-white p-6 shadow">{children}</section>
      </div>
    </main>
  )
}

export default Layout
