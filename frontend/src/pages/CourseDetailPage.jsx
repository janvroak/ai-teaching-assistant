import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import Layout from '../components/Layout'
import api from '../services/api'

const SECTION_ASSIGNMENTS = 'assignments'
const SECTION_CHATBOT = 'chatbot'
const SECTION_MATERIALS = 'materials'

function CourseDetailPage() {
  const { id, assignmentId } = useParams()

  const [assignments, setAssignments] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [title, setTitle] = useState('')
  const [question, setQuestion] = useState('')
  const [questionInputMode, setQuestionInputMode] = useState('text')
  const [questionFile, setQuestionFile] = useState(null)
  const [answerKeyInputMode, setAnswerKeyInputMode] = useState('text')
  const [answerKeyFile, setAnswerKeyFile] = useState(null)
  const [answerKey, setAnswerKey] = useState('')

  const [contentByAssignment, setContentByAssignment] = useState({})
  const [fileByAssignment, setFileByAssignment] = useState({})
  const [fileSubmittingByAssignment, setFileSubmittingByAssignment] = useState({})

  const [resultByAssignment, setResultByAssignment] = useState({})
  const [submittedByAssignment, setSubmittedByAssignment] = useState({})
  const [mySubmissionByAssignment, setMySubmissionByAssignment] = useState({})

  const [submissionsByAssignment, setSubmissionsByAssignment] = useState({})
  const [submissionLoadingByAssignment, setSubmissionLoadingByAssignment] = useState({})
  const [submissionErrorByAssignment, setSubmissionErrorByAssignment] = useState({})

  const [overrideFormByEvaluation, setOverrideFormByEvaluation] = useState({})
  const [overrideLoadingByEvaluation, setOverrideLoadingByEvaluation] = useState({})
  const [overrideMessageByAssignment, setOverrideMessageByAssignment] = useState({})

  const [actionMessage, setActionMessage] = useState('')

  const [questionPDFBlobURLByAssignment, setQuestionPDFBlobURLByAssignment] = useState({})
  const [answerKeyPDFBlobURLByAssignment, setAnswerKeyPDFBlobURLByAssignment] = useState({})
  const [submissionPDFBlobURLByID, setSubmissionPDFBlobURLByID] = useState({})

  const [materials, setMaterials] = useState([])
  const [materialsLoading, setMaterialsLoading] = useState(false)
  const [materialsError, setMaterialsError] = useState('')
  const [materialTitle, setMaterialTitle] = useState('')
  const [materialInputMode, setMaterialInputMode] = useState('text')
  const [materialContent, setMaterialContent] = useState('')
  const [materialFile, setMaterialFile] = useState(null)
  const [materialActionMessage, setMaterialActionMessage] = useState('')

  const [doubtQuestion, setDoubtQuestion] = useState('')
  const [doubtLoading, setDoubtLoading] = useState(false)
  const [doubtError, setDoubtError] = useState('')
  const [doubtResult, setDoubtResult] = useState(null)

  const [activeSection, setActiveSection] = useState(SECTION_ASSIGNMENTS)
  const [selectedAssignmentId, setSelectedAssignmentId] = useState(
    assignmentId ? Number(assignmentId) : null,
  )

  const user = useMemo(() => {
    try {
      const raw = localStorage.getItem('user')
      return raw ? JSON.parse(raw) : null
    } catch {
      return null
    }
  }, [])

  const isProfessor = user?.role === 'professor'
  const isStudent = user?.role === 'student'
  const isAssignmentPage = Boolean(assignmentId)

  const loadAssignments = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await api.get(`/courses/${id}/assignments`)
      const rows = Array.isArray(response.data) ? response.data : []
      setAssignments(rows)
      if (!assignmentId && rows.length > 0) {
        setSelectedAssignmentId((prev) => prev ?? rows[0].id)
      }
      return rows
    } catch (err) {
      const message = err?.response?.data?.error || 'Failed to fetch assignments'
      setError(message)
      return []
    } finally {
      setLoading(false)
    }
  }

  const setQuestionPDFBlobURL = (assignmentId, blob) => {
    const nextURL = URL.createObjectURL(blob)
    setQuestionPDFBlobURLByAssignment((prev) => {
      if (prev[assignmentId]) URL.revokeObjectURL(prev[assignmentId])
      return { ...prev, [assignmentId]: nextURL }
    })
  }

  const setAnswerKeyPDFBlobURL = (assignmentId, blob) => {
    const nextURL = URL.createObjectURL(blob)
    setAnswerKeyPDFBlobURLByAssignment((prev) => {
      if (prev[assignmentId]) URL.revokeObjectURL(prev[assignmentId])
      return { ...prev, [assignmentId]: nextURL }
    })
  }

  const setSubmissionPDFBlobURL = (submissionId, blob) => {
    const nextURL = URL.createObjectURL(blob)
    setSubmissionPDFBlobURLByID((prev) => {
      if (prev[submissionId]) URL.revokeObjectURL(prev[submissionId])
      return { ...prev, [submissionId]: nextURL }
    })
  }

  const loadAssignmentPDFViews = async (courseAssignments) => {
    const rows = Array.isArray(courseAssignments) ? courseAssignments : []
    await Promise.all(
      rows.map(async (assignment) => {
        if (assignment?.question_type === 'pdf' && assignment?.question_pdf_url) {
          try {
            const response = await api.get(assignment.question_pdf_url, { responseType: 'blob' })
            setQuestionPDFBlobURL(assignment.id, response.data)
          } catch {}
        }
        if (isProfessor && assignment?.answer_key_type === 'pdf' && assignment?.answer_key_pdf_url) {
          try {
            const response = await api.get(assignment.answer_key_pdf_url, { responseType: 'blob' })
            setAnswerKeyPDFBlobURL(assignment.id, response.data)
          } catch {}
        }
      }),
    )
  }

  const loadStudentSubmissionStatus = async (courseAssignments) => {
    if (!isStudent) return

    try {
      const response = await api.get('/my-submissions')
      const rows = Array.isArray(response.data) ? response.data : []
      const assignmentIDSet = new Set((courseAssignments || []).map((item) => item.id))

      const nextSubmittedByAssignment = {}
      const nextResultByAssignment = {}
      const nextMySubmissionByAssignment = {}

      for (const row of rows) {
        if (!assignmentIDSet.has(row.assignment_id)) continue
        if (nextSubmittedByAssignment[row.assignment_id]) continue

        nextSubmittedByAssignment[row.assignment_id] = true
        nextResultByAssignment[row.assignment_id] = {
          marks: row?.evaluation?.marks ?? null,
          feedback: row?.evaluation?.feedback ?? '',
          confidence: row?.evaluation?.confidence ?? null,
        }
        nextMySubmissionByAssignment[row.assignment_id] = {
          submissionId: row?.submission_id,
          submissionType: row?.submission_type || 'text',
          content: row?.content || '',
          submissionPDFURL: row?.submission_pdf_url || '',
        }

        if (row?.submission_type === 'pdf' && row?.submission_pdf_url && row?.submission_id) {
          try {
            const pdfResponse = await api.get(row.submission_pdf_url, { responseType: 'blob' })
            setSubmissionPDFBlobURL(row.submission_id, pdfResponse.data)
          } catch {}
        }
      }

      setSubmittedByAssignment(nextSubmittedByAssignment)
      setMySubmissionByAssignment(nextMySubmissionByAssignment)
      setResultByAssignment((prev) => ({ ...prev, ...nextResultByAssignment }))
    } catch {}
  }

  const loadMaterials = async () => {
    setMaterialsLoading(true)
    setMaterialsError('')
    try {
      const response = await api.get(`/courses/${id}/materials`)
      setMaterials(Array.isArray(response.data) ? response.data : [])
    } catch (err) {
      const message = err?.response?.data?.error || 'Failed to fetch course materials'
      setMaterialsError(message)
    } finally {
      setMaterialsLoading(false)
    }
  }

  useEffect(() => {
    if (!assignmentId) {
      setSelectedAssignmentId(null)
      return
    }
    const parsed = Number(assignmentId)
    if (!Number.isNaN(parsed)) {
      setSelectedAssignmentId(parsed)
    }
    setActiveSection(SECTION_ASSIGNMENTS)
  }, [assignmentId])

  useEffect(() => {
    const initialize = async () => {
      const courseAssignments = await loadAssignments()
      await loadAssignmentPDFViews(courseAssignments)
      await loadStudentSubmissionStatus(courseAssignments)
      await loadMaterials()
    }

    initialize()

    return () => {
      setQuestionPDFBlobURLByAssignment((prev) => {
        Object.values(prev).forEach((url) => URL.revokeObjectURL(url))
        return {}
      })
      setAnswerKeyPDFBlobURLByAssignment((prev) => {
        Object.values(prev).forEach((url) => URL.revokeObjectURL(url))
        return {}
      })
      setSubmissionPDFBlobURLByID((prev) => {
        Object.values(prev).forEach((url) => URL.revokeObjectURL(url))
        return {}
      })
    }
  }, [id, assignmentId, isProfessor, isStudent])

  const handleCreateAssignment = async (event) => {
    event.preventDefault()
    const trimmedTitle = title.trim()
    const trimmedQuestion = question.trim()
    const trimmedAnswerKey = answerKey.trim()

    if (!trimmedTitle) return setActionMessage('Title is required')
    if (questionInputMode === 'text' && !trimmedQuestion) return setActionMessage('Question text is required')
    if (questionInputMode === 'pdf' && !questionFile) return setActionMessage('Question PDF is required')
    if (answerKeyInputMode === 'text' && !trimmedAnswerKey) return setActionMessage('Answer key text is required')
    if (answerKeyInputMode === 'pdf' && !answerKeyFile) return setActionMessage('Answer key PDF is required')

    try {
      setActionMessage('')
      const formData = new FormData()
      formData.append('course_id', String(Number(id)))
      formData.append('title', trimmedTitle)
      if (questionInputMode === 'text') formData.append('question_text', trimmedQuestion)
      else if (questionFile) formData.append('question_file', questionFile)
      if (answerKeyInputMode === 'text') formData.append('answer_key_text', trimmedAnswerKey)
      else if (answerKeyFile) formData.append('answer_key_file', answerKeyFile)

      await api.post('/assignments', formData)
      setTitle('')
      setQuestion('')
      setQuestionFile(null)
      setAnswerKeyFile(null)
      setAnswerKey('')
      setActionMessage('Assignment created successfully')
      const courseAssignments = await loadAssignments()
      await loadAssignmentPDFViews(courseAssignments)
    } catch (err) {
      setActionMessage(err?.response?.data?.error || 'Failed to create assignment')
    }
  }

  const handleUploadMaterial = async (event) => {
    event.preventDefault()
    const trimmedTitle = materialTitle.trim()
    const trimmedContent = materialContent.trim()
    if (!trimmedTitle) return setMaterialActionMessage('Material title is required')
    if (materialInputMode === 'text' && !trimmedContent) return setMaterialActionMessage('Material text content is required')
    if (materialInputMode === 'pdf' && !materialFile) return setMaterialActionMessage('Material PDF file is required')

    try {
      setMaterialActionMessage('')
      const formData = new FormData()
      formData.append('title', trimmedTitle)
      if (materialInputMode === 'text') formData.append('content', trimmedContent)
      else formData.append('file', materialFile)
      await api.post(`/courses/${id}/materials`, formData)
      setMaterialTitle('')
      setMaterialContent('')
      setMaterialFile(null)
      setMaterialActionMessage('Material uploaded successfully')
      await loadMaterials()
    } catch (err) {
      setMaterialActionMessage(err?.response?.data?.error || 'Failed to upload material')
    }
  }

  const handleAskDoubt = async () => {
    const trimmedQuestion = doubtQuestion.trim()
    if (!trimmedQuestion) return setDoubtError('Please enter your question')
    setDoubtLoading(true)
    setDoubtError('')
    setDoubtResult(null)
    try {
      const response = await api.post(`/courses/${id}/doubt`, { question: trimmedQuestion })
      setDoubtResult(response?.data || null)
    } catch (err) {
      setDoubtError(err?.response?.data?.error || 'Failed to get response from course assistant')
    } finally {
      setDoubtLoading(false)
    }
  }

  const handleSubmitAnswer = async (assignmentId) => {
    if (submittedByAssignment[assignmentId]) return setActionMessage('You have already submitted this assignment')
    const content = (contentByAssignment[assignmentId] || '').trim()
    if (!content) return setActionMessage('Please write your answer before submitting')

    try {
      setActionMessage('')
      const response = await api.post('/submit', { assignment_id: assignmentId, content })
      setContentByAssignment((prev) => ({ ...prev, [assignmentId]: '' }))
      setResultByAssignment((prev) => ({
        ...prev,
        [assignmentId]: {
          marks: response?.data?.marks,
          feedback: response?.data?.feedback,
          confidence: response?.data?.confidence,
        },
      }))
      setSubmittedByAssignment((prev) => ({ ...prev, [assignmentId]: true }))
      setMySubmissionByAssignment((prev) => ({
        ...prev,
        [assignmentId]: { submissionId: null, submissionType: 'text', content, submissionPDFURL: '' },
      }))
      setActionMessage('Answer submitted successfully')
    } catch (err) {
      setActionMessage(err?.response?.data?.error || 'Failed to submit answer')
    }
  }

  const handleFileSelect = (assignmentId, file) => {
    setFileByAssignment((prev) => ({ ...prev, [assignmentId]: file || null }))
  }

  const handleSubmitFile = async (assignmentId) => {
    if (submittedByAssignment[assignmentId]) return setActionMessage('You have already submitted this assignment')
    const file = fileByAssignment[assignmentId]
    if (!file) return setActionMessage('Please choose a PDF file before uploading')
    const lowerName = String(file.name || '').toLowerCase()
    if (!lowerName.endsWith('.pdf')) return setActionMessage('Only PDF files are allowed')

    const formData = new FormData()
    formData.append('assignment_id', String(assignmentId))
    formData.append('file', file)
    setFileSubmittingByAssignment((prev) => ({ ...prev, [assignmentId]: true }))

    try {
      setActionMessage('')
      const response = await api.post('/submit-file', formData)
      setFileByAssignment((prev) => ({ ...prev, [assignmentId]: null }))
      setResultByAssignment((prev) => ({
        ...prev,
        [assignmentId]: {
          marks: response?.data?.marks,
          feedback: response?.data?.feedback,
          confidence: null,
        },
      }))
      setSubmittedByAssignment((prev) => ({ ...prev, [assignmentId]: true }))

      const responseSubmissionId = response?.data?.submission_id
      const responseSubmissionURL = response?.data?.submission_pdf_url
      if (responseSubmissionId && responseSubmissionURL) {
        try {
          const pdfResponse = await api.get(responseSubmissionURL, { responseType: 'blob' })
          setSubmissionPDFBlobURL(responseSubmissionId, pdfResponse.data)
        } catch {}
      }

      setMySubmissionByAssignment((prev) => ({
        ...prev,
        [assignmentId]: {
          submissionId: responseSubmissionId || null,
          submissionType: 'pdf',
          content: '',
          submissionPDFURL: responseSubmissionURL || '',
        },
      }))
      setActionMessage('PDF uploaded and evaluated successfully')
    } catch (err) {
      setActionMessage(err?.response?.data?.error || 'Failed to upload and evaluate file')
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
      for (const row of rows) {
        if (row?.submission_type !== 'pdf' || !row?.submission_pdf_url || !row?.submission_id) continue
        try {
          const pdfResponse = await api.get(row.submission_pdf_url, { responseType: 'blob' })
          setSubmissionPDFBlobURL(row.submission_id, pdfResponse.data)
        } catch {}
      }
    } catch (err) {
      setSubmissionErrorByAssignment((prev) => ({
        ...prev,
        [assignmentId]: err?.response?.data?.error || 'Failed to fetch submissions',
      }))
    } finally {
      setSubmissionLoadingByAssignment((prev) => ({ ...prev, [assignmentId]: false }))
    }
  }

  const handleOverrideChange = (evaluationId, field, value) => {
    setOverrideFormByEvaluation((prev) => ({
      ...prev,
      [evaluationId]: { ...prev[evaluationId], [field]: value },
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
      await api.put(`/evaluations/${evaluationId}`, { marks, feedback })
      setSubmissionsByAssignment((prev) => {
        const currentRows = prev[assignmentId]
        if (!Array.isArray(currentRows)) return prev
        return {
          ...prev,
          [assignmentId]: currentRows.map((row) => {
            if (row?.evaluation?.id !== evaluationId) return row
            return { ...row, evaluation: { ...row.evaluation, marks, feedback } }
          }),
        }
      })
      setOverrideFormByEvaluation((prev) => ({ ...prev, [evaluationId]: { marks, feedback } }))
      setOverrideMessageByAssignment((prev) => ({ ...prev, [assignmentId]: 'Evaluation updated successfully' }))
      await loadSubmissionsForAssignment(assignmentId)
    } catch (err) {
      setOverrideMessageByAssignment((prev) => ({
        ...prev,
        [assignmentId]: err?.response?.data?.error || 'Failed to update evaluation',
      }))
    } finally {
      setOverrideLoadingByEvaluation((prev) => ({ ...prev, [evaluationId]: false }))
    }
  }

  const selectedAssignment = assignments.find((item) => item.id === selectedAssignmentId) || null
  const getStudentScore = (assignmentId) => {
    const marks = resultByAssignment[assignmentId]?.marks
    return marks === undefined || marks === null ? 'Not graded' : String(marks)
  }
  const tabs = isProfessor
    ? [
        { id: SECTION_ASSIGNMENTS, label: 'Assignments' },
        { id: SECTION_CHATBOT, label: 'Chatbot' },
        { id: SECTION_MATERIALS, label: 'Materials' },
      ]
    : [
        { id: SECTION_ASSIGNMENTS, label: 'Assignments' },
        { id: SECTION_CHATBOT, label: 'Chatbot' },
      ]

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">Course Workspace</h1>
            <p className="mt-1 text-sm text-gray-600">Course ID: {id}</p>
          </div>
          <Link to="/dashboard" className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600">
            Back to Dashboard
          </Link>
        </div>

        <section className="rounded-xl bg-white p-4 shadow">
          <div className="flex flex-wrap gap-2">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveSection(tab.id)}
                className={`rounded-full px-4 py-2 text-sm font-medium ${
                  activeSection === tab.id
                    ? 'bg-blue-600 text-white'
                    : 'bg-slate-100 text-slate-700 hover:bg-slate-200'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </section>

        {activeSection === SECTION_ASSIGNMENTS ? (
          <section className="space-y-5 rounded-xl bg-white p-5 shadow">
            <div>
              <h2 className="text-lg font-medium text-slate-900">Assignments</h2>
              <p className="mt-1 text-sm text-gray-600">
                {isStudent
                  ? 'Choose an assignment to view details, submission, and score.'
                  : 'Choose an assignment to review submissions and manage grading.'}
              </p>
            </div>

            {isProfessor ? (
              <form onSubmit={handleCreateAssignment} className="rounded-lg bg-slate-50 p-4 ring-1 ring-slate-200">
                <h3 className="text-base font-semibold text-slate-900">Create Assignment</h3>
                <div className="mt-3 space-y-4">
                  <input
                    type="text"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    placeholder="Assignment title"
                    className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                  />
                  <div className="space-y-2 rounded border border-slate-200 p-3">
                    <p className="text-sm font-medium text-slate-900">Question input</p>
                    <div className="flex flex-wrap gap-4">
                      <label className="inline-flex items-center gap-2 text-sm text-slate-700">
                        <input
                          type="radio"
                          name="question-input-mode"
                          value="text"
                          checked={questionInputMode === 'text'}
                          onChange={() => setQuestionInputMode('text')}
                        />
                        Text
                      </label>
                      <label className="inline-flex items-center gap-2 text-sm text-slate-700">
                        <input
                          type="radio"
                          name="question-input-mode"
                          value="pdf"
                          checked={questionInputMode === 'pdf'}
                          onChange={() => setQuestionInputMode('pdf')}
                        />
                        PDF
                      </label>
                    </div>
                    {questionInputMode === 'text' ? (
                      <textarea
                        rows={4}
                        value={question}
                        onChange={(e) => setQuestion(e.target.value)}
                        placeholder="Question"
                        className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                      />
                    ) : (
                      <input
                        type="file"
                        accept="application/pdf,.pdf"
                        onChange={(e) => setQuestionFile(e.target.files?.[0] || null)}
                        className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                      />
                    )}
                  </div>
                  <div className="space-y-2 rounded border border-slate-200 p-3">
                    <p className="text-sm font-medium text-slate-900">Answer key input</p>
                    <div className="flex flex-wrap gap-4">
                      <label className="inline-flex items-center gap-2 text-sm text-slate-700">
                        <input
                          type="radio"
                          name="answer-key-input-mode"
                          value="text"
                          checked={answerKeyInputMode === 'text'}
                          onChange={() => setAnswerKeyInputMode('text')}
                        />
                        Text
                      </label>
                      <label className="inline-flex items-center gap-2 text-sm text-slate-700">
                        <input
                          type="radio"
                          name="answer-key-input-mode"
                          value="pdf"
                          checked={answerKeyInputMode === 'pdf'}
                          onChange={() => setAnswerKeyInputMode('pdf')}
                        />
                        PDF
                      </label>
                    </div>
                    {answerKeyInputMode === 'text' ? (
                      <textarea
                        rows={4}
                        value={answerKey}
                        onChange={(e) => setAnswerKey(e.target.value)}
                        placeholder="Answer key"
                        className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                      />
                    ) : (
                      <input
                        type="file"
                        accept="application/pdf,.pdf"
                        onChange={(e) => setAnswerKeyFile(e.target.files?.[0] || null)}
                        className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                      />
                    )}
                  </div>
                  <button type="submit" className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600">
                    Create Assignment
                  </button>
                </div>
              </form>
            ) : null}

            {loading ? <p className="text-sm text-slate-500">Loading assignments...</p> : null}
            {error ? <p className="text-sm text-red-600">{error}</p> : null}
            {!loading && !error && assignments.length === 0 ? (
              <p className="text-sm text-slate-500">No assignments available for this course.</p>
            ) : null}

            {!loading && !error && assignments.length > 0 ? (
              <div className={isAssignmentPage ? 'space-y-4' : 'grid gap-5 lg:grid-cols-[320px_minmax(0,1fr)]'}>
                {!isAssignmentPage ? (
                  <aside className="rounded-lg border border-slate-200 bg-slate-50 p-3">
                  <p className="mb-2 text-sm font-medium text-slate-900">Assignment List</p>
                  <div className="space-y-2">
                    {assignments.map((assignment) => {
                      const isSubmitted = !!submittedByAssignment[assignment.id]
                      return (
                        <Link
                          key={assignment.id}
                          to={`/courses/${id}/assignments/${assignment.id}`}
                          className="block w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-left transition hover:border-slate-300"
                        >
                          <p className="text-sm font-semibold text-slate-900">{assignment.title}</p>
                          {isStudent ? (
                            <>
                              <p className="mt-1 text-xs text-slate-600">Status: {isSubmitted ? 'Submitted' : 'Pending'}</p>
                              <p className="text-xs text-slate-600">Score: {getStudentScore(assignment.id)}</p>
                            </>
                          ) : (
                            <p className="mt-1 text-xs text-slate-600">Assignment ID: {assignment.id}</p>
                          )}
                        </Link>
                      )
                    })}
                  </div>
                </aside>
                ) : null}

                {isAssignmentPage ? (
                <div className="rounded-lg border border-slate-200 p-4">
                  {isAssignmentPage ? (
                    <div className="mb-3">
                      <Link
                        to={`/courses/${id}`}
                        className="inline-flex rounded bg-slate-100 px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-200"
                      >
                        Back to Assignment List
                      </Link>
                    </div>
                  ) : null}
                  {selectedAssignment ? (
                    <>
                      <h3 className="text-lg font-semibold text-slate-900">{selectedAssignment.title}</h3>

                      <div className="mt-3 rounded-lg border bg-slate-50 p-3">
                        <p className="text-sm font-semibold text-slate-900">Question</p>
                        {selectedAssignment.question_type === 'pdf' && questionPDFBlobURLByAssignment[selectedAssignment.id] ? (
                          <div className="mt-2 overflow-hidden rounded-lg border border-slate-200 bg-white">
                            <iframe
                              src={questionPDFBlobURLByAssignment[selectedAssignment.id]}
                              title={`Question PDF ${selectedAssignment.id}`}
                              className="h-80 w-full"
                            />
                          </div>
                        ) : null}
                        {selectedAssignment.question_type === 'pdf' && !questionPDFBlobURLByAssignment[selectedAssignment.id] ? (
                          <p className="mt-1 text-sm text-slate-500">Loading question PDF...</p>
                        ) : null}
                        {selectedAssignment.question_type !== 'pdf' ? (
                          <p className="mt-1 whitespace-pre-wrap text-sm text-slate-700">{selectedAssignment.question}</p>
                        ) : null}
                      </div>

                      {isProfessor ? (
                        <div className="mt-4 space-y-4">
                          <div className="rounded-lg border bg-slate-50 p-3">
                            <p className="text-sm font-semibold text-slate-900">Answer Key</p>
                            {selectedAssignment.answer_key_type === 'pdf' && answerKeyPDFBlobURLByAssignment[selectedAssignment.id] ? (
                              <div className="mt-2 overflow-hidden rounded-lg border border-slate-200 bg-white">
                                <iframe
                                  src={answerKeyPDFBlobURLByAssignment[selectedAssignment.id]}
                                  title={`Answer Key PDF ${selectedAssignment.id}`}
                                  className="h-80 w-full"
                                />
                              </div>
                            ) : null}
                            {selectedAssignment.answer_key_type === 'pdf' && !answerKeyPDFBlobURLByAssignment[selectedAssignment.id] ? (
                              <p className="mt-1 text-sm text-slate-500">Loading answer key PDF...</p>
                            ) : null}
                            {selectedAssignment.answer_key_type !== 'pdf' ? (
                              <p className="mt-1 whitespace-pre-wrap text-sm text-slate-700">{selectedAssignment.answer_key}</p>
                            ) : null}
                          </div>

                          <div className="rounded-lg bg-slate-50 p-3 ring-1 ring-slate-200">
                            <div className="flex items-center justify-between gap-2">
                              <p className="text-base font-medium text-slate-900">Professor Review Panel</p>
                              <button
                                onClick={() => loadSubmissionsForAssignment(selectedAssignment.id)}
                                className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600"
                              >
                                {submissionLoadingByAssignment[selectedAssignment.id] ? 'Loading...' : 'Load Submissions'}
                              </button>
                            </div>

                            {submissionErrorByAssignment[selectedAssignment.id] ? (
                              <p className="mt-2 text-sm text-red-600">{submissionErrorByAssignment[selectedAssignment.id]}</p>
                            ) : null}
                            {overrideMessageByAssignment[selectedAssignment.id] ? (
                              <p className="mt-2 text-sm text-slate-700">{overrideMessageByAssignment[selectedAssignment.id]}</p>
                            ) : null}

                            {Array.isArray(submissionsByAssignment[selectedAssignment.id]) &&
                            submissionsByAssignment[selectedAssignment.id].length > 0 ? (
                              <div className="mt-3">
                                {submissionsByAssignment[selectedAssignment.id].map((submission) => {
                                  const evaluation = submission.evaluation || {}
                                  const evaluationId = evaluation.id
                                  return (
                                    <div key={submission.submission_id} className="mb-4 rounded-xl border bg-white p-4">
                                      <p className="text-sm text-gray-600">Submission #{submission.submission_id}</p>
                                      <p className="text-sm text-gray-600">Student ID: {submission.student_id}</p>
                                      <div className="my-2 border-t" />
                                      <p className="text-sm font-semibold text-slate-900">Student Answer</p>
                                      {submission.submission_type === 'pdf' && submissionPDFBlobURLByID[submission.submission_id] ? (
                                        <div className="mt-2 overflow-hidden rounded-lg border border-slate-200 bg-white">
                                          <iframe
                                            src={submissionPDFBlobURLByID[submission.submission_id]}
                                            title={`Submission PDF ${submission.submission_id}`}
                                            className="h-72 w-full"
                                          />
                                        </div>
                                      ) : null}
                                      {submission.submission_type === 'pdf' && !submissionPDFBlobURLByID[submission.submission_id] ? (
                                        <p className="mt-1 text-sm text-slate-500">Loading submission PDF...</p>
                                      ) : null}
                                      {submission.submission_type !== 'pdf' ? (
                                        <p className="mt-1 whitespace-pre-wrap text-sm text-gray-700">{submission.content}</p>
                                      ) : null}
                                      <div className="my-2 border-t" />
                                      <div className="rounded-lg border border-blue-200 bg-blue-50 p-3">
                                        <p className="text-base font-medium text-slate-900">AI Evaluation</p>
                                        <p className="mt-1 text-sm text-gray-700">
                                          <span className="font-semibold text-slate-900">Marks:</span> {evaluation.marks ?? '-'}
                                        </p>
                                        <p className="mt-1 text-sm text-gray-700">
                                          <span className="font-semibold text-slate-900">Feedback:</span> {evaluation.feedback || '-'}
                                        </p>
                                        <p className="mt-1 text-sm text-gray-700">
                                          <span className="font-semibold text-slate-900">Confidence:</span> {evaluation.confidence ?? '-'}
                                        </p>
                                      </div>
                                      {evaluationId ? (
                                        <div className="mt-3 space-y-3">
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
                                          <button
                                            onClick={() =>
                                              handleOverrideSubmit(
                                                selectedAssignment.id,
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
                                      ) : (
                                        <p className="mt-2 text-sm text-slate-500">Evaluation not available for this submission.</p>
                                      )}
                                    </div>
                                  )
                                })}
                              </div>
                            ) : null}

                            {Array.isArray(submissionsByAssignment[selectedAssignment.id]) &&
                            submissionsByAssignment[selectedAssignment.id].length === 0 &&
                            !submissionLoadingByAssignment[selectedAssignment.id] &&
                            !submissionErrorByAssignment[selectedAssignment.id] ? (
                              <p className="mt-2 text-sm text-slate-500">No submissions yet for this assignment.</p>
                            ) : null}
                          </div>
                        </div>
                      ) : null}

                      {isStudent ? (
                        <div className="mt-4 space-y-4">
                          {submittedByAssignment[selectedAssignment.id] ? (
                            <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-3">
                              <p className="text-sm font-medium text-emerald-900">Submission completed.</p>
                              <p className="mt-1 text-xs text-emerald-800">Resubmission is disabled.</p>
                            </div>
                          ) : (
                            <>
                              <div className="rounded-lg border border-slate-200 bg-slate-50 p-3">
                                <p className="text-sm font-medium text-slate-900">Write Answer</p>
                                <textarea
                                  rows={4}
                                  value={contentByAssignment[selectedAssignment.id] || ''}
                                  onChange={(e) =>
                                    setContentByAssignment((prev) => ({
                                      ...prev,
                                      [selectedAssignment.id]: e.target.value,
                                    }))
                                  }
                                  placeholder="Write your answer"
                                  className="mt-2 w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                                />
                                <button
                                  onClick={() => handleSubmitAnswer(selectedAssignment.id)}
                                  className="mt-2 rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600"
                                >
                                  Submit Answer
                                </button>
                              </div>

                              <div className="rounded-lg border border-slate-200 bg-slate-50 p-3">
                                <p className="text-sm font-medium text-slate-900">Or Upload PDF Submission</p>
                                <input
                                  type="file"
                                  accept="application/pdf,.pdf"
                                  onChange={(e) => handleFileSelect(selectedAssignment.id, e.target.files?.[0])}
                                  className="mt-2 w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                                />
                                <button
                                  onClick={() => handleSubmitFile(selectedAssignment.id)}
                                  disabled={fileSubmittingByAssignment[selectedAssignment.id]}
                                  className="mt-2 rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600 disabled:opacity-60"
                                >
                                  {fileSubmittingByAssignment[selectedAssignment.id] ? 'Uploading...' : 'Upload PDF'}
                                </button>
                              </div>
                            </>
                          )}

                          {mySubmissionByAssignment[selectedAssignment.id] ? (
                            <div className="rounded-lg border border-slate-200 bg-white p-3">
                              <p className="text-sm font-semibold text-slate-900">Your Response</p>
                              {mySubmissionByAssignment[selectedAssignment.id].submissionType === 'pdf' &&
                              mySubmissionByAssignment[selectedAssignment.id].submissionId &&
                              submissionPDFBlobURLByID[mySubmissionByAssignment[selectedAssignment.id].submissionId] ? (
                                <div className="mt-2 overflow-hidden rounded-lg border border-slate-200 bg-white">
                                  <iframe
                                    src={submissionPDFBlobURLByID[mySubmissionByAssignment[selectedAssignment.id].submissionId]}
                                    title={`My Submission PDF ${mySubmissionByAssignment[selectedAssignment.id].submissionId}`}
                                    className="h-72 w-full"
                                  />
                                </div>
                              ) : null}
                              {mySubmissionByAssignment[selectedAssignment.id].submissionType !== 'pdf' ? (
                                <p className="mt-1 whitespace-pre-wrap text-sm text-slate-700">
                                  {mySubmissionByAssignment[selectedAssignment.id].content || '-'}
                                </p>
                              ) : null}
                            </div>
                          ) : null}

                          {resultByAssignment[selectedAssignment.id] ? (
                            <div className="rounded-xl border border-green-200 bg-green-50 p-4">
                              <p className="text-xl font-bold text-green-900">
                                Marks: {resultByAssignment[selectedAssignment.id].marks ?? '-'}
                              </p>
                              <p className="mt-2 text-sm text-green-900">
                                <span className="font-semibold">Feedback:</span>{' '}
                                {resultByAssignment[selectedAssignment.id].feedback || '-'}
                              </p>
                              <p className="mt-2 text-xs text-green-800">
                                <span className="font-semibold">Confidence:</span>{' '}
                                {resultByAssignment[selectedAssignment.id].confidence ?? '-'}
                              </p>
                            </div>
                          ) : null}
                        </div>
                      ) : null}
                    </>
                  ) : (
                    <p className="text-sm text-slate-500">Assignment not found for this course.</p>
                  )}
                </div>
                ) : null}
              </div>
            ) : null}
          </section>
        ) : null}

        {activeSection === SECTION_CHATBOT ? (
          <section className="rounded-xl bg-white p-5 shadow">
            <h2 className="text-lg font-medium text-slate-900">Course Helper Chatbot</h2>
            <p className="mt-1 text-sm text-gray-600">
              Explains concepts using only uploaded course materials. If a topic is missing, it will tell you.
            </p>
            <div className="mt-3 space-y-3">
              <textarea
                value={doubtQuestion}
                onChange={(e) => setDoubtQuestion(e.target.value)}
                rows={3}
                placeholder="Ask a course-related question..."
                className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              <button
                onClick={handleAskDoubt}
                disabled={doubtLoading}
                className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600 disabled:opacity-60"
              >
                {doubtLoading ? 'Thinking...' : 'Ask'}
              </button>
            </div>
            {doubtError ? <p className="mt-2 text-sm text-red-600">{doubtError}</p> : null}
            {doubtResult?.answer ? (
              <div className="mt-4 rounded-lg border border-blue-200 bg-blue-50 p-4">
                <p className="text-sm font-semibold text-slate-900">Answer</p>
                <p className="mt-1 whitespace-pre-wrap text-sm text-gray-700">{doubtResult.answer}</p>
                <p className="mt-2 text-xs text-gray-600">Confidence: {doubtResult.confidence ?? '-'}</p>
                {Array.isArray(doubtResult.citations) && doubtResult.citations.length > 0 ? (
                  <p className="mt-1 text-xs text-gray-600">Sources: {doubtResult.citations.join(', ')}</p>
                ) : null}
              </div>
            ) : null}
          </section>
        ) : null}

        {isProfessor && activeSection === SECTION_MATERIALS ? (
          <section className="space-y-5 rounded-xl bg-white p-5 shadow">
            <div>
              <h2 className="text-lg font-medium text-slate-900">Course Materials</h2>
              <p className="mt-1 text-sm text-gray-600">Upload and manage RAG content for this course.</p>
            </div>
            <form onSubmit={handleUploadMaterial} className="space-y-4 rounded-lg bg-slate-50 p-4 ring-1 ring-slate-200">
              <input
                type="text"
                value={materialTitle}
                onChange={(e) => setMaterialTitle(e.target.value)}
                placeholder="Material title"
                className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              <div className="space-y-2">
                <div className="flex flex-wrap gap-4">
                  <label className="inline-flex items-center gap-2 text-sm text-slate-700">
                    <input
                      type="radio"
                      name="material-input-mode"
                      value="text"
                      checked={materialInputMode === 'text'}
                      onChange={() => setMaterialInputMode('text')}
                    />
                    Text
                  </label>
                  <label className="inline-flex items-center gap-2 text-sm text-slate-700">
                    <input
                      type="radio"
                      name="material-input-mode"
                      value="pdf"
                      checked={materialInputMode === 'pdf'}
                      onChange={() => setMaterialInputMode('pdf')}
                    />
                    PDF
                  </label>
                </div>
                {materialInputMode === 'text' ? (
                  <textarea
                    value={materialContent}
                    onChange={(e) => setMaterialContent(e.target.value)}
                    rows={4}
                    placeholder="Paste course notes/content"
                    className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                  />
                ) : (
                  <input
                    type="file"
                    accept="application/pdf,.pdf"
                    onChange={(e) => setMaterialFile(e.target.files?.[0] || null)}
                    className="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                  />
                )}
              </div>
              <button type="submit" className="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600">
                Upload Material
              </button>
              {materialActionMessage ? <p className="text-sm text-slate-700">{materialActionMessage}</p> : null}
            </form>

            <div>
              <h3 className="text-base font-semibold text-slate-900">Material Library</h3>
              {materialsLoading ? <p className="mt-2 text-sm text-slate-500">Loading materials...</p> : null}
              {materialsError ? <p className="mt-2 text-sm text-red-600">{materialsError}</p> : null}
              {!materialsLoading && !materialsError && materials.length === 0 ? (
                <p className="mt-2 text-sm text-slate-500">No materials uploaded yet.</p>
              ) : null}
              {!materialsLoading && !materialsError && materials.length > 0 ? (
                <div className="mt-3 space-y-2">
                  {materials.map((material) => (
                    <div key={material.id} className="rounded border bg-slate-50 px-3 py-2 text-sm text-slate-700">
                      <p className="font-medium text-slate-900">{material.title}</p>
                      <p className="text-xs text-slate-600">Type: {material.has_file ? 'PDF' : 'Text'}</p>
                    </div>
                  ))}
                </div>
              ) : null}
            </div>
          </section>
        ) : null}

        {actionMessage ? <p className="text-sm text-slate-700">{actionMessage}</p> : null}
      </div>
    </Layout>
  )
}

export default CourseDetailPage
