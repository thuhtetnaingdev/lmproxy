import { useEffect, useState } from "react";
import { BrowserRouter, Routes, Route, Navigate, NavLink, useNavigate } from "react-router-dom";
import Dashboard from "@/pages/Dashboard";
import Models from "@/pages/Models";
import Requests from "@/pages/Requests";
import Login from "@/pages/Login";
import Settings from "@/pages/Settings";
import Tokens from "@/pages/Tokens";
import { getJWT, clearAuth, fetchMe, logout as apiLogout, getRefreshToken } from "@/lib/api";
import { BarChart3, LayoutDashboard, List, Settings2, Key, LogOut, User } from "lucide-react";

const navItems = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard },
  { to: "/models", label: "Models", icon: BarChart3 },
  { to: "/requests", label: "Requests", icon: List },
  { to: "/settings", label: "Settings", icon: Settings2 },
  { to: "/tokens", label: "API Tokens", icon: Key },
];

function ProtectedLayout() {
  const [auth, setAuth] = useState<{ id: number; username: string } | null>(null);
  const [checking, setChecking] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    const jwt = getJWT();
    if (!jwt) {
      setChecking(false);
      return;
    }
    fetchMe()
      .then(setAuth)
      .catch(() => clearAuth())
      .finally(() => setChecking(false));
  }, []);

  function handleLogout() {
    const rt = getRefreshToken();
    if (rt) apiLogout(rt).catch(() => {});
    clearAuth();
    setAuth(null);
    navigate("/login");
  }

  if (checking) {
    return <div className="flex items-center justify-center h-screen text-muted-foreground">Loading...</div>;
  }
  if (!auth) return <Navigate to="/login" replace />;

  return (
    <div className="flex h-screen bg-background">
      <aside className="w-56 border-r bg-card flex flex-col shrink-0">
        <div className="p-4 border-b">
          <h1 className="text-lg font-bold tracking-tight">LLMProxy</h1>
          <p className="text-xs text-muted-foreground">DeepSeek usage tracker</p>
        </div>
        <nav className="flex-1 p-3 space-y-1">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === "/"}
              className={({ isActive }) =>
                `w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md transition-colors ${
                  isActive
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                }`
              }
            >
              <item.icon className="h-4 w-4" />
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="p-3 border-t space-y-1">
          <div className="flex items-center gap-2 text-xs text-muted-foreground mb-1">
            <User className="h-3 w-3" />
            {auth.username}
          </div>
          <button
            onClick={handleLogout}
            className="w-full flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            <LogOut className="h-3 w-3" />
            Sign out
          </button>
        </div>
      </aside>
      <main className="flex-1 overflow-auto p-6">
        <Routes>
          <Route index element={<Dashboard />} />
          <Route path="models" element={<Models />} />
          <Route path="requests" element={<Requests />} />
          <Route path="settings" element={<Settings />} />
          <Route path="tokens" element={<Tokens />} />
        </Routes>
      </main>
    </div>
  );
}

function LoginPage() {
  const navigate = useNavigate();
  return <Login onLogin={() => navigate("/")} />;
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/*" element={<ProtectedLayout />} />
      </Routes>
    </BrowserRouter>
  );
}
