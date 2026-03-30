import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import Layout from '../components/Layout'
import api from '../services/api'

function DashboardPage() {
  const [courses, setCourses] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [mySubmissions, setMySubmissions] = useState([])
  const [mySubmissionsLoading, setMySubmissionsLoading] = useState(false)
  const [mySubmissionsError, setMySubmissionsError] = useState('')
  const [courseName, setCourseName] = useState('')
  const [courseCode, setCourseCode] = useState('')
  const [actionMessage, setActionMessage] = useState('')

  const user = useMemo(() => {
    try {
      const raw = localStorage.getItem('user')
      return raw ? JSON.parse(raw) : null
    } catch {
      return null
    }
  }, [])

  const loadCourses = async () => {
    setError('')
    setLoading(true)

    try {
      const response = await api.get('/courses')
      setCourses(Array.isArray(response.data) ? response.data : [])
    } catch (err) {
      const message = err?.response?.data?.error || 'Failed to fetch courses'
      setError(message)
    } finally {
      setLoading(false)
    }
  }

  const loadMySubmissions = async () => {
    setMySubmissionsLoading(true)
    setMySubmissionsError('')

    try {
      const response = await api.get('/my-submissions')
      setMySubmissions(Array.isArray(response.data) ? response.data : [])
    } catch (err) {
      const message = err?.response?.data?.error || 'Failed to fetch your submissions'
      setMySubmissionsError(message)
    } finally {
      setMySubmissionsLoading(false)
    }
  }

  useEffect(() => {
    loadCourses()
    if (user?.role === 'student') {
      loadMySubmissions()
    }
  }, [user?.role])

  const handleCreateCourse = async () => {
    const trimmedName = courseName.trim()
    if (!trimmedName) {
      setActionMessage('Enter a course name')
      return
    }

    try {
      setActionMessage('')
      await api.post('/courses', { name: trimmedName })
      setCourseName('')
      setActionMessage('Course created successfully')
      await loadCourses()
    } catch (err) {
      const message = err?.response?.data?.error || 'Failed to create course'
      setActionMessage(message)
    }
  }

  const handleJoinCourse = async () => {
    const trimmedCode = courseCode.trim().toUpperCase()
    if (!trimmedCode) {
      setActionMessage('Enter a course code')
      return
    }

    try {
      setActionMessage('')
      await api.post('/courses/join', { course_code: trimmedCode })
      setCourseCode('')
      setActionMessage('Joined course successfully')
      await loadCourses()
    } catch (err) {
      const message = err?.response?.data?.error || 'Failed to join course'
      setActionMessage(message)
    }
  }

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">Dashboard</h1>
            <p className="mt-1 text-sm text-gray-600">You are logged in.</p>
          </div>
        </div>

        <div className="rounded-lg bg-slate-50 p-4 ring-1 ring-slate-200">
          <p className="text-sm text-gray-600">
            Logged in as <span className="font-medium text-slate-900">{user?.role || 'unknown'}</span>
          </p>

          {user?.role === 'professor' ? (
            <div className="mt-3 flex flex-col gap-2 sm:flex-row">
              <input
                type="text"
                value={courseName}
                onChange={(e) => setCourseName(e.target.value)}
                placeholder="Course name"
                className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              <button
                onClick={handleCreateCourse}
                className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600"
              >
                Create Course
              </button>
            </div>
          ) : null}

          {user?.role === 'student' ? (
            <div className="mt-3 flex flex-col gap-2 sm:flex-row">
              <input
                type="text"
                value={courseCode}
                onChange={(e) => setCourseCode(e.target.value)}
                placeholder="Course code (e.g., ABC123)"
                className="w-full rounded border px-3 py-2 text-sm uppercase focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              <button
                onClick={handleJoinCourse}
                className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600"
              >
                Join Course
              </button>
            </div>
          ) : null}

          {actionMessage ? (
            <p className="mt-2 text-sm text-slate-700">{actionMessage}</p>
          ) : null}
        </div>

        <section>
          <h2 className="mb-4 text-lg font-medium text-slate-900">Courses</h2>

          {loading ? <p className="mt-2 text-sm text-slate-500">Loading courses...</p> : null}
          {error ? <p className="mt-2 text-sm text-red-600">{error}</p> : null}

          {!loading && !error && courses.length === 0 ? (
            <p className="mt-2 text-sm text-slate-500">No courses found.</p>
          ) : null}

          {!loading && !error && courses.length > 0 ? (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              {courses.map((course) => (
                <Link key={course.id} to={`/courses/${course.id}`}>
                  <article className="rounded-2xl bg-white p-4 shadow-md transition hover:shadow-lg">
                    <h3 className="text-base font-semibold text-slate-900">{course.name}</h3>
                    <p className="mt-1 text-sm text-gray-600">Code: {course.course_code}</p>
                  </article>
                </Link>
              ))}
            </div>
          ) : null}
        </section>

        {user?.role === 'student' ? (
          <section>
            <h2 className="mb-4 text-lg font-medium text-slate-900">Your Submissions</h2>

            {mySubmissionsLoading ? <p className="mt-2 text-sm text-slate-500">Loading submissions...</p> : null}
            {mySubmissionsError ? <p className="mt-2 text-sm text-red-600">{mySubmissionsError}</p> : null}

            {!mySubmissionsLoading && !mySubmissionsError && mySubmissions.length === 0 ? (
              <p className="mt-2 text-sm text-slate-500">No submissions yet.</p>
            ) : null}

            {!mySubmissionsLoading && !mySubmissionsError && mySubmissions.length > 0 ? (
              <div className="space-y-4">
                {mySubmissions.map((submission) => (
                  <article key={submission.submission_id} className="mb-4 rounded bg-white p-4 shadow">
                    <p className="text-sm text-gray-600">Assignment title</p>
                    <p className="text-lg font-medium text-slate-900">
                      {submission.assignment_title || `Assignment #${submission.assignment_id}`}
                    </p>

                    <div className="mt-3">
                      <p className="text-sm text-gray-600">Your Answer</p>
                      <p className="text-sm text-slate-900">{submission.content}</p>
                    </div>

                    <div className="mt-3 rounded-lg bg-slate-50 p-3">
                      <p className="text-sm text-gray-600">AI Evaluation</p>
                      <p className="mt-1 text-sm text-gray-700">
                        <span className="font-medium text-slate-900">Marks:</span> {submission?.evaluation?.marks ?? '-'}
                      </p>
                      <p className="mt-1 text-sm text-gray-700">
                        <span className="font-medium text-slate-900">Feedback:</span> {submission?.evaluation?.feedback || '-'}
                      </p>
                      <p className="mt-1 text-sm text-gray-700">
                        <span className="font-medium text-slate-900">Confidence:</span>{' '}
                        {submission?.evaluation?.confidence ?? '-'}
                      </p>
                    </div>
                  </article>
                ))}
              </div>
            ) : null}
          </section>
        ) : null}

        <div className="rounded-lg bg-slate-50 p-4 ring-1 ring-slate-200">
          <p className="text-sm text-gray-600">Current user payload</p>
          <pre className="mt-2 overflow-auto text-xs text-slate-800">
            {JSON.stringify(user, null, 2)}
          </pre>
        </div>
      </div>
    </Layout>
  )
}

export default DashboardPage
