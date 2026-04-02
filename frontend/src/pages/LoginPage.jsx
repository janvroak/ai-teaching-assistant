import { useState, useEffect, useRef } from "react";
import { Link, useNavigate } from "react-router-dom";
import api from "../services/api";

export default function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const canvasRef = useRef(null);
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const response = await api.post("/login", { email, password });
      const token = response?.data?.token;

      if (!token) {
        setError("Invalid response from server");
        return;
      }

      localStorage.setItem("token", token);

      if (response?.data?.user) {
        localStorage.setItem("user", JSON.stringify(response.data.user));
      }

      navigate("/dashboard");
    } catch (err) {
      const message = err?.response?.data?.error || "Login failed";
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const canvas = canvasRef.current;
    const ctx = canvas.getContext("2d");
    let animId;
    let t = 0;
    let stars = [];

    const orbs = [
      { x: 0.15, y: 0.2, r: 320, color: "rgba(37,99,235,0.18)", vx: 0.00015, vy: 0.0001 },
      { x: 0.8, y: 0.3, r: 260, color: "rgba(124,58,237,0.14)", vx: -0.00012, vy: 0.00015 },
      { x: 0.5, y: 0.85, r: 280, color: "rgba(79,142,247,0.12)", vx: 0.0001, vy: -0.00012 },
      { x: 0.9, y: 0.75, r: 200, color: "rgba(167,139,250,0.10)", vx: -0.0001, vy: -0.0001 },
    ];

    function resize() {
      canvas.width = window.innerWidth;
      canvas.height = window.innerHeight;
    }
    resize();
    window.addEventListener("resize", resize);

    stars = Array.from({ length: 80 }, () => ({
      x: Math.random(), y: Math.random(),
      r: Math.random() * 1.2 + 0.3,
      a: Math.random(),
      speed: Math.random() * 0.005 + 0.002,
      phase: Math.random() * Math.PI * 2,
    }));

    function draw() {
      t++;
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      ctx.fillStyle = "#05060f";
      ctx.fillRect(0, 0, canvas.width, canvas.height);

      ctx.strokeStyle = "rgba(79,142,247,0.025)";
      ctx.lineWidth = 1;
      const step = 60;
      for (let x = 0; x < canvas.width; x += step) {
        ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, canvas.height); ctx.stroke();
      }
      for (let y = 0; y < canvas.height; y += step) {
        ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(canvas.width, y); ctx.stroke();
      }

      orbs.forEach((o) => {
        o.x += o.vx * Math.sin(t * 0.01);
        o.y += o.vy * Math.cos(t * 0.013);
        if (o.x < -0.1) o.x = 1.1;
        if (o.x > 1.1) o.x = -0.1;
        if (o.y < -0.1) o.y = 1.1;
        if (o.y > 1.1) o.y = -0.1;
        const gx = o.x * canvas.width;
        const gy = o.y * canvas.height;
        const grad = ctx.createRadialGradient(gx, gy, 0, gx, gy, o.r);
        grad.addColorStop(0, o.color);
        grad.addColorStop(1, "transparent");
        ctx.fillStyle = grad;
        ctx.beginPath();
        ctx.arc(gx, gy, o.r, 0, Math.PI * 2);
        ctx.fill();
      });

      stars.forEach((s) => {
        const alpha = (Math.sin(t * s.speed + s.phase) * 0.5 + 0.5) * s.a * 0.7;
        ctx.fillStyle = `rgba(200,210,255,${alpha})`;
        ctx.beginPath();
        ctx.arc(s.x * canvas.width, s.y * canvas.height, s.r, 0, Math.PI * 2);
        ctx.fill();
      });

      animId = requestAnimationFrame(draw);
    }
    draw();

    return () => {
      cancelAnimationFrame(animId);
      window.removeEventListener("resize", resize);
    };
  }, []);

  return (
    <>
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=Syne:wght@400;600;700;800&family=DM+Sans:ital,wght@0,300;0,400;0,500;1,300&display=swap');

        .login-root {
          font-family: 'DM Sans', sans-serif;
          min-height: 100vh;
          display: flex;
          align-items: center;
          justify-content: center;
          position: relative;
          overflow: hidden;
          background: #05060f;
        }
        .login-root::after {
          content: '';
          position: fixed;
          inset: 0;
          background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)' opacity='1'/%3E%3C/svg%3E");
          opacity: 0.035;
          z-index: 1;
          pointer-events: none;
        }
        .login-canvas {
          position: fixed; inset: 0; z-index: 0; pointer-events: none;
        }
        .login-scanline {
          position: fixed; top: -100%; left: 0; width: 100%; height: 2px;
          background: linear-gradient(90deg, transparent, rgba(79,142,247,0.15), transparent);
          animation: loginScan 8s linear infinite; z-index: 2; pointer-events: none;
        }
        @keyframes loginScan { to { top: 110%; } }

        .login-card {
          position: relative; z-index: 10;
          width: min(900px, 95vw);
          display: grid; grid-template-columns: 1fr 1fr;
          border-radius: 24px; overflow: hidden;
          box-shadow: 0 0 0 1px rgba(255,255,255,0.08), 0 40px 80px rgba(0,0,0,0.6), 0 0 120px rgba(79,142,247,0.08);
          backdrop-filter: blur(24px);
          animation: loginFadeUp 0.8s cubic-bezier(0.16,1,0.3,1) both;
        }
        @keyframes loginFadeUp {
          from { opacity: 0; transform: translateY(32px) scale(0.97); }
          to   { opacity: 1; transform: translateY(0) scale(1); }
        }

        .login-left {
          background: linear-gradient(145deg, #0d1530 0%, #0a0f20 60%, #12082e 100%);
          padding: 52px 44px;
          display: flex; flex-direction: column; justify-content: space-between;
          position: relative; overflow: hidden;
          border-right: 1px solid rgba(255,255,255,0.08);
        }
        .login-left::before {
          content: ''; position: absolute; top: -60px; left: -60px;
          width: 280px; height: 280px;
          background: radial-gradient(circle, rgba(79,142,247,0.18) 0%, transparent 70%);
          border-radius: 50%; animation: loginPulse 4s ease-in-out infinite;
        }
        .login-left::after {
          content: ''; position: absolute; bottom: -40px; right: -40px;
          width: 220px; height: 220px;
          background: radial-gradient(circle, rgba(167,139,250,0.15) 0%, transparent 70%);
          border-radius: 50%; animation: loginPulse 5s ease-in-out infinite 1.5s;
        }
        @keyframes loginPulse {
          0%,100% { transform: scale(1); opacity: 0.7; }
          50%      { transform: scale(1.15); opacity: 1; }
        }

        .login-brand { position: relative; z-index: 2; }
        .login-brand-icon {
          width: 48px; height: 48px;
          background: linear-gradient(135deg, #4f8ef7, #a78bfa);
          border-radius: 14px;
          display: flex; align-items: center; justify-content: center;
          font-size: 22px; margin-bottom: 24px;
          box-shadow: 0 0 24px rgba(79,142,247,0.4);
        }
        .login-brand h1 {
          font-family: 'Syne', sans-serif;
          font-size: 26px; font-weight: 800; line-height: 1.2; letter-spacing: -0.5px;
          background: linear-gradient(135deg, #e8eaf6 30%, #a78bfa);
          -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text;
        }
        .login-brand p { margin-top: 12px; font-size: 13.5px; color: #7986a3; line-height: 1.6; font-weight: 300; }

        .login-features { position: relative; z-index: 2; display: flex; flex-direction: column; gap: 14px; }
        .login-feature-item {
          display: flex; align-items: center; gap: 12px;
          font-size: 13px; color: #7986a3;
          animation: loginFadeUp 0.6s cubic-bezier(0.16,1,0.3,1) both;
        }
        .login-feature-item:nth-child(1) { animation-delay: 0.2s; }
        .login-feature-item:nth-child(2) { animation-delay: 0.35s; }
        .login-feature-item:nth-child(3) { animation-delay: 0.5s; }
        .login-feature-dot {
          width: 32px; height: 32px; border-radius: 10px;
          display: flex; align-items: center; justify-content: center;
          font-size: 15px; flex-shrink: 0;
        }
        .login-feature-dot.blue   { background: rgba(79,142,247,0.15); }
        .login-feature-dot.purple { background: rgba(167,139,250,0.15); }
        .login-feature-dot.pink   { background: rgba(244,114,182,0.15); }

        .login-right {
          background: rgba(8,10,22,0.92);
          padding: 52px 44px;
          display: flex; flex-direction: column; justify-content: center;
          color: #e8eaf6;
        }

        .login-welcome { margin-bottom: 36px; animation: loginFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) 0.15s both; }
        .login-welcome-tag {
          display: inline-flex; align-items: center; gap: 6px;
          font-size: 11px; font-weight: 500; letter-spacing: 1.5px; text-transform: uppercase;
          color: #4f8ef7; margin-bottom: 12px;
        }
        .login-welcome-tag::before {
          content: ''; display: block;
          width: 6px; height: 6px; border-radius: 50%; background: #4f8ef7;
          animation: loginBlink 1.5s ease-in-out infinite;
        }
        @keyframes loginBlink { 0%,100% { opacity: 1; } 50% { opacity: 0.3; } }
        .login-welcome h2 {
          font-family: 'Syne', sans-serif;
          font-size: 30px; font-weight: 700; letter-spacing: -0.5px; line-height: 1.1;
        }
        .login-welcome p { margin-top: 8px; font-size: 13.5px; color: #7986a3; font-weight: 300; }

        .login-form { display: flex; flex-direction: column; }

        .login-form-group {
          display: flex; flex-direction: column; gap: 6px; margin-bottom: 18px;
          animation: loginFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) both;
        }
        .login-form-group:nth-child(1) { animation-delay: 0.25s; }
        .login-form-group:nth-child(2) { animation-delay: 0.4s; }

        .login-label { font-size: 12px; font-weight: 500; letter-spacing: 0.5px; color: #7986a3; text-transform: uppercase; }
        .login-input-wrap { position: relative; }
        .login-input-icon {
          position: absolute; left: 14px; top: 50%; transform: translateY(-50%);
          font-size: 15px; opacity: 0.5; pointer-events: none; transition: opacity 0.2s;
        }
        .login-input {
          width: 100%; box-sizing: border-box;
          background: rgba(255,255,255,0.04);
          border: 1px solid rgba(255,255,255,0.08);
          border-radius: 12px;
          padding: 13px 14px 13px 42px;
          font-family: 'DM Sans', sans-serif;
          font-size: 14px; color: #e8eaf6; outline: none;
          transition: border-color 0.25s, box-shadow 0.25s, background 0.25s;
        }
        .login-input::placeholder { color: rgba(121,134,163,0.5); }
        .login-input:focus {
          border-color: #4f8ef7;
          background: rgba(79,142,247,0.06);
          box-shadow: 0 0 0 3px rgba(79,142,247,0.12), inset 0 1px 0 rgba(255,255,255,0.05);
        }

        .login-forgot {
          text-align: right; margin-top: -10px; margin-bottom: 8px;
          animation: loginFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) 0.5s both;
        }
        .login-forgot a { font-size: 12px; color: #4f8ef7; text-decoration: none; opacity: 0.8; transition: opacity 0.2s; }
        .login-forgot a:hover { opacity: 1; }

        .login-error {
          display: flex; align-items: center; gap: 8px;
          padding: 10px 14px; border-radius: 10px; margin-bottom: 14px;
          background: rgba(239,68,68,0.1); border: 1px solid rgba(239,68,68,0.25);
          color: #fca5a5; font-size: 13px;
          animation: loginFadeUp 0.3s ease both;
        }

        .login-btn {
          width: 100%; padding: 14px; border: none; border-radius: 12px;
          font-family: 'Syne', sans-serif; font-size: 15px; font-weight: 600; letter-spacing: 0.3px;
          color: #fff; cursor: pointer; position: relative; overflow: hidden;
          background: linear-gradient(135deg, #2563eb, #4f46e5, #7c3aed);
          background-size: 200% 200%;
          animation: loginGradShift 4s ease infinite, loginFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) 0.55s both;
          transition: transform 0.15s, box-shadow 0.15s, opacity 0.2s;
          box-shadow: 0 4px 24px rgba(79,70,229,0.4);
        }
        @keyframes loginGradShift {
          0%   { background-position: 0% 50%; }
          50%  { background-position: 100% 50%; }
          100% { background-position: 0% 50%; }
        }
        .login-btn::before {
          content: ''; position: absolute; top: 0; left: -100%;
          width: 100%; height: 100%;
          background: linear-gradient(90deg, transparent, rgba(255,255,255,0.12), transparent);
          transition: left 0.5s;
        }
        .login-btn:hover::before { left: 100%; }
        .login-btn:hover { transform: translateY(-1px); box-shadow: 0 8px 32px rgba(79,70,229,0.5); }
        .login-btn:active { transform: translateY(0) scale(0.99); }
        .login-btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none; }

        .login-btn-spinner {
          display: inline-block; width: 14px; height: 14px;
          border: 2px solid rgba(255,255,255,0.3);
          border-top-color: #fff; border-radius: 50%;
          animation: loginSpin 0.7s linear infinite;
          margin-right: 8px; vertical-align: middle;
        }
        @keyframes loginSpin { to { transform: rotate(360deg); } }

        .login-divider {
          display: flex; align-items: center; gap: 12px; margin: 20px 0;
          animation: loginFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) 0.6s both;
        }
        .login-divider::before, .login-divider::after {
          content: ''; flex: 1; height: 1px; background: rgba(255,255,255,0.08);
        }
        .login-divider span { font-size: 11px; color: #7986a3; text-transform: uppercase; letter-spacing: 1px; }

        .login-social-row { display: flex; gap: 10px; animation: loginFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) 0.65s both; }
        .login-social-btn {
          flex: 1; padding: 11px 14px;
          background: rgba(255,255,255,0.04);
          border: 1px solid rgba(255,255,255,0.08);
          border-radius: 12px;
          font-family: 'DM Sans', sans-serif; font-size: 13px; color: #7986a3;
          cursor: pointer; display: flex; align-items: center; justify-content: center; gap: 8px;
          transition: border-color 0.2s, background 0.2s, color 0.2s;
        }
        .login-social-btn:hover {
          border-color: rgba(99,179,237,0.4);
          background: rgba(79,142,247,0.06);
          color: #e8eaf6;
        }

        .login-signup-row {
          margin-top: 24px; text-align: center; font-size: 13px; color: #7986a3;
          animation: loginFadeUp 0.7s cubic-bezier(0.16,1,0.3,1) 0.7s both;
        }
        .login-signup-row a { color: #4f8ef7; text-decoration: none; font-weight: 500; transition: opacity 0.2s; }
        .login-signup-row a:hover { opacity: 0.8; }

        @media (max-width: 620px) {
          .login-card { grid-template-columns: 1fr; }
          .login-left { display: none; }
          .login-right { padding: 40px 28px; }
        }
      `}</style>

      <div className="login-root">
        <canvas ref={canvasRef} className="login-canvas" />
        <div className="login-scanline" />

        <div className="login-card">
          {/* LEFT PANEL */}
          <div className="login-left">
            <div className="login-brand">
              <div className="login-brand-icon">🧠</div>
              <h1>AI Teaching<br />Assistant</h1>
              <p>Your intelligent companion for smarter, faster, deeper learning.</p>
            </div>
            <div className="login-features">
              <div className="login-feature-item">
                <div className="login-feature-dot blue">⚡</div>
                <span>Adaptive AI-powered lessons</span>
              </div>
              <div className="login-feature-item">
                <div className="login-feature-dot purple">🔮</div>
                <span>Real-time knowledge insights</span>
              </div>
              <div className="login-feature-item">
                <div className="login-feature-dot pink">✨</div>
                <span>Personalized study paths</span>
              </div>
            </div>
          </div>

          {/* RIGHT PANEL */}
          <div className="login-right">
            <div className="login-welcome">
              <div className="login-welcome-tag">Secure Login</div>
              <h2>Welcome Back</h2>
              <p>Sign in to continue your learning journey.</p>
            </div>

            <form className="login-form" onSubmit={handleSubmit}>
              <div className="login-form-group">
                <label className="login-label">Email Address</label>
                <div className="login-input-wrap">
                  <input
                    type="email"
                    className="login-input"
                    placeholder="you@example.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                  />
                  <span className="login-input-icon">📧</span>
                </div>
              </div>

              <div className="login-form-group">
                <label className="login-label">Password</label>
                <div className="login-input-wrap">
                  <input
                    type="password"
                    className="login-input"
                    placeholder="••••••••"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                  />
                  <span className="login-input-icon">🔑</span>
                </div>
              </div>

              <div className="login-forgot">
                <a href="#">Forgot password?</a>
              </div>

              {error && (
                <div className="login-error">
                  ⚠ {error}
                </div>
              )}

              <button type="submit" className="login-btn" disabled={loading}>
                {loading ? (
                  <><span className="login-btn-spinner" />Signing in…</>
                ) : (
                  "Continue →"
                )}
              </button>
            </form>

            <div className="login-divider"><span>or</span></div>

            <div className="login-social-row">
              <button className="login-social-btn">
                <svg width="16" height="16" viewBox="0 0 24 24">
                  <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4" />
                  <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853" />
                  <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05" />
                  <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335" />
                </svg>
                Google
              </button>
              <button className="login-social-btn">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="#fff">
                  <path d="M12 0C5.37 0 0 5.37 0 12c0 5.3 3.44 9.8 8.21 11.39.6.11.82-.26.82-.58v-2.03c-3.34.73-4.04-1.61-4.04-1.61-.55-1.39-1.34-1.76-1.34-1.76-1.09-.75.08-.73.08-.73 1.2.08 1.84 1.24 1.84 1.24 1.07 1.83 2.81 1.3 3.49 1 .11-.78.42-1.3.76-1.6-2.67-.3-5.47-1.33-5.47-5.93 0-1.31.47-2.38 1.24-3.22-.13-.3-.54-1.52.12-3.18 0 0 1.01-.32 3.3 1.23a11.5 11.5 0 0 1 3-.4 11.5 11.5 0 0 1 3 .4c2.29-1.55 3.3-1.23 3.3-1.23.66 1.66.25 2.88.12 3.18.77.84 1.24 1.91 1.24 3.22 0 4.61-2.81 5.63-5.48 5.92.43.37.81 1.1.81 2.22v3.29c0 .32.22.7.83.58C20.56 21.8 24 17.3 24 12c0-6.63-5.37-12-12-12z" />
                </svg>
                GitHub
              </button>
            </div>

            <div className="login-signup-row">
              Don't have an account?{" "}
              <Link to="/signup">Create one →</Link>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}