import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import Layout from '../components/Layout'
import api from '../services/api'

function DashboardPage() {
  const [courses, setCourses] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [courseName, setCourseName] = useState('')
  const [courseCode, setCourseCode] = useState('')
  const [message, setMessage] = useState('')
  const [messageType, setMessageType] = useState('info')
  const [deletingCourseID, setDeletingCourseID] = useState(null)

  const user = useMemo(() => {
    try {
      const raw = localStorage.getItem('user')
      return raw ? JSON.parse(raw) : null
    } catch {
      return null
    }
  }, [])

  const isProfessor = user?.role === 'professor'

  const showMessage = (text, type = 'info') => {
    setMessage(text)
    setMessageType(type)
  }

  const loadCourses = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await api.get('/courses')
      setCourses(Array.isArray(response.data) ? response.data : [])
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to fetch courses')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadCourses()
  }, [user?.role])

  const handleCreateCourse = async () => {
    const trimmedName = courseName.trim()
    if (!trimmedName) return showMessage('Enter a course name', 'error')

    try {
      await api.post('/courses', { name: trimmedName })
      setCourseName('')
      showMessage('Course created successfully', 'success')
      await loadCourses()
    } catch (err) {
      showMessage(err?.response?.data?.error || 'Failed to create course', 'error')
    }
  }

  const handleJoinCourse = async () => {
    const trimmedCode = courseCode.trim().toUpperCase()
    if (!trimmedCode) return showMessage('Enter a course code', 'error')

    try {
      await api.post('/courses/join', { course_code: trimmedCode })
      setCourseCode('')
      showMessage('Joined course successfully', 'success')
      await loadCourses()
    } catch (err) {
      showMessage(err?.response?.data?.error || 'Failed to join course', 'error')
    }
  }

  const handleDeleteCourse = async (course) => {
    const confirmed = window.confirm(
      `Delete "${course.name}"? This removes assignments, submissions, evaluations, and enrollments.`,
    )
    if (!confirmed) return

    try {
      setDeletingCourseID(course.id)
      await api.delete(`/courses/${course.id}`)
      showMessage('Course deleted successfully', 'success')
      await loadCourses()
    } catch (err) {
      showMessage(err?.response?.data?.error || 'Failed to delete course', 'error')
    } finally {
      setDeletingCourseID(null)
    }
  }

  return (
    <Layout>
      <div className="space-y-6 p-2 md:p-4">
        <section className="app-card relative overflow-hidden p-6">
          <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-cyan-300/80 to-transparent" />
          <h1 className="text-2xl font-semibold text-slate-100">Dashboard</h1>
          <p className="mt-2 text-sm text-slate-300">
            Welcome back {user?.email ? `(${user.email})` : ''}. Manage your courses from one clean workspace.
          </p>
        </section>

        <section className="app-card space-y-4 p-6">
          <div>
            <h2 className="text-lg font-semibold text-slate-100">
              {isProfessor ? 'Create a Course' : 'Join a Course'}
            </h2>
            <p className="mt-1 text-sm text-slate-400">
              {isProfessor
                ? 'Create a new course and share its code with students.'
                : 'Enter your course code to join quickly.'}
            </p>
          </div>

          {isProfessor ? (
            <div className="flex flex-col gap-3 md:flex-row">
              <input
                type="text"
                value={courseName}
                onChange={(event) => setCourseName(event.target.value)}
                placeholder="Course name"
                className="input-field"
              />
              <button type="button" onClick={handleCreateCourse} className="btn-primary md:w-auto">
                Create Course
              </button>
            </div>
          ) : (
            <div className="flex flex-col gap-3 md:flex-row">
              <input
                type="text"
                value={courseCode}
                onChange={(event) => setCourseCode(event.target.value)}
                placeholder="Course code"
                className="input-field uppercase"
              />
              <button type="button" onClick={handleJoinCourse} className="btn-primary md:w-auto">
                Join Course
              </button>
            </div>
          )}

          {message ? (
            <div
              className={`rounded-xl border p-3 text-sm ${
                messageType === 'success'
                  ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-300'
                  : messageType === 'error'
                    ? 'border-red-500/20 bg-red-500/10 text-red-300'
                    : 'border-blue-500/20 bg-blue-500/10 text-slate-100'
              }`}
            >
              {message}
            </div>
          ) : null}
        </section>

        <section className="space-y-4">
          <div>
            <h2 className="section-title">Courses</h2>
            <p className="section-subtitle">Open a course to navigate assignments and analytics.</p>
          </div>

          {loading ? <p className="text-sm text-slate-400">Loading courses...</p> : null}
          {error ? <p className="text-sm text-red-300">{error}</p> : null}

          {!loading && !error && courses.length === 0 ? (
            <div className="app-card p-6">
              <p className="text-sm text-slate-300">
                No courses yet. {isProfessor ? 'Create your first course above.' : 'Join a course with code.'}
              </p>
            </div>
          ) : null}

          {!loading && !error && courses.length > 0 ? (
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {courses.map((course) => (
                <article
                  key={course.id}
                  className="app-card app-card-hover p-5"
                >
                  <Link to={`/courses/${course.id}`} className="block">
                    <div className="flex items-start justify-between gap-3">
                      <h3 className="text-base font-semibold text-slate-100">{course.name}</h3>
                      <span className="rounded-md border border-cyan-400/30 bg-cyan-400/10 px-2 py-0.5 text-[11px] font-medium uppercase tracking-wide text-cyan-200">
                        {isProfessor ? 'Owned' : 'Joined'}
                      </span>
                    </div>
                    <p className="mt-3 inline-flex items-center rounded-md border border-white/10 bg-white/5 px-2 py-1 text-xs tracking-wide text-slate-300">
                      Course Code: {course.course_code}
                    </p>
                  </Link>
                  {isProfessor ? (
                    <div className="mt-4 flex justify-end">
                      <button
                        type="button"
                        onClick={() => handleDeleteCourse(course)}
                        disabled={deletingCourseID === course.id}
                        className="rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-1.5 text-xs text-red-300 transition-all duration-300 hover:bg-red-500/20 disabled:opacity-60"
                      >
                        {deletingCourseID === course.id ? 'Deleting...' : 'Delete'}
                      </button>
                    </div>
                  ) : null}
                </article>
              ))}
            </div>
          ) : null}
        </section>
      </div>
    </Layout>
  )
}

export default DashboardPage
