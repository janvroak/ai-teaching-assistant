import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import CourseShell from '../components/CourseShell'
import api from '../services/api'

function AssignmentDetailPage() {
  const { courseId, assignmentId } = useParams()
  const [assignment, setAssignment] = useState(null)
  const [questionPDFURL, setQuestionPDFURL] = useState('')
  const [status, setStatus] = useState('Pending')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const user = useMemo(() => {
    try {
      const raw = localStorage.getItem('user')
      return raw ? JSON.parse(raw) : null
    } catch {
      return null
    }
  }, [])
  const statusTone = status === 'Submitted'
    ? 'border-blue-500/30 bg-blue-500/10 text-blue-200'
    : 'border-amber-500/30 bg-amber-500/10 text-amber-200'

  useEffect(() => {
    const loadData = async () => {
      setLoading(true)
      setError('')
      try {
        const assignmentResponse = await api.get(`/courses/${courseId}/assignments`)
        const rows = Array.isArray(assignmentResponse.data) ? assignmentResponse.data : []
        const match = rows.find((item) => String(item.id) === String(assignmentId))
        if (!match) {
          setError('Assignment not found')
          return
        }
        setAssignment(match)

        if (match.question_type === 'pdf' && match.question_pdf_url) {
          try {
            const fileResponse = await api.get(match.question_pdf_url, { responseType: 'blob' })
            const objectURL = URL.createObjectURL(fileResponse.data)
            setQuestionPDFURL((prev) => {
              if (prev) URL.revokeObjectURL(prev)
              return objectURL
            })
          } catch {
            setQuestionPDFURL('')
          }
        } else {
          setQuestionPDFURL('')
        }

        if (user?.role === 'student') {
          try {
            const mySubmissions = await api.get('/my-submissions')
            const rowsSub = Array.isArray(mySubmissions.data) ? mySubmissions.data : []
            const existing = rowsSub.find((item) => String(item.assignment_id) === String(assignmentId))
            setStatus(existing ? 'Submitted' : 'Pending')
          } catch {
            setStatus('Pending')
          }
        }
      } catch (err) {
        setError(err?.response?.data?.error || 'Failed to load assignment details')
      } finally {
        setLoading(false)
      }
    }

    loadData()
  }, [courseId, assignmentId, user?.role])

  useEffect(() => () => {
    if (questionPDFURL) URL.revokeObjectURL(questionPDFURL)
  }, [questionPDFURL])

  return (
    <CourseShell
      courseId={courseId}
      activeTab="assignments"
      title={assignment?.title || `Assignment ${assignmentId}`}
      description="Review assignment content and metadata before opening the submission workflow."
    >
      <section className="space-y-6">
        <div className="flex flex-wrap items-center gap-3">
          <Link
            to={`/courses/${courseId}/assignments`}
            className="rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm text-slate-300 transition-all duration-300 hover:border-blue-500/30 hover:bg-white/10"
          >
            Back to Assignments
          </Link>
          <Link
            to={`/courses/${courseId}/assignments/${assignmentId}/submission`}
            className="btn-primary inline-flex w-auto items-center"
          >
            Open Submission
          </Link>
        </div>

        {loading ? <p className="text-sm text-slate-400">Loading assignment details...</p> : null}
        {error ? <p className="text-sm text-red-300">{error}</p> : null}

        {!loading && !error && assignment ? (
          <>
            <div className="app-card p-6">
              <h2 className="text-lg font-semibold text-slate-100">Assignment Content</h2>
              {assignment.question_type === 'pdf' ? (
                <div className="mt-4 max-h-[70vh] overflow-hidden rounded-xl border border-white/10">
                  {questionPDFURL ? (
                    <iframe
                      src={questionPDFURL}
                      title={`Question PDF ${assignment.id}`}
                      className="h-[70vh] w-full"
                    />
                  ) : (
                    <div className="flex h-56 items-center justify-center text-sm text-slate-400">
                      Loading PDF...
                    </div>
                  )}
                </div>
              ) : (
                <p className="mt-4 whitespace-pre-wrap text-sm text-slate-300">{assignment.question || '-'}</p>
              )}
            </div>

            <div className="grid gap-4 md:grid-cols-3">
              <div className="app-card p-5">
                <p className="text-xs uppercase tracking-wide text-slate-400">Due Date</p>
                <p className="mt-2 text-sm text-slate-100">
                  {assignment.due_date ? new Date(assignment.due_date).toLocaleString() : 'Not set'}
                </p>
              </div>
              <div className="app-card p-5">
                <p className="text-xs uppercase tracking-wide text-slate-400">Max Marks</p>
                <p className="mt-2 text-sm text-slate-100">{assignment.max_marks ?? 'Not specified'}</p>
              </div>
              <div className="app-card p-5">
                <p className="text-xs uppercase tracking-wide text-slate-400">Status</p>
                <p className="mt-2">
                  <span className={`status-chip ${statusTone}`}>
                    <span className={`status-dot ${status === 'Submitted' ? 'bg-blue-300' : 'bg-amber-300'}`} />
                    {status}
                  </span>
                </p>
              </div>
            </div>
          </>
        ) : null}
      </section>
    </CourseShell>
  )
}

export default AssignmentDetailPage
