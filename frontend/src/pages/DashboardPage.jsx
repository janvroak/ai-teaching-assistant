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
  const [actionMessage, setActionMessage] = useState('')
  const [actionSuccess, setActionSuccess] = useState(false)
  const [deletingCourseID, setDeletingCourseID] = useState(null)

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
    if (!trimmedName) { setActionMessage('Enter a course name'); setActionSuccess(false); return }
    try {
      setActionMessage('')
      await api.post('/courses', { name: trimmedName })
      setCourseName('')
      setActionMessage('Course created successfully')
      setActionSuccess(true)
      await loadCourses()
    } catch (err) {
      setActionMessage(err?.response?.data?.error || 'Failed to create course')
      setActionSuccess(false)
    }
  }

  const handleJoinCourse = async () => {
    const trimmedCode = courseCode.trim().toUpperCase()
    if (!trimmedCode) { setActionMessage('Enter a course code'); setActionSuccess(false); return }
    try {
      setActionMessage('')
      await api.post('/courses/join', { course_code: trimmedCode })
      setCourseCode('')
      setActionMessage('Joined course successfully')
      setActionSuccess(true)
      await loadCourses()
    } catch (err) {
      setActionMessage(err?.response?.data?.error || 'Failed to join course')
      setActionSuccess(false)
    }
  }

  const handleDeleteCourse = async (courseId, courseName) => {
    const confirmed = window.confirm(`Delete "${courseName}"? This action will permanently remove all assignments, submissions, evaluations, and enrollments in this course.`)
    if (!confirmed) return

    try {
      setDeletingCourseID(courseId)
      setActionMessage('')
      await api.delete(`/courses/${courseId}`)
      setActionMessage('Course deleted successfully')
      setActionSuccess(true)
      await loadCourses()
    } catch (err) {
      setActionMessage(err?.response?.data?.error || 'Failed to delete course')
      setActionSuccess(false)
    } finally {
      setDeletingCourseID(null)
    }
  }

  const firstName = user?.email?.split('@')[0] || 'there'
  const hour = new Date().getHours()
  const greeting = hour < 12 ? 'Good morning' : hour < 17 ? 'Good afternoon' : 'Good evening'

  return (
    <Layout>
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=Syne:wght@400;600;700;800&family=DM+Sans:ital,wght@0,300;0,400;0,500;1,300&display=swap');

        .db-root {
          font-family: 'DM Sans', sans-serif;
          color: #e8eaf6;
          min-height: 100vh;
          background: #05060f;
          padding: clamp(16px, 3vw, 32px);
          position: relative;
        }

        .db-root::before {
          content: '';
          position: fixed;
          inset: 0;
          background-image:
            linear-gradient(rgba(79,142,247,0.025) 1px, transparent 1px),
            linear-gradient(90deg, rgba(79,142,247,0.025) 1px, transparent 1px);
          background-size: 60px 60px;
          pointer-events: none;
          z-index: 0;
        }

        .db-root::after {
          content: '';
          position: fixed;
          inset: 0;
          background:
            radial-gradient(ellipse at 10% 20%, rgba(37,99,235,0.13) 0%, transparent 45%),
            radial-gradient(ellipse at 90% 70%, rgba(124,58,237,0.11) 0%, transparent 40%),
            radial-gradient(ellipse at 50% 95%, rgba(79,142,247,0.09) 0%, transparent 40%);
          pointer-events: none;
          z-index: 0;
          animation: dbAurora 12s ease-in-out infinite alternate;
        }
        @keyframes dbAurora {
          0%   { opacity: 0.7; }
          100% { opacity: 1; }
        }

        .db-noise {
          position: fixed; inset: 0;
          background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)' opacity='1'/%3E%3C/svg%3E");
          opacity: 0.03; pointer-events: none; z-index: 0;
        }

        .db-scanline {
          position: fixed; top: -100%; left: 0; width: 100%; height: 2px;
          background: linear-gradient(90deg, transparent, rgba(79,142,247,0.12), transparent);
          animation: dbScan 10s linear infinite;
          z-index: 1; pointer-events: none;
        }
        @keyframes dbScan { to { top: 110%; } }

        .db-inner {
          position: relative; z-index: 2;
          max-width: 960px; margin: 0 auto;
          display: flex; flex-direction: column; gap: 20px;
        }

        /* HERO */
        .db-hero {
          border-radius: 20px;
          background: linear-gradient(145deg, #0d1530 0%, #0a0f20 60%, #12082e 100%);
          border: 1px solid rgba(255,255,255,0.07);
          padding: 36px;
          position: relative; overflow: hidden;
          box-shadow: 0 20px 60px rgba(0,0,0,0.5), 0 0 80px rgba(79,70,229,0.07);
          animation: dbFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) both;
        }
        .db-hero::before {
          content: '';
          position: absolute; top: -80px; left: -80px;
          width: 300px; height: 300px;
          background: radial-gradient(circle, rgba(79,142,247,0.15) 0%, transparent 70%);
          border-radius: 50%;
          animation: dbPulse 5s ease-in-out infinite;
        }
        .db-hero::after {
          content: '';
          position: absolute; bottom: -60px; right: -40px;
          width: 240px; height: 240px;
          background: radial-gradient(circle, rgba(167,139,250,0.12) 0%, transparent 70%);
          border-radius: 50%;
          animation: dbPulse 6s ease-in-out infinite 1.5s;
        }
        @keyframes dbPulse {
          0%,100% { transform: scale(1); opacity: 0.7; }
          50%      { transform: scale(1.15); opacity: 1; }
        }

        .db-hero-content { position: relative; z-index: 2; }

        .db-badge {
          display: inline-flex; align-items: center; gap: 6px;
          font-size: 11px; font-weight: 600; letter-spacing: 1.5px; text-transform: uppercase;
          color: #4f8ef7; margin-bottom: 14px;
        }
        .db-badge::before {
          content: ''; width: 6px; height: 6px; border-radius: 50%;
          background: #4f8ef7; box-shadow: 0 0 8px #4f8ef7;
          animation: dbBlink 1.5s ease-in-out infinite;
        }
        @keyframes dbBlink { 0%,100% { opacity: 1; } 50% { opacity: 0.3; } }

        .db-title {
          font-family: 'Syne', sans-serif;
          font-size: clamp(28px, 4vw, 42px);
          font-weight: 800; letter-spacing: -1px; line-height: 1.1;
          background: linear-gradient(135deg, #e8eaf6 30%, #a78bfa);
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          background-clip: text;
        }

        .db-subtitle {
          margin-top: 10px; font-size: 14px; color: #7986a3; font-weight: 300;
        }
        .db-subtitle strong { color: #818cf8; font-weight: 500; }

        .db-meta-row {
          margin-top: 20px; display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
        }

        .db-chip {
          display: inline-flex; align-items: center; gap: 6px;
          padding: 6px 12px; border-radius: 8px;
          font-size: 12px; font-weight: 500;
          border: 1px solid rgba(255,255,255,0.08);
          background: rgba(255,255,255,0.04); color: #a0aec0;
        }
        .db-chip.role {
          background: rgba(99,102,241,0.1);
          border-color: rgba(99,102,241,0.25); color: #c7d2fe;
        }

        /* PANELS */
        .db-panel {
          border-radius: 16px;
          background: rgba(255,255,255,0.025);
          border: 1px solid rgba(255,255,255,0.07);
          padding: 24px;
          backdrop-filter: blur(8px);
          animation: dbFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) both;
        }
        @keyframes dbFadeUp {
          from { opacity: 0; transform: translateY(20px); }
          to   { opacity: 1; transform: translateY(0); }
        }
        .db-panel:nth-child(2) { animation-delay: 0.1s; }
        .db-panel:nth-child(3) { animation-delay: 0.2s; }
        .db-panel:nth-child(4) { animation-delay: 0.3s; }

        .db-panel-title {
          font-family: 'Syne', sans-serif;
          font-size: 17px; font-weight: 700; color: #e8eaf6;
          margin-bottom: 16px;
          display: flex; align-items: center; gap: 8px;
        }
        .db-panel-title-icon {
          width: 30px; height: 30px; border-radius: 8px;
          display: flex; align-items: center; justify-content: center;
          font-size: 14px; flex-shrink: 0;
        }
        .db-panel-title-icon.blue   { background: rgba(79,142,247,0.15); }
        .db-panel-title-icon.purple { background: rgba(167,139,250,0.15); }

        /* FORM */
        .db-form-row { display: flex; flex-direction: column; gap: 10px; }
        @media (min-width: 560px) {
          .db-form-row { flex-direction: row; align-items: center; }
        }

        .db-input-wrap { position: relative; flex: 1; }
        .db-input-icon {
          position: absolute; left: 13px; top: 50%; transform: translateY(-50%);
          font-size: 15px; opacity: 0.45; pointer-events: none;
        }
        .db-input {
          width: 100%; box-sizing: border-box;
          border-radius: 11px; border: 1px solid rgba(255,255,255,0.08);
          background: rgba(255,255,255,0.04); color: #e8eaf6;
          font-size: 14px; font-family: 'DM Sans', sans-serif;
          padding: 12px 14px 12px 40px; outline: none;
          transition: border-color 0.2s, box-shadow 0.2s, background 0.2s;
        }
        .db-input::placeholder { color: rgba(107,122,153,0.6); }
        .db-input:focus {
          border-color: rgba(99,102,241,0.55);
          background: rgba(99,102,241,0.06);
          box-shadow: 0 0 0 3px rgba(99,102,241,0.12);
        }

        .db-btn {
          border: none; border-radius: 11px; padding: 12px 22px;
          cursor: pointer; color: #fff;
          font-family: 'Syne', sans-serif; font-size: 14px; font-weight: 600;
          background: linear-gradient(135deg, #2563eb, #4f46e5, #7c3aed);
          background-size: 200% 200%;
          animation: dbGradShift 4s ease infinite;
          transition: transform 0.15s, box-shadow 0.15s;
          box-shadow: 0 4px 20px rgba(79,70,229,0.4);
          white-space: nowrap; position: relative; overflow: hidden;
        }
        @keyframes dbGradShift {
          0%   { background-position: 0% 50%; }
          50%  { background-position: 100% 50%; }
          100% { background-position: 0% 50%; }
        }
        .db-btn::before {
          content: ''; position: absolute; top: 0; left: -100%;
          width: 100%; height: 100%;
          background: linear-gradient(90deg, transparent, rgba(255,255,255,0.1), transparent);
          transition: left 0.4s;
        }
        .db-btn:hover::before { left: 100%; }
        .db-btn:hover { transform: translateY(-1px); box-shadow: 0 8px 28px rgba(79,70,229,0.55); }
        .db-btn:active { transform: scale(0.98); }

        .db-action-msg {
          margin-top: 10px; font-size: 13px; font-weight: 500;
          padding: 8px 12px; border-radius: 8px;
        }
        .db-action-msg.success { color: #6ee7b7; background: rgba(16,185,129,0.08); border: 1px solid rgba(16,185,129,0.2); }
        .db-action-msg.error   { color: #fca5a5; background: rgba(239,68,68,0.08);  border: 1px solid rgba(239,68,68,0.2); }

        /* COURSES */
        .db-courses-grid { display: grid; grid-template-columns: 1fr; gap: 10px; }
        @media (min-width: 560px) { .db-courses-grid { grid-template-columns: repeat(2, 1fr); } }
        @media (min-width: 860px) { .db-courses-grid { grid-template-columns: repeat(3, 1fr); } }

        .db-course-card {
          border-radius: 13px; border: 1px solid rgba(255,255,255,0.07);
          background: rgba(255,255,255,0.025);
          padding: 16px 18px; text-decoration: none; display: block;
          transition: border-color 0.2s, background 0.2s, transform 0.18s, box-shadow 0.18s;
          position: relative; overflow: hidden;
        }
        .db-course-card::before {
          content: ''; position: absolute;
          top: 0; left: 0; right: 0; height: 2px;
          background: linear-gradient(90deg, #4f46e5, #7c3aed);
          opacity: 0; transition: opacity 0.2s;
        }
        .db-course-card:hover {
          border-color: rgba(99,102,241,0.35);
          background: rgba(99,102,241,0.07);
          transform: translateY(-3px);
          box-shadow: 0 12px 32px rgba(0,0,0,0.3);
        }
        .db-course-card:hover::before { opacity: 1; }

        .db-course-name { color: #e8eaf6; font-size: 15px; font-weight: 600; margin: 0 0 6px; }
        .db-course-code {
          color: #6b7a99; font-size: 11px;
          font-family: 'Courier New', monospace; letter-spacing: 0.08em;
          background: rgba(255,255,255,0.04); border: 1px solid rgba(255,255,255,0.06);
          border-radius: 5px; padding: 2px 7px; display: inline-block;
        }
        .db-course-arrow {
          position: absolute; right: 14px; top: 50%; transform: translateY(-50%);
          color: #4f46e5; font-size: 16px; opacity: 0;
          transition: opacity 0.2s, right 0.2s;
        }
        .db-course-card:hover .db-course-arrow { opacity: 1; right: 12px; }
        .db-course-actions {
          margin-top: 10px;
          display: flex;
          justify-content: flex-end;
        }
        .db-delete-btn {
          border: 1px solid rgba(239,68,68,0.35);
          background: rgba(239,68,68,0.08);
          color: #fca5a5;
          border-radius: 8px;
          padding: 6px 10px;
          font-size: 12px;
          cursor: pointer;
          transition: background 0.15s, border-color 0.15s, opacity 0.15s;
        }
        .db-delete-btn:hover {
          background: rgba(239,68,68,0.14);
          border-color: rgba(239,68,68,0.5);
        }
        .db-delete-btn:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        /* LOADING */
        .db-muted { font-size: 13px; color: #6b7a99; padding: 10px 0; }
        .db-loading-dots { display: inline-flex; gap: 4px; margin-left: 4px; }
        .db-loading-dots span {
          width: 5px; height: 5px; border-radius: 50%;
          background: #4f8ef7; display: inline-block;
          animation: dbDot 1.2s ease-in-out infinite;
        }
        .db-loading-dots span:nth-child(2) { animation-delay: 0.2s; }
        .db-loading-dots span:nth-child(3) { animation-delay: 0.4s; }
        @keyframes dbDot {
          0%,80%,100% { opacity: 0.2; transform: scale(0.8); }
          40%          { opacity: 1;   transform: scale(1); }
        }

      `}</style>

      <div className="db-root">
        <div className="db-noise" />
        <div className="db-scanline" />

        <div className="db-inner">

          {/* HERO */}
          <div className="db-hero">
            <div className="db-hero-content">
              <div className="db-badge">Dashboard</div>
              <h1 className="db-title">{greeting}, {firstName} 👋</h1>
              <p className="db-subtitle">
                Ready to continue your journey with <strong>AI Teaching Assistant</strong>.
              </p>
              <div className="db-meta-row">
                <span className="db-chip role">
                  {user?.role === 'professor' ? '🎓' : '📚'} {user?.role || 'unknown'}
                </span>
                <span className="db-chip">📧 {user?.email || '—'}</span>
              </div>
            </div>
          </div>

          {/* ACTION PANEL */}
          <div className="db-panel">
            <h2 className="db-panel-title">
              <span className={`db-panel-title-icon ${user?.role === 'professor' ? 'blue' : 'purple'}`}>
                {user?.role === 'professor' ? '➕' : '🔗'}
              </span>
              {user?.role === 'professor' ? 'Create a Course' : 'Join a Course'}
            </h2>

            {user?.role === 'professor' && (
              <div className="db-form-row">
                <div className="db-input-wrap">
                  <span className="db-input-icon">📖</span>
                  <input
                    type="text"
                    value={courseName}
                    onChange={(e) => setCourseName(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleCreateCourse()}
                    placeholder="Course name"
                    className="db-input"
                  />
                </div>
                <button onClick={handleCreateCourse} className="db-btn">Create Course →</button>
              </div>
            )}

            {user?.role === 'student' && (
              <div className="db-form-row">
                <div className="db-input-wrap">
                  <span className="db-input-icon">🔑</span>
                  <input
                    type="text"
                    value={courseCode}
                    onChange={(e) => setCourseCode(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleJoinCourse()}
                    placeholder="Course code (e.g. ABC123)"
                    className="db-input"
                  />
                </div>
                <button onClick={handleJoinCourse} className="db-btn">Join Course →</button>
              </div>
            )}

            {actionMessage && (
              <p className={`db-action-msg ${actionSuccess ? 'success' : 'error'}`}>
                {actionSuccess ? '✅' : '⚠️'} {actionMessage}
              </p>
            )}
          </div>

          {/* COURSES */}
          <div className="db-panel">
            <h2 className="db-panel-title">
              <span className="db-panel-title-icon purple">🗂️</span>
              Courses
            </h2>

            {loading && (
              <p className="db-muted">
                Loading courses
                <span className="db-loading-dots"><span /><span /><span /></span>
              </p>
            )}
            {error && <p className="db-muted" style={{ color: '#fca5a5' }}>{error}</p>}
            {!loading && !error && courses.length === 0 && (
              <p className="db-muted">
                No courses yet.{' '}
                {user?.role === 'professor' ? 'Create your first one above.' : 'Join one using a course code.'}
              </p>
            )}
            {!loading && !error && courses.length > 0 && (
              <div className="db-courses-grid">
                {courses.map((course) => (
                  <div key={course.id} className="db-course-card">
                    <Link to={`/courses/${course.id}`} style={{ display: 'block', textDecoration: 'none' }}>
                      <h3 className="db-course-name">{course.name}</h3>
                      <span className="db-course-code">{course.course_code}</span>
                      <span className="db-course-arrow">?</span>
                    </Link>
                    {user?.role === 'professor' ? (
                      <div className="db-course-actions">
                        <button
                          className="db-delete-btn"
                          onClick={() => handleDeleteCourse(course.id, course.name)}
                          disabled={deletingCourseID === course.id}
                        >
                          {deletingCourseID === course.id ? 'Deleting...' : 'Delete'}
                        </button>
                      </div>
                    ) : null}
                  </div>
                ))}
              </div>
            )}
          </div>

        </div>
      </div>
    </Layout>
  )
}

export default DashboardPage
