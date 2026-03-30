import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import Layout from '../components/Layout'
import api from '../services/api'

function CourseDetailPage() {
  const { id } = useParams()
  const [assignments, setAssignments] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [title, setTitle] = useState('')
  const [question, setQuestion] = useState('')
  const [answerKey, setAnswerKey] = useState('')
  const [contentByAssignment, setContentByAssignment] = useState({})
  const [fileByAssignment, setFileByAssignment] = useState({})
  const [fileSubmittingByAssignment, setFileSubmittingByAssignment] = useState({})
  const [resultByAssignment, setResultByAssignment] = useState({})
  const [submissionsByAssignment, setSubmissionsByAssignment] = useState({})
  const [submissionLoadingByAssignment, setSubmissionLoadingByAssignment] = useState({})
  const [submissionErrorByAssignment, setSubmissionErrorByAssignment] = useState({})
  const [overrideFormByEvaluation, setOverrideFormByEvaluation] = useState({})
  const [overrideLoadingByEvaluation, setOverrideLoadingByEvaluation] = useState({})
  const [overrideMessageByAssignment, setOverrideMessageByAssignment] = useState({})
  const [actionMessage, setActionMessage] = useState('')

  const user = useMemo(() => {
    try {
      const raw = localStorage.getItem('user')
      return raw ? JSON.parse(raw) : null
    } catch {
      return null
    }
  }, [])

  const loadAssignments = async () => {
    setLoading(true)
    setError('')

    try {
      const response = await api.get(`/courses/${id}/assignments`)
      setAssignments(Array.isArray(response.data) ? response.data : [])
    } catch (err) {
      const message = err?.response?.data?.error || 'Failed to fetch assignments'
      setError(message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadAssignments()
  }, [id])

  const handleCreateAssignment = async (event) => {
    event.preventDefault()
    const trimmedTitle = title.trim()
    const trimmedQuestion = question.trim()
    const trimmedAnswerKey = answerKey.trim()

    if (!trimmedTitle || !trimmedQuestion || !trimmedAnswerKey) {
      setActionMessage('Title, question, and answer key are required')
      return
    }

    try {
      setActionMessage('')
      await api.post('/assignments', {
        course_id: Number(id),
        title: trimmedTitle,
        question: trimmedQuestion,
        answer_key: trimmedAnswerKey,
      })
      setTitle('')
      setQuestion('')
      setAnswerKey('')
      setActionMessage('Assignment created successfully')
      await loadAssignments()
    } catch (err) {
      const message = err?.response?.data?.error || 'Failed to create assignment'
      setActionMessage(message)
    }
  }

  const handleSubmitAnswer = async (assignmentId) => {
    const content = (contentByAssignment[assignmentId] || '').trim()
    if (!content) {
      setActionMessage('Please write your answer before submitting')
      return
    }

    try {
      setActionMessage('')
      const response = await api.post('/submit', {
        assignment_id: assignmentId,
        content,
      })

      setContentByAssignment((prev) => ({ ...prev, [assignmentId]: '' }))
      setResultByAssignment((prev) => ({
        ...prev,
        [assignmentId]: {
          marks: response?.data?.marks,
          feedback: response?.data?.feedback,
          confidence: response?.data?.confidence,
        },
      }))
      setActionMessage('Answer submitted successfully')
    } catch (err) {
      const message = err?.response?.data?.error || 'Failed to submit answer'
      setActionMessage(message)
    }
  }

  const handleFileSelect = (assignmentId, file) => {
    setFileByAssignment((prev) => ({
      ...prev,
      [assignmentId]: file || null,
    }))
  }

  const handleSubmitFile = async (assignmentId) => {
    const file = fileByAssignment[assignmentId]
    if (!file) {
      setActionMessage('Please choose a PDF file before uploading')
      return
    }

    const lowerName = String(file.name || '').toLowerCase()
    if (!lowerName.endsWith('.pdf')) {
      setActionMessage('Only PDF files are allowed')
      return
    }

    const formData = new FormData()
    formData.append('assignment_id', String(assignmentId))
    formData.append('file', file)

    setFileSubmittingByAssignment((prev) => ({ ...prev, [assignmentId]: true }))

    try {
      setActionMessage('')
      const response = await api.post('/submit-file', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      })

      setFileByAssignment((prev) => ({ ...prev, [assignmentId]: null }))
      setResultByAssignment((prev) => ({
        ...prev,
        [assignmentId]: {
          marks: response?.data?.marks,
          feedback: response?.data?.feedback,
          confidence: null,
        },
      }))
      setActionMessage('PDF uploaded and evaluated successfully')
    } catch (err) {
      const message = err?.response?.data?.error || 'Failed to upload and evaluate file'
      setActionMessage(message)
    } finally {
      setFileSubmittingByAssignment((prev) => ({ ...prev, [assignmentId]: false }))
    }
  }

  const loadSubmissionsForAssignment = async (assignmentId) => {
    setSubmissionLoadingByAssignment((prev) => ({ ...prev, [assignmentId]: true }))
    setSubmissionErrorByAssignment((prev) => ({ ...prev, [assignmentId]: '' }))
    setOverrideMessageByAssignment((prev) => ({ ...prev, [assignmentId]: '' }))

    try {
      const response = await api.get(`/assignments/${assignmentId}/submissions`)
      const rows = Array.isArray(response.data) ? response.data : []
      setSubmissionsByAssignment((prev) => ({ ...prev, [assignmentId]: rows }))
    } catch (err) {
      const message = err?.response?.data?.error || 'Failed to fetch submissions'
      setSubmissionErrorByAssignment((prev) => ({ ...prev, [assignmentId]: message }))
    } finally {
      setSubmissionLoadingByAssignment((prev) => ({ ...prev, [assignmentId]: false }))
    }
  }

  const handleOverrideChange = (evaluationId, field, value) => {
    setOverrideFormByEvaluation((prev) => ({
      ...prev,
      [evaluationId]: {
        ...prev[evaluationId],
        [field]: value,
      },
    }))
  }

  const handleOverrideSubmit = async (assignmentId, evaluationId, defaultMarks, defaultFeedback) => {
    const rawMarks = overrideFormByEvaluation[evaluationId]?.marks
    const marks = rawMarks === undefined || rawMarks === '' ? Number(defaultMarks) : Number(rawMarks)
    const feedback = (
      overrideFormByEvaluation[evaluationId]?.feedback !== undefined
        ? overrideFormByEvaluation[evaluationId].feedback
        : defaultFeedback
    )?.trim()

    if (Number.isNaN(marks)) {
      setOverrideMessageByAssignment((prev) => ({ ...prev, [assignmentId]: 'Marks must be a valid number' }))
      return
    }

    if (!feedback) {
      setOverrideMessageByAssignment((prev) => ({ ...prev, [assignmentId]: 'Feedback is required' }))
      return
    }

    setOverrideLoadingByEvaluation((prev) => ({ ...prev, [evaluationId]: true }))
    setOverrideMessageByAssignment((prev) => ({ ...prev, [assignmentId]: '' }))

    try {
      await api.put(`/evaluations/${evaluationId}`, {
        marks,
        feedback,
      })

      // Immediate UI sync so professor sees updated values without waiting.
      setSubmissionsByAssignment((prev) => {
        const currentRows = prev[assignmentId]
        if (!Array.isArray(currentRows)) {
          return prev
        }

        const updatedRows = currentRows.map((row) => {
          const rowEvaluationID = row?.evaluation?.id
          if (rowEvaluationID !== evaluationId) {
            return row
          }

          return {
            ...row,
            evaluation: {
              ...row.evaluation,
              marks,
              feedback,
            },
          }
        })

        return {
          ...prev,
          [assignmentId]: updatedRows,
        }
      })

      setOverrideFormByEvaluation((prev) => ({
        ...prev,
        [evaluationId]: {
          marks,
          feedback,
        },
      }))

      setOverrideMessageByAssignment((prev) => ({ ...prev, [assignmentId]: 'Evaluation updated successfully' }))
      await loadSubmissionsForAssignment(assignmentId)
    } catch (err) {
      const message = err?.response?.data?.error || 'Failed to update evaluation'
      setOverrideMessageByAssignment((prev) => ({ ...prev, [assignmentId]: message }))
    } finally {
      setOverrideLoadingByEvaluation((prev) => ({ ...prev, [evaluationId]: false }))
    }
  }

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">Course Detail</h1>
            <p className="mt-1 text-sm text-gray-600">Course ID: {id}</p>
          </div>
          <Link to="/dashboard" className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600">
            Back to Dashboard
          </Link>
        </div>

        {user?.role === 'professor' ? (
          <form onSubmit={handleCreateAssignment} className="mt-6 rounded-lg bg-slate-50 p-4 ring-1 ring-slate-200">
            <h2 className="text-lg font-semibold text-slate-900">Create Assignment</h2>
            <div className="mt-3 space-y-4">
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Assignment title"
                className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              <textarea
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                placeholder="Question visible to students"
                rows={3}
                className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              <textarea
                value={answerKey}
                onChange={(e) => setAnswerKey(e.target.value)}
                placeholder="Answer key"
                rows={4}
                className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              <div>
                <button type="submit" className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600">
                  Create Assignment
                </button>
              </div>
            </div>
          </form>
        ) : null}

        <section className="rounded-xl bg-white p-5 shadow">
          <h2 className="text-lg font-medium text-slate-900">Assignments</h2>
          {loading ? <p className="mt-2 text-sm text-slate-500">Loading assignments...</p> : null}
          {error ? <p className="mt-2 text-sm text-red-600">{error}</p> : null}

          {!loading && !error && assignments.length === 0 ? (
            <p className="mt-2 text-sm text-slate-500">No assignments available.</p>
          ) : null}

          {!loading && !error && assignments.length > 0 ? (
            <div className="mt-3 space-y-6">
              {assignments.map((assignment) => (
                <article key={assignment.id} className="rounded-2xl bg-white p-4 shadow-md transition hover:shadow-lg">
                  {user?.role === 'student' ? (
                    <>
                      {assignment.question ? (
                        <p className="mb-2 text-lg font-medium text-slate-900">{assignment.question}</p>
                      ) : null}
                      <p className="text-sm text-gray-600">Title: {assignment.title}</p>
                    </>
                  ) : (
                    <>
                      <h3 className="text-lg font-medium text-slate-900">{assignment.title}</h3>
                      {assignment.question ? (
                        <p className="mt-1 text-sm text-gray-600">{assignment.question}</p>
                      ) : null}
                    </>
                  )}

                  {user?.role === 'student' ? (
                    <div className="mt-3 space-y-4">
                      <textarea
                        rows={3}
                        value={contentByAssignment[assignment.id] || ''}
                        onChange={(e) =>
                          setContentByAssignment((prev) => ({
                            ...prev,
                            [assignment.id]: e.target.value,
                          }))
                        }
                        placeholder="Write your answer"
                        className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                      />
                      <div>
                        <button
                          onClick={() => handleSubmitAnswer(assignment.id)}
                          className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600"
                        >
                          Submit Answer
                        </button>
                      </div>

                      <div className="space-y-3 rounded-lg border border-slate-200 bg-slate-50 p-3">
                        <p className="text-sm font-medium text-slate-900">Upload PDF Submission</p>
                        <input
                          type="file"
                          accept="application/pdf,.pdf"
                          onChange={(e) => handleFileSelect(assignment.id, e.target.files?.[0])}
                          className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                        />
                        <button
                          onClick={() => handleSubmitFile(assignment.id)}
                          disabled={fileSubmittingByAssignment[assignment.id]}
                          className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600 disabled:opacity-60"
                        >
                          {fileSubmittingByAssignment[assignment.id] ? 'Uploading...' : 'Upload PDF'}
                        </button>
                      </div>

                      {resultByAssignment[assignment.id] ? (
                        <div className="mt-4 rounded-xl border border-green-200 bg-green-50 p-4">
                          <p className="text-xl font-bold text-green-900">
                            Marks: {resultByAssignment[assignment.id].marks ?? '-'}
                          </p>
                          <p className="mt-2 text-sm text-green-900">
                            <span className="font-semibold">Feedback:</span>{' '}
                            {resultByAssignment[assignment.id].feedback || '-'}
                          </p>
                          <p className="mt-2 text-xs text-green-800">
                            <span className="font-semibold">Confidence:</span>{' '}
                            {resultByAssignment[assignment.id].confidence ?? '-'}
                          </p>
                        </div>
                      ) : null}
                    </div>
                  ) : null}

                  {user?.role === 'professor' ? (
                    <div className="mt-3 rounded-lg bg-slate-50 p-3 ring-1 ring-slate-200">
                      <div className="flex items-center justify-between gap-2">
                        <p className="text-lg font-medium text-slate-900">Professor Review Panel</p>
                        <button
                          onClick={() => loadSubmissionsForAssignment(assignment.id)}
                          className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600"
                        >
                          {submissionLoadingByAssignment[assignment.id] ? 'Loading...' : 'Load Submissions'}
                        </button>
                      </div>

                      {submissionErrorByAssignment[assignment.id] ? (
                        <p className="mt-2 text-sm text-red-600">{submissionErrorByAssignment[assignment.id]}</p>
                      ) : null}

                      {overrideMessageByAssignment[assignment.id] ? (
                        <p className="mt-2 text-sm text-slate-700">{overrideMessageByAssignment[assignment.id]}</p>
                      ) : null}

                      {Array.isArray(submissionsByAssignment[assignment.id]) &&
                      submissionsByAssignment[assignment.id].length > 0 ? (
                        <div className="mt-3">
                          {submissionsByAssignment[assignment.id].map((submission) => {
                            const evaluation = submission.evaluation || {}
                            const evaluationId = evaluation.id
                            return (
                              <div key={submission.submission_id} className="mb-4 rounded-xl border bg-gray-50 p-4">
                                <p className="text-sm text-gray-600">Submission #{submission.submission_id}</p>
                                <p className="text-sm text-gray-600">Student ID: {submission.student_id}</p>
                                <div className="my-2 border-t" />
                                <p className="text-sm font-semibold text-slate-900">Student Answer</p>
                                <p className="mt-1 text-sm text-gray-600">{submission.content}</p>
                                <div className="my-2 border-t" />
                                <div className="rounded-lg border border-blue-200 bg-blue-50 p-3">
                                  <p className="text-lg font-medium text-slate-900">AI Evaluation</p>
                                  <p className="mt-1 text-sm text-gray-600">
                                    <span className="font-semibold text-slate-900">Marks:</span> {evaluation.marks ?? '-'}
                                  </p>
                                  <p className="mt-1 text-sm text-gray-600">
                                    <span className="font-semibold text-slate-900">Feedback:</span>{' '}
                                    {evaluation.feedback || '-'}
                                  </p>
                                  <p className="mt-1 text-sm text-gray-600">
                                    <span className="font-semibold text-slate-900">Confidence:</span>{' '}
                                    {evaluation.confidence ?? '-'}
                                  </p>
                                </div>

                                {evaluationId ? (
                                  <div className="mt-3 space-y-4">
                                    <div className="my-2 border-t" />
                                    <input
                                      type="number"
                                      step="0.1"
                                      value={overrideFormByEvaluation[evaluationId]?.marks ?? evaluation.marks ?? ''}
                                      onChange={(e) => handleOverrideChange(evaluationId, 'marks', e.target.value)}
                                      placeholder="Override marks"
                                      className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                                    />
                                    <textarea
                                      rows={3}
                                      value={overrideFormByEvaluation[evaluationId]?.feedback ?? evaluation.feedback ?? ''}
                                      onChange={(e) => handleOverrideChange(evaluationId, 'feedback', e.target.value)}
                                      placeholder="Override feedback"
                                      className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                                    />
                                    <div>
                                      <button
                                        onClick={() =>
                                          handleOverrideSubmit(
                                            assignment.id,
                                            evaluationId,
                                            evaluation.marks,
                                            evaluation.feedback,
                                          )
                                        }
                                        disabled={overrideLoadingByEvaluation[evaluationId]}
                                        className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600 disabled:opacity-60"
                                      >
                                        {overrideLoadingByEvaluation[evaluationId] ? 'Saving...' : 'Save Override'}
                                      </button>
                                    </div>
                                  </div>
                                ) : (
                                  <p className="mt-2 text-sm text-slate-500">Evaluation not available for this submission.</p>
                                )}
                              </div>
                            )
                          })}
                        </div>
                      ) : null}

                      {Array.isArray(submissionsByAssignment[assignment.id]) &&
                      submissionsByAssignment[assignment.id].length === 0 &&
                      !submissionLoadingByAssignment[assignment.id] &&
                      !submissionErrorByAssignment[assignment.id] ? (
                        <p className="mt-2 text-sm text-slate-500">No submissions yet for this assignment.</p>
                      ) : null}
                    </div>
                  ) : null}
                </article>
              ))}
            </div>
          ) : null}
        </section>

        {actionMessage ? (
          <p className="mt-4 text-sm text-slate-700">{actionMessage}</p>
        ) : null}
      </div>
    </Layout>
  )
}

export default CourseDetailPage
