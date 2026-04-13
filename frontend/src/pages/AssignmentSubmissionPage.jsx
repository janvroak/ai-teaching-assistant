import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import CourseShell from '../components/CourseShell'
import api from '../services/api'

function AssignmentSubmissionPage() {
  const { courseId, assignmentId } = useParams()
  const [assignment, setAssignment] = useState(null)
  const [answerText, setAnswerText] = useState('')
  const [selectedFile, setSelectedFile] = useState(null)
  const [submitting, setSubmitting] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [studentSubmission, setStudentSubmission] = useState(null)
  const [evaluation, setEvaluation] = useState(null)
  const [studentSubmissionFileURL, setStudentSubmissionFileURL] = useState('')
  const [professorRows, setProfessorRows] = useState([])
  const [professorLoading, setProfessorLoading] = useState(false)
  const [overrideDraftByEvaluation, setOverrideDraftByEvaluation] = useState({})
  const [professorSubmissionFileURLBySubmissionID, setProfessorSubmissionFileURLBySubmissionID] = useState({})
  const professorSubmissionFileURLRef = useRef({})

  const user = useMemo(() => {
    try {
      const raw = localStorage.getItem('user')
      return raw ? JSON.parse(raw) : null
    } catch {
      return null
    }
  }, [])

  const isStudent = user?.role === 'student'
  const isProfessor = user?.role === 'professor'
  const questionID = (questionRow) => {
    const raw = questionRow?.id ?? questionRow?.ID
    const parsed = Number(raw)
    if (!Number.isFinite(parsed) || parsed <= 0) return null
    return parsed
  }
  const questionDraftKey = (questionRow, index) => {
    const id = questionID(questionRow)
    return id ? `id-${id}` : `idx-${index}`
  }
  const questionText = (questionRow) => questionRow?.question_text ?? questionRow?.QuestionText ?? 'Question'
  const questionMarks = (questionRow) => questionRow?.marks ?? questionRow?.Marks
  const questionFeedback = (questionRow) => questionRow?.feedback ?? questionRow?.Feedback
  const questionStudentAnswer = (questionRow) => questionRow?.student_answer_text ?? questionRow?.StudentAnswerText
  const submissionStatus = (() => {
    if (!isStudent || !studentSubmission) {
      return {
        label: 'Not Submitted',
        className: 'border-amber-500/20 bg-amber-500/10 text-amber-200',
        dotClass: 'bg-amber-300',
      }
    }
    if (evaluation?.is_final) {
      return {
        label: 'Evaluated & Published',
        className: 'border-emerald-500/20 bg-emerald-500/10 text-emerald-200',
        dotClass: 'bg-emerald-300',
      }
    }
    return {
      label: 'Submitted, Awaiting Publish',
      className: 'border-blue-500/20 bg-blue-500/10 text-blue-200',
      dotClass: 'bg-blue-300',
    }
  })()

  const loadAssignment = async () => {
    const response = await api.get(`/courses/${courseId}/assignments`)
    const rows = Array.isArray(response.data) ? response.data : []
    return rows.find((item) => String(item.id) === String(assignmentId)) || null
  }

  const loadStudentState = async () => {
    if (!isStudent) return
    try {
      const response = await api.get('/my-submissions')
      const rows = Array.isArray(response.data) ? response.data : []
      const existing = rows.find((item) => String(item.assignment_id) === String(assignmentId))
      if (!existing) {
        setStudentSubmission(null)
        setEvaluation(null)
        return
      }

      setStudentSubmission(existing)
      setEvaluation(existing.evaluation || null)

      if (existing.submission_file_url || existing.submission_pdf_url) {
        try {
          const fileResponse = await api.get(existing.submission_file_url || existing.submission_pdf_url, {
            responseType: 'blob',
          })
          const nextURL = URL.createObjectURL(fileResponse.data)
          setStudentSubmissionFileURL((prev) => {
            if (prev) URL.revokeObjectURL(prev)
            return nextURL
          })
        } catch {
          setStudentSubmissionFileURL('')
        }
      } else {
        setStudentSubmissionFileURL('')
      }
    } catch {
      setStudentSubmission(null)
      setEvaluation(null)
    }
  }

  const loadProfessorRows = async () => {
    if (!isProfessor) return
    setProfessorLoading(true)
    try {
      const response = await api.get(`/assignments/${assignmentId}/submissions`)
      const rows = Array.isArray(response.data) ? response.data : []
      setProfessorRows(rows)
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to fetch submissions')
    } finally {
      setProfessorLoading(false)
    }
  }

  const initialize = async () => {
    setError('')
    try {
      const foundAssignment = await loadAssignment()
      setAssignment(foundAssignment)
      if (!foundAssignment) {
        setError('Assignment not found')
        return
      }
      await loadStudentState()
      await loadProfessorRows()
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to load submission page')
    }
  }

  useEffect(() => {
    initialize()
  }, [courseId, assignmentId, isStudent, isProfessor])

  useEffect(() => () => {
    if (studentSubmissionFileURL) URL.revokeObjectURL(studentSubmissionFileURL)
  }, [studentSubmissionFileURL])

  useEffect(() => {
    professorSubmissionFileURLRef.current = professorSubmissionFileURLBySubmissionID
  }, [professorSubmissionFileURLBySubmissionID])

  useEffect(() => () => {
    Object.values(professorSubmissionFileURLRef.current).forEach((url) => {
      if (url) URL.revokeObjectURL(url)
    })
  }, [])

  const handleSubmitText = async () => {
    if (!answerText.trim()) {
      setError('Please write your answer before submitting')
      return
    }
    setSubmitting(true)
    setError('')
    setMessage('')
    try {
      await api.post('/submit', { assignment_id: Number(assignmentId), content: answerText.trim() })
      setAnswerText('')
      setMessage('Answer submitted successfully.')
      await loadStudentState()
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to submit answer')
    } finally {
      setSubmitting(false)
    }
  }

  const handleSubmitFile = async () => {
    if (!selectedFile) {
      setError('Please choose a file before uploading')
      return
    }

    const formData = new FormData()
    formData.append('assignment_id', String(assignmentId))
    formData.append('file', selectedFile)

    setSubmitting(true)
    setError('')
    setMessage('')
    try {
      await api.post('/submit-file', formData)
      setSelectedFile(null)
      setMessage('File uploaded and evaluated successfully.')
      await loadStudentState()
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to upload file')
    } finally {
      setSubmitting(false)
    }
  }

  const handleOverride = async (row) => {
    const evaluationId = row?.evaluation?.id
    if (!evaluationId) return
    const draft = overrideDraftByEvaluation[evaluationId] || {}
    const questionRows = Array.isArray(row?.evaluation?.question_results) ? row.evaluation.question_results : []
    const questionDrafts = draft.questions || {}
    const payloadQuestions = questionRows.map((questionRow, index) => {
      const key = questionDraftKey(questionRow, index)
      const id = questionID(questionRow)
      const questionDraft = questionDrafts[key] || {}
      if (!id) {
        return null
      }
      return {
        question_id: id,
        marks: Number(questionDraft.marks ?? questionMarks(questionRow) ?? 0),
        feedback: (questionDraft.feedback ?? questionFeedback(questionRow) ?? '').trim(),
      }
    }).filter(Boolean)
    if (questionRows.length > 0 && payloadQuestions.length !== questionRows.length) {
      setError('Question-wise rows are incomplete. Please refresh submissions and try again.')
      return
    }
    const questionWiseTotal = payloadQuestions.reduce((sum, question) => sum + (Number(question.marks) || 0), 0)
    const shouldAutoComputeTotal = payloadQuestions.length > 0
    const payload = {
      marks: shouldAutoComputeTotal
        ? questionWiseTotal
        : Number(draft.marks ?? row?.evaluation?.marks ?? 0),
      feedback: (draft.feedback ?? row?.evaluation?.feedback ?? '').trim(),
      questions: payloadQuestions,
    }
    const invalidQuestionOverride = payload.questions.some(
      (question) => Number.isNaN(question.marks) || !question.feedback,
    )
    if (Number.isNaN(payload.marks) || !payload.feedback || invalidQuestionOverride) {
      setError('Override requires valid marks and feedback')
      return
    }

    setError('')
    setMessage('')
    try {
      await api.put(`/evaluations/${evaluationId}`, payload)
      setMessage('Professor override saved.')
      await loadProfessorRows()
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to save override')
    }
  }

  const handleFinalize = async (row, isFinal) => {
    const evaluationId = row?.evaluation?.id
    if (!evaluationId) return
    setError('')
    setMessage('')
    try {
      await api.put(`/evaluations/${evaluationId}/finalize`, { is_final: isFinal })
      setMessage(isFinal ? 'Evaluation finalized and published.' : 'Evaluation moved back to draft.')
      await loadProfessorRows()
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to update finalization state')
    }
  }

  const handleLoadProfessorSubmissionFile = async (row) => {
    const submissionID = row?.submission_id
    if (!submissionID) return
    if (professorSubmissionFileURLBySubmissionID[submissionID]) return
    const filePath = row?.submission_file_url || row?.submission_pdf_url
    if (!filePath) return

    try {
      const fileResponse = await api.get(filePath, { responseType: 'blob' })
      const nextURL = URL.createObjectURL(fileResponse.data)
      setProfessorSubmissionFileURLBySubmissionID((prev) => {
        if (prev[submissionID]) URL.revokeObjectURL(prev[submissionID])
        return {
          ...prev,
          [submissionID]: nextURL,
        }
      })
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to load submission file preview')
    }
  }

  return (
    <CourseShell
      courseId={courseId}
      activeTab="assignments"
      title={assignment?.title || `Assignment ${assignmentId} Submission`}
      description="Submit work, review AI evaluation, and apply professor overrides in a dedicated workflow."
    >
      <section className="space-y-6">
        <div className="flex flex-wrap items-center gap-3">
          <Link
            to={`/courses/${courseId}/assignments/${assignmentId}`}
            className="rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm text-slate-300 transition-all duration-300 hover:border-blue-500/30 hover:bg-white/10"
          >
            Back to Assignment
          </Link>
        </div>

        {message ? (
          <div className="rounded-xl border border-blue-500/20 bg-blue-500/10 p-4 text-sm text-slate-100">
            {message}
          </div>
        ) : null}
        {error ? <p className="text-sm text-red-300">{error}</p> : null}

        {isStudent ? (
          <div className="space-y-6">
            <div className={`rounded-xl border p-4 text-sm ${submissionStatus.className}`}>
              <p className="font-semibold">Submission Status</p>
              <p className="mt-1 inline-flex items-center gap-2">
                <span className={`status-dot ${submissionStatus.dotClass}`} />
                {submissionStatus.label}
              </p>
            </div>

            {!studentSubmission ? (
              <div className="grid gap-4 lg:grid-cols-2">
                <div className="app-card p-6">
                  <h2 className="text-base font-semibold text-slate-100">Text Submission</h2>
                  <textarea
                    rows={9}
                    value={answerText}
                    onChange={(event) => setAnswerText(event.target.value)}
                    placeholder="Write your answer here..."
                    className="input-field mt-4"
                  />
                  <button
                    type="button"
                    onClick={handleSubmitText}
                    disabled={submitting}
                    className="btn-primary mt-4"
                  >
                    {submitting ? 'Submitting...' : 'Submit Answer'}
                  </button>
                </div>

                <div className="app-card p-6">
                  <h2 className="text-base font-semibold text-slate-100">File Submission</h2>
                  <p className="mt-2 text-sm text-slate-400">Supported: PDF, PNG, JPG, JPEG, WEBP</p>
                  <input
                    type="file"
                    accept="application/pdf,.pdf,image/png,.png,image/jpeg,.jpg,.jpeg,image/webp,.webp"
                    onChange={(event) => setSelectedFile(event.target.files?.[0] || null)}
                    className="input-field mt-4"
                  />
                  {selectedFile ? (
                    <p className="mt-2 text-sm text-slate-300">Selected: {selectedFile.name}</p>
                  ) : null}
                  <button
                    type="button"
                    onClick={handleSubmitFile}
                    disabled={submitting}
                    className="btn-primary mt-4"
                  >
                    {submitting ? 'Uploading...' : 'Upload File'}
                  </button>
                </div>
              </div>
            ) : (
              <div className="app-card p-6">
                <h2 className="text-base font-semibold text-slate-100">Your Submission</h2>
                <p className="mt-2 text-sm text-slate-400">Resubmission is disabled for this assignment.</p>
                {studentSubmission?.submission_type === 'text' ? (
                  <p className="mt-4 whitespace-pre-wrap text-sm text-slate-300">{studentSubmission?.content || '-'}</p>
                ) : null}
                {studentSubmission?.submission_type !== 'text' && studentSubmissionFileURL ? (
                  <div className="mt-4 max-h-[70vh] overflow-hidden rounded-xl border border-white/10">
                    {studentSubmission?.submission_type === 'pdf' ? (
                      <iframe
                        src={studentSubmissionFileURL}
                        title="Submission PDF"
                        className="h-[70vh] w-full"
                      />
                    ) : (
                      <img src={studentSubmissionFileURL} alt="Submission File" className="h-[70vh] w-full object-contain" />
                    )}
                  </div>
                ) : null}
              </div>
            )}

            {evaluation ? (
              <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/10 p-6 text-emerald-300">
                {evaluation.is_final ? (
                  <div className="space-y-2">
                    <p className="text-lg font-semibold">AI Evaluation</p>
                    <p className="text-sm">Marks: {evaluation.marks ?? '-'}</p>
                    <p className="text-sm">Feedback: {evaluation.feedback || '-'}</p>
                    <p className="text-sm">Confidence: {evaluation.confidence ?? '-'}</p>
                    {Array.isArray(evaluation.question_results) && evaluation.question_results.length > 0 ? (
                      <div className="mt-4 space-y-3 rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-4">
                        <p className="text-sm font-semibold text-emerald-200">Question-wise Mark Distribution</p>
                        {evaluation.question_results.map((questionRow, index) => (
                          <div
                            key={questionID(questionRow) || `question-result-${index}`}
                            className="rounded-lg border border-emerald-500/20 bg-emerald-500/10 p-3"
                          >
                            <p className="text-sm text-emerald-100">
                              Q{index + 1}: {questionText(questionRow)}
                            </p>
                            <p className="mt-1 text-sm text-emerald-200">
                              Your Answer: {questionStudentAnswer(questionRow) || '-'}
                            </p>
                            <p className="mt-1 text-sm text-emerald-200">
                              Marks: {questionMarks(questionRow) ?? '-'}
                            </p>
                            <p className="mt-1 text-sm text-emerald-200">
                              Feedback: {questionFeedback(questionRow) || '-'}
                            </p>
                          </div>
                        ))}
                      </div>
                    ) : null}
                  </div>
                ) : (
                  <p className="text-sm">Evaluation is available but not published by professor yet.</p>
                )}
              </div>
            ) : null}
          </div>
        ) : null}

        {isProfessor ? (
          <div className="space-y-4">
            <div className="flex items-center justify-between gap-3">
              <h2 className="text-base font-semibold text-slate-100">Professor Override</h2>
              <button type="button" onClick={loadProfessorRows} className="btn-primary w-auto px-5 py-2.5">
                {professorLoading ? 'Loading...' : 'Refresh Submissions'}
              </button>
            </div>

            {professorRows.length === 0 && !professorLoading ? (
              <div className="app-card p-5">
                <p className="text-sm text-slate-300">No submissions available yet.</p>
              </div>
            ) : null}

            {professorRows.map((row) => {
              const evaluationRow = row?.evaluation || {}
              const evaluationId = evaluationRow?.id
              const draft = overrideDraftByEvaluation[evaluationId] || {}
              const questionResults = Array.isArray(evaluationRow?.question_results) ? evaluationRow.question_results : []
              const professorFileURL = professorSubmissionFileURLBySubmissionID[row?.submission_id]
              const computedQuestionWiseTotal = questionResults.reduce((sum, questionRow, index) => {
                const key = questionDraftKey(questionRow, index)
                const draftMarks = draft.questions?.[key]?.marks
                const marksValue = Number(draftMarks ?? questionMarks(questionRow) ?? 0)
                return sum + (Number.isFinite(marksValue) ? marksValue : 0)
              }, 0)
              const autoTotalEnabled = questionResults.length > 0
              return (
                <div key={row.submission_id} className="app-card space-y-4 p-5">
                  <div>
                    <p className="text-sm text-slate-400">Submission #{row.submission_id}</p>
                    <p className="text-sm text-slate-300">Student ID: {row.student_id}</p>
                  </div>

                  <div className="rounded-xl border border-white/10 bg-white/5 p-4">
                    <p className="text-sm font-semibold text-slate-100">Submitted Response</p>
                    {row?.submission_type === 'text' ? (
                      <p className="mt-2 whitespace-pre-wrap text-sm text-slate-300">{row?.content || '-'}</p>
                    ) : (
                      <div className="mt-3 space-y-3">
                        {!professorFileURL ? (
                          <button
                            type="button"
                            onClick={() => handleLoadProfessorSubmissionFile(row)}
                            className="rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm text-slate-200 transition-all duration-300 hover:border-blue-500/30 hover:bg-white/10"
                          >
                            Load Submission Preview
                          </button>
                        ) : null}

                        {professorFileURL ? (
                          row?.submission_type === 'pdf' ? (
                            <iframe
                              src={professorFileURL}
                              title={`Submission ${row.submission_id}`}
                              className="h-[55vh] w-full rounded-lg border border-white/10"
                            />
                          ) : (
                            <img
                              src={professorFileURL}
                              alt={`Submission ${row.submission_id}`}
                              className="max-h-[55vh] w-full rounded-lg border border-white/10 object-contain"
                            />
                          )
                        ) : null}
                      </div>
                    )}
                  </div>

                  <div className="rounded-xl border border-white/10 bg-white/5 p-4">
                    <p className="text-sm font-semibold text-slate-100">Current AI Evaluation</p>
                    <p className="mt-2 text-sm text-slate-300">Marks: {evaluationRow?.marks ?? '-'}</p>
                    <p className="mt-1 text-sm text-slate-300">Feedback: {evaluationRow?.feedback || '-'}</p>
                    <p className="mt-1 text-sm text-slate-400">Status: {evaluationRow?.is_final ? 'Finalized' : 'Draft'}</p>
                  </div>

                  {questionResults.length > 0 ? (
                    <div className="rounded-xl border border-white/10 bg-[#0b1020] p-4">
                      <p className="text-sm font-semibold text-slate-100">LLM Question-wise Marks and Reasons</p>
                      <div className="mt-3 space-y-3">
                        {questionResults.map((questionRow, index) => (
                          <div key={questionID(questionRow) || `llm-q-${index}`} className="rounded-lg border border-white/10 bg-white/5 p-3">
                            <p className="text-sm text-slate-100">Q{index + 1}: {questionText(questionRow)}</p>
                            <p className="mt-1 text-sm text-slate-300">LLM Marks: {questionMarks(questionRow) ?? '-'}</p>
                            <p className="mt-1 text-sm text-slate-300">Reason: {questionFeedback(questionRow) || '-'}</p>
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : null}

                  <div className="grid gap-3 md:grid-cols-2">
                    <input
                      type="number"
                      step="0.1"
                      value={autoTotalEnabled ? computedQuestionWiseTotal : (draft.marks ?? evaluationRow?.marks ?? '')}
                      onChange={(event) => {
                        if (autoTotalEnabled) return
                        setOverrideDraftByEvaluation((prev) => ({
                          ...prev,
                          [evaluationId]: {
                            ...prev[evaluationId],
                            marks: event.target.value,
                          },
                        }))
                      }}
                      className="input-field"
                      placeholder={autoTotalEnabled ? 'Auto computed from question-wise marks' : 'Override marks'}
                      readOnly={autoTotalEnabled}
                    />
                    <input
                      type="text"
                      value={draft.feedback ?? evaluationRow?.feedback ?? ''}
                      onChange={(event) =>
                        setOverrideDraftByEvaluation((prev) => ({
                          ...prev,
                          [evaluationId]: {
                            ...prev[evaluationId],
                            feedback: event.target.value,
                          },
                        }))
                      }
                      className="input-field"
                      placeholder="Override feedback"
                    />
                  </div>
                  {autoTotalEnabled ? (
                    <p className="text-xs text-slate-400">
                      Total marks are auto-computed from question-wise overrides.
                    </p>
                  ) : null}

                  {questionResults.length > 0 ? (
                    <div className="space-y-3 rounded-xl border border-white/10 bg-[#0b1020] p-4">
                      <p className="text-sm font-semibold text-slate-100">Question-wise Override</p>
                      {questionResults.map((questionRow, index) => {
                        const key = questionDraftKey(questionRow, index)
                        const questionDraft = draft.questions?.[key] || {}
                        return (
                          <div key={questionID(questionRow) || `question-override-${index}`} className="space-y-2 rounded-lg border border-white/10 bg-white/5 p-3">
                            <p className="text-xs text-slate-400">Question {index + 1}</p>
                            <div className="grid gap-3 md:grid-cols-2">
                              <input
                                type="number"
                                step="0.1"
                                value={questionDraft.marks ?? questionMarks(questionRow) ?? ''}
                                onChange={(event) =>
                                  setOverrideDraftByEvaluation((prev) => ({
                                    ...prev,
                                    [evaluationId]: {
                                      ...prev[evaluationId],
                                      questions: {
                                        ...(prev[evaluationId]?.questions || {}),
                                        [key]: {
                                          ...(prev[evaluationId]?.questions?.[key] || {}),
                                          marks: event.target.value,
                                        },
                                      },
                                    },
                                  }))
                                }
                                className="input-field"
                                placeholder="Question marks"
                              />
                              <input
                                type="text"
                                value={questionDraft.feedback ?? questionFeedback(questionRow) ?? ''}
                                onChange={(event) =>
                                  setOverrideDraftByEvaluation((prev) => ({
                                    ...prev,
                                    [evaluationId]: {
                                      ...prev[evaluationId],
                                      questions: {
                                        ...(prev[evaluationId]?.questions || {}),
                                        [key]: {
                                          ...(prev[evaluationId]?.questions?.[key] || {}),
                                          feedback: event.target.value,
                                        },
                                      },
                                    },
                                  }))
                                }
                                className="input-field"
                                placeholder="Question feedback"
                              />
                            </div>
                          </div>
                        )
                      })}
                    </div>
                  ) : null}

                  <div className="flex flex-wrap gap-3">
                    <button type="button" onClick={() => handleOverride(row)} className="btn-primary w-auto px-5 py-2.5">
                      Save Override
                    </button>
                    <button
                      type="button"
                      onClick={() => handleFinalize(row, !evaluationRow?.is_final)}
                      className="rounded-xl border border-white/10 bg-white/5 px-5 py-2.5 text-sm text-slate-100 transition-all duration-300 hover:border-blue-500/30 hover:bg-white/10"
                    >
                      {evaluationRow?.is_final ? 'Unfinalize' : 'Finalize & Publish'}
                    </button>
                  </div>
                </div>
              )
            })}
          </div>
        ) : null}
      </section>
    </CourseShell>
  )
}

export default AssignmentSubmissionPage
