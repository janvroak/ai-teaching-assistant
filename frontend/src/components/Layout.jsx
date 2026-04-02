import { useNavigate } from 'react-router-dom';

function Layout({ children }) {
  const navigate = useNavigate();

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    navigate('/login', { replace: true });
  };

  return (
    <main className="min-h-screen bg-[#05050a] text-white">
      {/* Top Navigation Bar - Matching the screenshot */}
      <nav className="border-b border-white/10 bg-[#0a0a0f] px-8 py-5">
        <div className="mx-auto max-w-7xl flex items-center justify-between">
          <h1 className="text-2xl font-semibold tracking-tight">
            AI Teaching Assistant
          </h1>
          
          <button
            onClick={handleLogout}
            className="rounded-lg bg-blue-600 px-6 py-2 text-sm font-medium text-white hover:bg-blue-700 transition-colors"
          >
            Logout
          </button>
        </div>
      </nav>

      {/* Main Content Area - Dark & Clean */}
      <div className="mx-auto max-w-7xl px-8 py-10">
        {children}
      </div>
    </main>
  );
}

export default Layout;