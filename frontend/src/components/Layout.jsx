import { useNavigate } from 'react-router-dom';
import { clearAuthSession } from '../utils/auth';

function Layout({ children }) {
  const navigate = useNavigate();

  const handleLogout = () => {
    clearAuthSession();
    navigate('/login', { replace: true });
  };

  return (
    <main className="relative min-h-screen bg-dark-background text-slate-100">
      <div className="app-noise-overlay" />
      <nav className="border-b border-white/10 bg-[#0b1020]/80 px-8 py-5 backdrop-blur-xl">
        <div className="mx-auto max-w-7xl flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-blue-400/30 bg-blue-500/10 text-blue-200">
              <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current" aria-hidden="true">
                <path d="M12 2 3 6.5 12 11l7-3.5V14h2V6.5L12 2Zm-7 8.2V15c0 2.97 3.47 5 7 5s7-2.03 7-5v-4.8l-7 3.5-7-3.5Z" />
              </svg>
            </div>
            <div>
              <h1 className="text-xl font-semibold tracking-tight text-slate-100">
                AI Teaching Assistant
              </h1>
              <p className="text-xs uppercase tracking-[0.16em] text-slate-400">Learning Workspace</p>
            </div>
          </div>
          
          <button
            onClick={handleLogout}
            className="rounded-xl border border-white/10 bg-white/5 px-6 py-2 text-sm font-medium text-slate-300 transition-all duration-300 hover:-translate-y-0.5 hover:border-blue-500/30 hover:bg-white/10 hover:text-slate-100"
          >
            Logout
          </button>
        </div>
      </nav>

      <div className="mx-auto max-w-7xl px-8 py-10">
        {children}
      </div>
    </main>
  );
}

export default Layout;
