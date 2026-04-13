import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import CourseShell from '../components/CourseShell'
import api from '../services/api'

function CourseAssignmentsPage() {
  const { courseId } = useParams()
  const [assignments, setAssignments] = useState([])
  const [submissionByAssignmentId, setSubmissionByAssignmentId] = useState({})
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

  const isStudent = user?.role === 'student'

  useEffect(() => {
    const loadAssignments = async () => {
      setLoading(true)
      setError('')
      try {
        const response = await api.get(`/courses/${courseId}/assignments`)
        setAssignments(Array.isArray(response.data) ? response.data : [])
        if (isStudent) {
          const submissionResponse = await api.get('/my-submissions')
          const rows = Array.isArray(submissionResponse.data) ? submissionResponse.data : []
          const map = {}
          rows.forEach((row) => {
            if (String(row.assignment_id) in map) return
            map[String(row.assignment_id)] = row
          })
          setSubmissionByAssignmentId(map)
        } else {
          setSubmissionByAssignmentId({})
        }
      } catch (err) {
        setError(err?.response?.data?.error || 'Failed to fetch assignments')
      } finally {
        setLoading(false)
      }
    }

    loadAssignments()
  }, [courseId, isStudent])

  const statusMeta = (assignmentId) => {
    const existing = submissionByAssignmentId[String(assignmentId)]
    if (!existing) {
      return {
        label: 'Not Submitted',
        tone: 'border-amber-500/30 bg-amber-500/10 text-amber-200',
        dot: 'bg-amber-300',
      }
    }
    const isFinal = Boolean(existing?.evaluation?.is_final)
    if (isFinal) {
      return {
        label: 'Evaluated',
        tone: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-200',
        dot: 'bg-emerald-300',
      }
    }
    return {
      label: 'Submitted',
      tone: 'border-blue-500/30 bg-blue-500/10 text-blue-200',
      dot: 'bg-blue-300',
    }
  }

  return (
    <CourseShell
      courseId={courseId}
      activeTab="assignments"
      title="Assignments"
      description="Select an assignment to open a dedicated detail page."
    >
      <section className="space-y-6">
        <header>
          <h2 className="section-title">All Assignments</h2>
          <p className="section-subtitle">
            Click any assignment card to navigate to its detail page.
          </p>
        </header>

        {loading ? <p className="text-sm text-slate-400">Loading assignments...</p> : null}
        {error ? <p className="text-sm text-red-300">{error}</p> : null}

        {!loading && !error && assignments.length === 0 ? (
          <div className="app-card p-6">
            <p className="text-sm text-slate-300">No assignments available for this course yet.</p>
          </div>
        ) : null}

        {!loading && !error && assignments.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2">
            {assignments.map((assignment) => (
              <Link
                key={assignment.id}
                to={`/courses/${courseId}/assignments/${assignment.id}`}
                className="app-card app-card-hover p-5"
              >
                <h3 className="text-base font-semibold text-slate-100">{assignment.title}</h3>
                <p className="mt-2 text-sm text-slate-300">
                  {assignment.question_type === 'pdf'
                    ? 'Question format: PDF'
                    : 'Question format: Text'}
                </p>
                {isStudent ? (
                  <div className="mt-3">
                    <span
                      className={`status-chip ${statusMeta(assignment.id).tone}`}
                    >
                      <span className={`status-dot ${statusMeta(assignment.id).dot}`} />
                      {statusMeta(assignment.id).label}
                    </span>
                  </div>
                ) : null}
                <p className="mt-1 text-xs text-slate-400">Assignment ID: {assignment.id}</p>
              </Link>
            ))}
          </div>
        ) : null}
      </section>
    </CourseShell>
  )
}

export default CourseAssignmentsPage
