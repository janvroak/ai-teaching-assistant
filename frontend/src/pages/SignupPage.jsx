import { useState, useEffect, useRef } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import api from '../services/api'

function SignupPage() {
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('student')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)
  const canvasRef = useRef(null)

  useEffect(() => {
    const canvas = canvasRef.current
    const ctx = canvas.getContext('2d')
    let animId
    let t = 0
    let stars = []

    const orbs = [
      { x: 0.15, y: 0.2, r: 320, color: 'rgba(37,99,235,0.18)', vx: 0.00015, vy: 0.0001 },
      { x: 0.8, y: 0.3, r: 260, color: 'rgba(124,58,237,0.14)', vx: -0.00012, vy: 0.00015 },
      { x: 0.5, y: 0.85, r: 280, color: 'rgba(79,142,247,0.12)', vx: 0.0001, vy: -0.00012 },
      { x: 0.9, y: 0.75, r: 200, color: 'rgba(167,139,250,0.10)', vx: -0.0001, vy: -0.0001 },
    ]

    function resize() {
      canvas.width = window.innerWidth
      canvas.height = window.innerHeight
    }
    resize()
    window.addEventListener('resize', resize)

    stars = Array.from({ length: 80 }, () => ({
      x: Math.random(), y: Math.random(),
      r: Math.random() * 1.2 + 0.3,
      a: Math.random(),
      speed: Math.random() * 0.005 + 0.002,
      phase: Math.random() * Math.PI * 2,
    }))

    function draw() {
      t++
      ctx.clearRect(0, 0, canvas.width, canvas.height)
      ctx.fillStyle = '#05060f'
      ctx.fillRect(0, 0, canvas.width, canvas.height)

      ctx.strokeStyle = 'rgba(79,142,247,0.025)'
      ctx.lineWidth = 1
      const step = 60
      for (let x = 0; x < canvas.width; x += step) {
        ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, canvas.height); ctx.stroke()
      }
      for (let y = 0; y < canvas.height; y += step) {
        ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(canvas.width, y); ctx.stroke()
      }

      orbs.forEach((o) => {
        o.x += o.vx * Math.sin(t * 0.01)
        o.y += o.vy * Math.cos(t * 0.013)
        if (o.x < -0.1) o.x = 1.1
        if (o.x > 1.1) o.x = -0.1
        if (o.y < -0.1) o.y = 1.1
        if (o.y > 1.1) o.y = -0.1
        const gx = o.x * canvas.width
        const gy = o.y * canvas.height
        const grad = ctx.createRadialGradient(gx, gy, 0, gx, gy, o.r)
        grad.addColorStop(0, o.color)
        grad.addColorStop(1, 'transparent')
        ctx.fillStyle = grad
        ctx.beginPath()
        ctx.arc(gx, gy, o.r, 0, Math.PI * 2)
        ctx.fill()
      })

      stars.forEach((s) => {
        const alpha = (Math.sin(t * s.speed + s.phase) * 0.5 + 0.5) * s.a * 0.7
        ctx.fillStyle = `rgba(200,210,255,${alpha})`
        ctx.beginPath()
        ctx.arc(s.x * canvas.width, s.y * canvas.height, s.r, 0, Math.PI * 2)
        ctx.fill()
      })

      animId = requestAnimationFrame(draw)
    }
    draw()

    return () => {
      cancelAnimationFrame(animId)
      window.removeEventListener('resize', resize)
    }
  }, [])

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setSuccess('')
    setLoading(true)
    try {
      await api.post('/signup', { name, email, password, role })
      setSuccess('Account created! Redirecting to login...')
      setTimeout(() => navigate('/login', { replace: true }), 700)
    } catch (err) {
      const message =
        err?.response?.data?.error ||
        (err?.request ? 'Cannot reach backend API. Check backend server and CORS settings.' : '') ||
        err?.message ||
        'Signup failed'
      setError(message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=Syne:wght@400;600;700;800&family=DM+Sans:ital,wght@0,300;0,400;0,500;1,300&display=swap');

        .su-root {
          font-family: 'DM Sans', sans-serif;
          min-height: 100vh;
          display: flex;
          align-items: center;
          justify-content: center;
          position: relative;
          overflow: hidden;
          background: #05060f;
        }
        .su-root::after {
          content: '';
          position: fixed;
          inset: 0;
          background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)' opacity='1'/%3E%3C/svg%3E");
          opacity: 0.035;
          z-index: 1;
          pointer-events: none;
        }
        .su-canvas {
          position: fixed;
          inset: 0;
          z-index: 0;
          pointer-events: none;
        }
        .su-scanline {
          position: fixed;
          top: -100%;
          left: 0;
          width: 100%;
          height: 2px;
          background: linear-gradient(90deg, transparent, rgba(79,142,247,0.15), transparent);
          animation: suScan 8s linear infinite;
          z-index: 2;
          pointer-events: none;
        }
        @keyframes suScan { to { top: 110%; } }

        .su-card {
          position: relative;
          z-index: 10;
          width: min(900px, 95vw);
          display: grid;
          grid-template-columns: 1fr 1fr;
          border-radius: 24px;
          overflow: hidden;
          box-shadow: 0 0 0 1px rgba(255,255,255,0.08), 0 40px 80px rgba(0,0,0,0.6), 0 0 120px rgba(79,142,247,0.08);
          backdrop-filter: blur(24px);
          animation: suFadeUp 0.8s cubic-bezier(0.16,1,0.3,1) both;
        }
        @keyframes suFadeUp {
          from { opacity: 0; transform: translateY(32px) scale(0.97); }
          to   { opacity: 1; transform: translateY(0) scale(1); }
        }

        /* LEFT PANEL */
        .su-left {
          background: linear-gradient(145deg, #0d1530 0%, #0a0f20 60%, #12082e 100%);
          padding: 52px 44px;
          display: flex;
          flex-direction: column;
          justify-content: space-between;
          position: relative;
          overflow: hidden;
          border-right: 1px solid rgba(255,255,255,0.08);
        }
        .su-left::before {
          content: '';
          position: absolute;
          top: -60px; left: -60px;
          width: 280px; height: 280px;
          background: radial-gradient(circle, rgba(79,142,247,0.18) 0%, transparent 70%);
          border-radius: 50%;
          animation: suPulse 4s ease-in-out infinite;
        }
        .su-left::after {
          content: '';
          position: absolute;
          bottom: -40px; right: -40px;
          width: 220px; height: 220px;
          background: radial-gradient(circle, rgba(167,139,250,0.15) 0%, transparent 70%);
          border-radius: 50%;
          animation: suPulse 5s ease-in-out infinite 1.5s;
        }
        @keyframes suPulse {
          0%,100% { transform: scale(1); opacity: 0.7; }
          50%      { transform: scale(1.15); opacity: 1; }
        }

        .su-brand { position: relative; z-index: 2; }
        .su-brand-icon {
          width: 48px; height: 48px;
          background: linear-gradient(135deg, #4f8ef7, #a78bfa);
          border-radius: 14px;
          display: flex; align-items: center; justify-content: center;
          font-size: 22px;
          margin-bottom: 24px;
          box-shadow: 0 0 24px rgba(79,142,247,0.4);
        }
        .su-brand h1 {
          font-family: 'Syne', sans-serif;
          font-size: 26px; font-weight: 800;
          line-height: 1.2; letter-spacing: -0.5px;
          background: linear-gradient(135deg, #e8eaf6 30%, #a78bfa);
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          background-clip: text;
        }
        .su-brand p {
          margin-top: 12px;
          font-size: 13.5px; color: #7986a3;
          line-height: 1.6; font-weight: 300;
        }

        .su-steps { position: relative; z-index: 2; display: flex; flex-direction: column; gap: 0; }
        .su-step {
          display: flex; gap: 16px;
          animation: suFadeUp 0.6s cubic-bezier(0.16,1,0.3,1) both;
        }
        .su-step:nth-child(1) { animation-delay: 0.2s; }
        .su-step:nth-child(2) { animation-delay: 0.35s; }
        .su-step:nth-child(3) { animation-delay: 0.5s; }
        .su-step-left { display: flex; flex-direction: column; align-items: center; }
        .su-step-num {
          width: 32px; height: 32px; border-radius: 50%;
          border: 1px solid rgba(79,142,247,0.4);
          display: flex; align-items: center; justify-content: center;
          font-family: 'Syne', sans-serif; font-size: 13px; font-weight: 700;
          color: #4f8ef7; flex-shrink: 0;
        }
        .su-step-line {
          flex: 1; width: 1px;
          background: linear-gradient(to bottom, rgba(79,142,247,0.3), transparent);
          margin: 6px 0;
          min-height: 28px;
        }
        .su-step:last-child .su-step-line { display: none; }
        .su-step-content { padding-bottom: 24px; }
        .su-step-content h3 {
          font-family: 'Syne', sans-serif; font-size: 13.5px; font-weight: 600;
          color: #e8eaf6; margin-bottom: 4px;
        }
        .su-step-content p { font-size: 12.5px; color: #7986a3; line-height: 1.5; }

        /* RIGHT PANEL */
        .su-right {
          background: rgba(8,10,22,0.92);
          padding: 48px 44px;
          display: flex; flex-direction: column; justify-content: center;
          color: #e8eaf6;
          overflow-y: auto;
        }

        .su-welcome { margin-bottom: 28px; animation: suFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) 0.15s both; }
        .su-welcome-tag {
          display: inline-flex; align-items: center; gap: 6px;
          font-size: 11px; font-weight: 500; letter-spacing: 1.5px; text-transform: uppercase;
          color: #a78bfa; margin-bottom: 12px;
        }
        .su-welcome-tag::before {
          content: ''; display: block;
          width: 6px; height: 6px; border-radius: 50%; background: #a78bfa;
          animation: suBlink 1.5s ease-in-out infinite;
        }
        @keyframes suBlink { 0%,100% { opacity: 1; } 50% { opacity: 0.3; } }
        .su-welcome h2 {
          font-family: 'Syne', sans-serif;
          font-size: 28px; font-weight: 700; letter-spacing: -0.5px; line-height: 1.1;
        }
        .su-welcome p { margin-top: 8px; font-size: 13.5px; color: #7986a3; font-weight: 300; }

        .su-form-group {
          display: flex; flex-direction: column; gap: 6px;
          margin-bottom: 16px;
          animation: suFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) both;
        }
        .su-form-group:nth-child(1) { animation-delay: 0.2s; }
        .su-form-group:nth-child(2) { animation-delay: 0.3s; }
        .su-form-group:nth-child(3) { animation-delay: 0.4s; }
        .su-form-group:nth-child(4) { animation-delay: 0.5s; }

        .su-label { font-size: 12px; font-weight: 500; letter-spacing: 0.5px; color: #7986a3; text-transform: uppercase; }
        .su-input-wrap { position: relative; }
        .su-input-icon {
          position: absolute; left: 14px; top: 50%; transform: translateY(-50%);
          font-size: 15px; opacity: 0.5; pointer-events: none;
        }
        .su-input, .su-select {
          width: 100%;
          background: rgba(255,255,255,0.04);
          border: 1px solid rgba(255,255,255,0.08);
          border-radius: 12px;
          padding: 12px 14px 12px 42px;
          font-family: 'DM Sans', sans-serif;
          font-size: 14px; color: #e8eaf6;
          outline: none;
          transition: border-color 0.25s, box-shadow 0.25s, background 0.25s;
          box-sizing: border-box;
        }
        .su-select { appearance: none; cursor: pointer; }
        .su-input::placeholder { color: rgba(121,134,163,0.5); }
        .su-input:focus, .su-select:focus {
          border-color: #a78bfa;
          background: rgba(167,139,250,0.06);
          box-shadow: 0 0 0 3px rgba(167,139,250,0.12), inset 0 1px 0 rgba(255,255,255,0.05);
        }
        .su-select option { background: #0a0f20; color: #e8eaf6; }
        .su-select-arrow {
          position: absolute; right: 14px; top: 50%; transform: translateY(-50%);
          font-size: 11px; color: #7986a3; pointer-events: none;
        }

        /* Role toggle */
        .su-role-toggle {
          display: grid; grid-template-columns: 1fr 1fr;
          gap: 8px; margin-top: 2px;
        }
        .su-role-btn {
          padding: 10px 12px;
          background: rgba(255,255,255,0.04);
          border: 1px solid rgba(255,255,255,0.08);
          border-radius: 10px;
          font-family: 'DM Sans', sans-serif; font-size: 13px;
          color: #7986a3; cursor: pointer;
          display: flex; align-items: center; justify-content: center; gap: 8px;
          transition: all 0.2s;
        }
        .su-role-btn.active {
          border-color: #a78bfa;
          background: rgba(167,139,250,0.12);
          color: #e8eaf6;
          box-shadow: 0 0 0 1px rgba(167,139,250,0.3);
        }

        .su-alert {
          padding: 10px 14px; border-radius: 10px;
          font-size: 13px; margin-bottom: 14px;
          animation: suFadeUp 0.3s ease both;
        }
        .su-alert.error  { background: rgba(239,68,68,0.1);  border: 1px solid rgba(239,68,68,0.25);  color: #fca5a5; }
        .su-alert.success{ background: rgba(34,197,94,0.1);  border: 1px solid rgba(34,197,94,0.25);  color: #86efac; }

        .su-btn {
          width: 100%; padding: 14px; border: none; border-radius: 12px;
          font-family: 'Syne', sans-serif; font-size: 15px; font-weight: 600; letter-spacing: 0.3px;
          color: #fff; cursor: pointer; position: relative; overflow: hidden;
          background: linear-gradient(135deg, #4f46e5, #7c3aed, #a78bfa);
          background-size: 200% 200%;
          animation: suGradShift 4s ease infinite, suFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) 0.55s both;
          transition: transform 0.15s, box-shadow 0.15s, opacity 0.2s;
          box-shadow: 0 4px 24px rgba(124,58,237,0.4);
          margin-top: 4px;
        }
        @keyframes suGradShift {
          0%   { background-position: 0% 50%; }
          50%  { background-position: 100% 50%; }
          100% { background-position: 0% 50%; }
        }
        .su-btn::before {
          content: ''; position: absolute; top: 0; left: -100%;
          width: 100%; height: 100%;
          background: linear-gradient(90deg, transparent, rgba(255,255,255,0.12), transparent);
          transition: left 0.5s;
        }
        .su-btn:hover::before { left: 100%; }
        .su-btn:hover { transform: translateY(-1px); box-shadow: 0 8px 32px rgba(124,58,237,0.5); }
        .su-btn:active { transform: translateY(0) scale(0.99); }
        .su-btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none; }

        .su-login-row {
          margin-top: 22px; text-align: center; font-size: 13px; color: #7986a3;
          animation: suFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) 0.7s both;
        }
        .su-login-row a { color: #a78bfa; text-decoration: none; font-weight: 500; transition: opacity 0.2s; }
        .su-login-row a:hover { opacity: 0.8; }

        @media (max-width: 620px) {
          .su-card { grid-template-columns: 1fr; }
          .su-left { display: none; }
          .su-right { padding: 40px 28px; }
        }
      `}</style>

      <div className="su-root">
        <canvas ref={canvasRef} className="su-canvas" />
        <div className="su-scanline" />

        <div className="su-card">
          {/* LEFT PANEL */}
          <div className="su-left">
            <div className="su-brand">
              <div className="su-brand-icon">🧠</div>
              <h1>AI Teaching<br />Assistant</h1>
              <p>Join thousands of learners and educators on the smarter path forward.</p>
            </div>
            <div className="su-steps">
              <div className="su-step">
                <div className="su-step-left">
                  <div className="su-step-num">1</div>
                  <div className="su-step-line" />
                </div>
                <div className="su-step-content">
                  <h3>Create your account</h3>
                  <p>Fill in your details and choose your role to get started.</p>
                </div>
              </div>
              <div className="su-step">
                <div className="su-step-left">
                  <div className="su-step-num">2</div>
                  <div className="su-step-line" />
                </div>
                <div className="su-step-content">
                  <h3>Set up your profile</h3>
                  <p>Personalize your learning preferences and goals.</p>
                </div>
              </div>
              <div className="su-step">
                <div className="su-step-left">
                  <div className="su-step-num">3</div>
                  <div className="su-step-line" />
                </div>
                <div className="su-step-content">
                  <h3>Start learning</h3>
                  <p>Access AI-powered lessons tailored just for you.</p>
                </div>
              </div>
            </div>
          </div>

          {/* RIGHT PANEL */}
          <div className="su-right">
            <div className="su-welcome">
              <div className="su-welcome-tag">Get Started</div>
              <h2>Create Account</h2>
              <p>Join the platform and begin your journey.</p>
            </div>

            <div className="su-form-group">
              <label className="su-label">Full Name</label>
              <div className="su-input-wrap">
                <input
                  type="text"
                  className="su-input bg-[#1F2937] text-white placeholder-gray-400 border border-gray-700"
                  placeholder="Jane Smith"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                />
                <span className="su-input-icon">👤</span>
              </div>
            </div>

            <div className="su-form-group">
              <label className="su-label">Email Address</label>
              <div className="su-input-wrap">
                <input
                  type="email"
                  className="su-input bg-[#1F2937] text-white placeholder-gray-400 border border-gray-700"
                  placeholder="you@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                />
                <span className="su-input-icon">📧</span>
              </div>
            </div>

            <div className="su-form-group">
              <label className="su-label">Password</label>
              <div className="su-input-wrap">
                <input
                  type="password"
                  className="su-input bg-[#1F2937] text-white placeholder-gray-400 border border-gray-700"
                  placeholder="••••••••"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
                <span className="su-input-icon">🔑</span>
              </div>
            </div>

            <div className="su-form-group">
              <label className="su-label">I am a</label>
              <div className="su-role-toggle">
                <button
                  type="button"
                  className={`su-role-btn ${role === 'student' ? 'active' : ''}`}
                  onClick={() => setRole('student')}
                >
                  🎓 Student
                </button>
                <button
                  type="button"
                  className={`su-role-btn ${role === 'professor' ? 'active' : ''}`}
                  onClick={() => setRole('professor')}
                >
                  🏫 Professor
                </button>
              </div>
            </div>

            {error   && <div className="su-alert error"  >⚠ {error}</div>}
            {success && <div className="su-alert success">✓ {success}</div>}

            <button
              className="su-btn"
              disabled={loading}
              onClick={handleSubmit}
            >
              {loading ? 'Creating account…' : 'Create Account →'}
            </button>

            <div className="su-login-row">
              Already have an account?{' '}
              <Link to="/login">Sign in →</Link>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

export default SignupPage
