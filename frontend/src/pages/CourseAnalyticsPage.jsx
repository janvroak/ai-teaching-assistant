import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import CourseShell from '../components/CourseShell'
import api from '../services/api'

function CourseAnalyticsPage() {
  const { courseId } = useParams()
  const [analytics, setAnalytics] = useState(null)
  const [leaderboard, setLeaderboard] = useState([])
  const [studentRank, setStudentRank] = useState(null)
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

  useEffect(() => {
    const loadAnalytics = async () => {
      setLoading(true)
      setError('')
      try {
        const [courseAnalyticsResponse, leaderboardResponse] = await Promise.all([
          api.get(`/courses/${courseId}/analytics`),
          api.get(`/courses/${courseId}/leaderboard`),
        ])

        setAnalytics(courseAnalyticsResponse?.data?.course || null)
        setLeaderboard(Array.isArray(leaderboardResponse?.data?.leaderboard) ? leaderboardResponse.data.leaderboard : [])

        if (user?.role === 'student') {
          try {
            const studentRankResponse = await api.get(`/courses/${courseId}/student-rank`)
            setStudentRank(studentRankResponse?.data || null)
          } catch {
            setStudentRank(null)
          }
        }
      } catch (err) {
        setError(err?.response?.data?.error || 'Failed to load analytics')
      } finally {
        setLoading(false)
      }
    }

    loadAnalytics()
  }, [courseId, user?.role])

  const weakTopics = Array.isArray(analytics?.weak_topics) ? analytics.weak_topics : []
  const strongTopics = Array.isArray(analytics?.strong_topics) ? analytics.strong_topics : []

  return (
    <CourseShell
      courseId={courseId}
      activeTab="analytics"
      title="Course Analytics"
      description="A focused performance dashboard with clear metrics and topic-level insights."
    >
      <section className="space-y-6">
        {loading ? <p className="text-sm text-slate-400">Loading analytics...</p> : null}
        {error ? <p className="text-sm text-red-300">{error}</p> : null}

        {!loading && !error ? (
          <>
            {user?.role === 'student' && studentRank ? (
              <div className="app-card relative overflow-hidden p-6">
                <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-emerald-300/80 to-transparent" />
                <h2 className="text-base font-semibold text-slate-100">Your Rank</h2>
                <p className="mt-2 text-sm text-slate-300">
                  Rank #{studentRank.rank} of {studentRank.total_students}
                </p>
                <p className="mt-1 text-sm text-slate-400">
                  Percentile: {Number(studentRank.percentile || 0).toFixed(1)}%
                </p>
              </div>
            ) : null}

            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
              <div className="app-card p-5">
                <p className="text-xs uppercase tracking-wide text-slate-400">Average Score</p>
                <p className="mt-2 text-2xl font-semibold text-slate-100">
                  {analytics?.avg_score !== undefined && analytics?.avg_score !== null
                    ? Number(analytics.avg_score).toFixed(2)
                    : '-'}
                </p>
              </div>
              <div className="app-card p-5">
                <p className="text-xs uppercase tracking-wide text-slate-400">Top Student</p>
                <p className="mt-2 text-sm text-slate-100">{analytics?.top_performer?.name || '-'}</p>
              </div>
              <div className="app-card p-5">
                <p className="text-xs uppercase tracking-wide text-slate-400">Weak Topics</p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {weakTopics.length > 0 ? weakTopics.map((topic, index) => (
                    <span key={`weak-${index}`} className="rounded-md border border-amber-500/20 bg-amber-500/10 px-2 py-1 text-xs text-amber-100">
                      {topic}
                    </span>
                  )) : <span className="text-sm text-slate-400">None yet</span>}
                </div>
              </div>
              <div className="app-card p-5">
                <p className="text-xs uppercase tracking-wide text-slate-400">Strong Topics</p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {strongTopics.length > 0 ? strongTopics.map((topic, index) => (
                    <span key={`strong-${index}`} className="rounded-md border border-emerald-500/20 bg-emerald-500/10 px-2 py-1 text-xs text-emerald-100">
                      {topic}
                    </span>
                  )) : <span className="text-sm text-slate-400">None yet</span>}
                </div>
              </div>
            </div>

            <div className="app-card p-6">
              <h2 className="text-base font-semibold text-slate-100">Leaderboard</h2>
              {leaderboard.length === 0 ? (
                <p className="mt-3 text-sm text-slate-400">No leaderboard data yet.</p>
              ) : (
                <div className="mt-3 space-y-2">
                  {leaderboard.map((row, index) => (
                    <div
                      key={row.user_id}
                      className="grid grid-cols-[auto_1fr_auto] items-center gap-3 rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-slate-300 transition-all duration-300 hover:border-blue-500/30 hover:bg-white/10"
                    >
                      <span className="text-slate-400">#{index + 1}</span>
                      <span className="text-slate-100">{row.name || `Student ${row.user_id}`}</span>
                      <span>{Number(row.avg_score || 0).toFixed(2)}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </>
        ) : null}
      </section>
    </CourseShell>
  )
}

export default CourseAnalyticsPage
