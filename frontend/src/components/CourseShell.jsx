import { Link } from 'react-router-dom'
import Layout from './Layout'

function CourseShell({
  courseId,
  title,
  description,
  activeTab,
  children,
}) {
  const tabs = [
    { id: 'overview', label: 'Overview', to: `/courses/${courseId}`, icon: 'home' },
    { id: 'materials', label: 'Materials', to: `/courses/${courseId}/materials`, icon: 'stack' },
    { id: 'assignments', label: 'Assignments', to: `/courses/${courseId}/assignments`, icon: 'file' },
    { id: 'analytics', label: 'Analytics', to: `/courses/${courseId}/analytics`, icon: 'chart' },
  ]

  const tabIcon = (name) => {
    if (name === 'home') {
      return (
        <svg viewBox="0 0 24 24" className="h-3.5 w-3.5 fill-current" aria-hidden="true">
          <path d="M12 3 2 10h2v10h6v-6h4v6h6V10h2L12 3Z" />
        </svg>
      )
    }
    if (name === 'file') {
      return (
        <svg viewBox="0 0 24 24" className="h-3.5 w-3.5 fill-current" aria-hidden="true">
          <path d="M6 2h8l4 4v16H6V2Zm2 2v16h8V7h-3V4H8Zm2 7h4v2h-4v-2Z" />
        </svg>
      )
    }
    if (name === 'stack') {
      return (
        <svg viewBox="0 0 24 24" className="h-3.5 w-3.5 fill-current" aria-hidden="true">
          <path d="M12 2 2 7l10 5 10-5-10-5Zm0 7L2 4v3l10 5 10-5V4l-10 5Zm0 5L2 9v3l10 5 10-5V9l-10 5Zm0 5L2 14v3l10 5 10-5v-3l-10 5Z" />
        </svg>
      )
    }
    return (
      <svg viewBox="0 0 24 24" className="h-3.5 w-3.5 fill-current" aria-hidden="true">
        <path d="M4 19h16v2H2V3h2v16Zm3-4 3-4 3 2 4-6 2 1-5 8-3-2-2.4 3L7 15Z" />
      </svg>
    )
  }

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex items-center">
          <Link
            to="/dashboard"
            className="inline-flex items-center gap-2 rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-slate-300 transition-all duration-300 hover:border-blue-500/30 hover:bg-white/10 hover:text-slate-100"
          >
            <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current" aria-hidden="true">
              <path d="M14.7 6.3a1 1 0 0 1 0 1.4L11.41 11H20a1 1 0 1 1 0 2h-8.59l3.3 3.3a1 1 0 1 1-1.42 1.4l-5-5a1 1 0 0 1 0-1.4l5-5a1 1 0 0 1 1.42 0Z" />
            </svg>
            Back to Dashboard
          </Link>
        </div>

        <header className="app-card relative overflow-hidden p-5 md:p-6">
          <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-blue-400/70 to-transparent" />
          <div className="space-y-2">
            <p className="text-xs uppercase tracking-[0.16em] text-slate-400">Course Workspace</p>
            <h1 className="text-2xl font-semibold tracking-tight text-slate-100">{title}</h1>
            <p className="max-w-3xl text-sm text-slate-300">{description}</p>
          </div>
        </header>

        <section className="app-card p-3">
          <div className="flex flex-wrap gap-2">
            {tabs.map((tab) => (
              <Link
                key={tab.id}
                to={tab.to}
                className={`app-tab inline-flex items-center gap-1.5 ${activeTab === tab.id ? 'app-tab-active' : ''}`}
              >
                {tabIcon(tab.icon)}
                {tab.label}
              </Link>
            ))}
          </div>
        </section>

        {children}
      </div>
    </Layout>
  )
}

export default CourseShell
