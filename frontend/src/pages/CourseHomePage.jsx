import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import CourseShell from '../components/CourseShell'
import api from '../services/api'

function CourseHomePage() {
  const { courseId } = useParams()
  const [question, setQuestion] = useState('')
  const [chatRows, setChatRows] = useState([])
  const [chatLoading, setChatLoading] = useState(false)
  const [chatSending, setChatSending] = useState(false)
  const [chatError, setChatError] = useState('')

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
    const loadChatHistory = async () => {
      if (!isStudent) return
      setChatLoading(true)
      setChatError('')
      try {
        const response = await api.get(`/courses/${courseId}/chat-history`)
        const rows = Array.isArray(response.data) ? response.data : []
        setChatRows(rows)
      } catch (err) {
        setChatError(err?.response?.data?.error || 'Failed to load chatbot history')
      } finally {
        setChatLoading(false)
      }
    }

    loadChatHistory()
  }, [courseId, isStudent])

  const handleAskDoubt = async () => {
    const trimmed = question.trim()
    if (!trimmed) {
      setChatError('Please type a question before sending')
      return
    }
    setChatSending(true)
    setChatError('')
    try {
      const response = await api.post(`/courses/${courseId}/doubt`, { question: trimmed })
      const answer = response?.data?.answer || 'No answer returned.'
      const nextItem = {
        id: `${Date.now()}`,
        question: trimmed,
        answer,
        timestamp: new Date().toISOString(),
      }
      setChatRows((prev) => [...prev, nextItem])
      setQuestion('')
    } catch (err) {
      setChatError(err?.response?.data?.error || 'Failed to ask doubt')
    } finally {
      setChatSending(false)
    }
  }

  return (
    <CourseShell
      courseId={courseId}
      activeTab="overview"
      title={`Course ${courseId}`}
      description="Use this workspace to manage assignments and review analytics. Each section has a focused purpose for better navigation."
    >
      <section className="space-y-6">
        <div className="grid gap-4 md:grid-cols-3">
          <Link
            to={`/courses/${courseId}/materials`}
            className="app-card app-card-hover p-6"
          >
            <div className="flex items-center gap-3">
              <span className="flex h-9 w-9 items-center justify-center rounded-lg border border-emerald-400/30 bg-emerald-500/10 text-emerald-200">
                <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current" aria-hidden="true">
                  <path d="M12 2 2 7l10 5 10-5-10-5Zm0 7L2 4v3l10 5 10-5V4l-10 5Zm0 5L2 9v3l10 5 10-5V9l-10 5Zm0 5L2 14v3l10 5 10-5v-3l-10 5Z" />
                </svg>
              </span>
              <h2 className="text-lg font-semibold text-slate-100">Materials</h2>
            </div>
            <p className="mt-2 text-sm text-slate-300">
              Upload and view course materials used by the RAG chatbot.
            </p>
          </Link>

          <Link
            to={`/courses/${courseId}/assignments`}
            className="app-card app-card-hover p-6"
          >
            <div className="flex items-center gap-3">
              <span className="flex h-9 w-9 items-center justify-center rounded-lg border border-blue-400/30 bg-blue-500/10 text-blue-200">
                <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current" aria-hidden="true">
                  <path d="M5 3h10l4 4v14H5V3Zm2 2v14h10V8h-3V5H7Zm2 6h6v2H9v-2Zm0 4h6v2H9v-2Z" />
                </svg>
              </span>
              <h2 className="text-lg font-semibold text-slate-100">Assignments</h2>
            </div>
            <p className="mt-2 text-sm text-slate-300">
              Browse all assignments and open details in dedicated pages.
            </p>
          </Link>

          <Link
            to={`/courses/${courseId}/analytics`}
            className="app-card app-card-hover p-6"
          >
            <div className="flex items-center gap-3">
              <span className="flex h-9 w-9 items-center justify-center rounded-lg border border-cyan-400/30 bg-cyan-500/10 text-cyan-200">
                <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current" aria-hidden="true">
                  <path d="M4 20h16v2H2V4h2v16Zm2-3 3.5-4 3 2L17 9l1.5 1-5.2 7-3.2-2.1L7.6 18 6 17Z" />
                </svg>
              </span>
              <h2 className="text-lg font-semibold text-slate-100">Analytics</h2>
            </div>
            <p className="mt-2 text-sm text-slate-300">
              Track course performance, strengths, weak topics, and leaderboard.
            </p>
          </Link>
        </div>

        {isStudent ? (
          <div className="app-card p-6">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-3">
                <span className="flex h-8 w-8 items-center justify-center rounded-lg border border-violet-400/30 bg-violet-500/10 text-violet-200">
                  <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current" aria-hidden="true">
                    <path d="M12 2a7 7 0 0 0-7 7v3.2c-1.2.7-2 2-2 3.4V20h18v-4.4c0-1.4-.8-2.7-2-3.4V9a7 7 0 0 0-7-7Zm-5 7a5 5 0 1 1 10 0v2H7V9Zm1 6h2v2H8v-2Zm6 0h2v2h-2v-2Z" />
                  </svg>
                </span>
                <h2 className="text-lg font-semibold text-slate-100">Course Chatbot</h2>
              </div>
              <p className="text-xs text-slate-400">Ask doubts with text or symbols. We keep chat history in this course.</p>
            </div>

            <div className="mt-4 max-h-[320px] space-y-4 overflow-y-auto rounded-xl border border-white/10 bg-[#0b1020] p-4">
              {chatLoading ? <p className="text-sm text-slate-400">Loading chat history...</p> : null}
              {!chatLoading && chatRows.length === 0 ? (
                <p className="text-sm text-slate-400">No messages yet. Ask your first question.</p>
              ) : null}
              {chatRows.map((row) => (
                <div key={row.id} className="space-y-2">
                  <p className="rounded-lg border border-blue-500/20 bg-blue-500/10 p-3 text-sm text-slate-100">
                    <span className="font-semibold text-blue-300">You:</span> {row.question}
                  </p>
                  <p className="rounded-lg border border-emerald-500/20 bg-emerald-500/10 p-3 text-sm text-slate-100">
                    <span className="font-semibold text-emerald-300">Assistant:</span> {row.answer}
                  </p>
                </div>
              ))}
            </div>

            <div className="mt-4 flex flex-col gap-3 md:flex-row">
              <input
                type="text"
                value={question}
                onChange={(event) => setQuestion(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' && !event.shiftKey) {
                    event.preventDefault()
                    handleAskDoubt()
                  }
                }}
                className="input-field"
                placeholder="Ask your doubt... e.g. Explain recursion with an example"
              />
              <button type="button" onClick={handleAskDoubt} disabled={chatSending} className="btn-primary w-auto px-5 py-3">
                {chatSending ? 'Sending...' : 'Send'}
              </button>
            </div>
            {chatError ? <p className="mt-3 text-sm text-red-300">{chatError}</p> : null}
          </div>
        ) : null}
      </section>
    </CourseShell>
  )
}

export default CourseHomePage
