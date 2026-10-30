// src/App.jsx
import React, { useState, useEffect } from 'react';
import { fetchWithAuth } from 'Global/utils/Auth';
import {
  BrowserRouter as Router,
  Routes,
  Route,
  Navigate,
  useLocation,
  useNavigate,
  useParams,
} from 'react-router-dom';

import './App.css';
import Users from './Pages/Users/Users';
import Roles from './Pages/Roles/Roles';
import Modules from './Pages/Modules/Modules';
import ModuleDetails from './Pages/Modules/ModuleDetails/ModuleDetails';
import Redirections from './Pages/Redirections/Redirections';
import RoleDetail from 'Pages/Roles/RoleDetail/RoleDetail';
import RoleRuleBuilder from 'Pages/Roles/RoleRuleBuilder/RoleRuleBuilder';
import UserDetail from './Pages/Users/UserDetail';
import ModulePage from './Pages/Modules/ModulePage/ModulePage';
import LoginPage from "./Pages/Login/Login";
import Sidebar from 'Global/Sidebar/Sidebar';
import UserSettingsPage from 'Pages/UserSettings/UserSettingsPage';
import { TourProvider } from 'Global/Tour/TourProvider';
import { socketService, setCurrentUserIsAdmin } from 'Global/SocketService/SocketService';
import { ToastContainer, toast } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import "./Notifications.css";
import { initTheme } from 'Global/Theme/theme';
import SSHKeys from './Pages/SSHKeys/SSHKeys';
import Stats from './Pages/Stats/Stats';
import ModuleStats from './Pages/Stats/ModuleStats';
import ContainersPage from './Pages/Containers/ContainersPage';
import DashboardEditor from './Pages/Dashboard/DashboardEditor';
import DashboardView from './Pages/Dashboard/DashboardView';
import TermsOfUse from './Pages/Legal/TermsOfUse';
import PrivacyPolicy from './Pages/Legal/PrivacyPolicy';

function Main() {
  const location = useLocation();
  const navigate = useNavigate();
  const { slug } = useParams();
  const path = location.pathname;
  const mode = path.startsWith('/admin/') ? 'admin' : 'user';

  const [user, setUser] = useState(null);
  const [userLoaded, setUserLoaded] = useState(false);
  const [pages, setPages] = useState([]);

  const showSidebar = path !== "/login";

  useEffect(() => {
    try {
      const raw = sessionStorage.getItem('pb:pendingToast');
      if (raw) {
        const t = setTimeout(() => {
          sessionStorage.removeItem('pb:pendingToast');
        }, 300);
        const data = JSON.parse(raw) || {};
        const kind = data.kind || 'info';
        const msg = data.message || '';
        if (msg) (toast[kind] || toast.info)(msg);
      }
    } catch (e) {
      console.log(e)
    }
  }, []); // run once on mount

  // Load current user
  useEffect(() => {
    if (!showSidebar) return;
    let cancelled = false;

    fetchWithAuth('/api/v1/users/me')
      .then((res) => {
        if (!res || !res.ok) throw new Error(res ? res.statusText : "no response");
        return res.json();
      })
      .then((u) => { if (!cancelled) { setUser(u); setCurrentUserIsAdmin(u?.is_staff); } })
      .catch(() => { if (!cancelled) { setUser(null); setCurrentUserIsAdmin(false); } })
      .finally(() => { if (!cancelled) setUserLoaded(true); });

    return () => { cancelled = true; };
  }, [showSidebar]);

  // Claude tab notifications: the owner of a session is told, wherever they are
  // in PB, when Claude waits for them or has finished its task.
  useEffect(() => {
    if (!user?.id) return undefined;
    const topic = `claude-user:${user.id}`;
    socketService.subscribeTopic(topic);
    const unsub = socketService.subscribe((evt) => {
      if (evt?.eventType !== 'claude_notify' || !evt.payload) return;
      // The topic is already owner-only (backend); never toast someone else's session.
      if (evt.payload.user_id !== user.id) return;
      const { kind, module_id, module_name, title, message } = evt.payload;
      const url = `/admin/modules/${module_id}?tab=claude`;
      const heading = {
        waiting: `Claude is waiting for you on ${module_name}`,
        error: `Claude stopped on ${module_name}`,
      }[kind] || `Claude finished on ${module_name}`;
      const body = [title, message].filter(Boolean).join(' — ');
      toast(`${heading}${body ? `: ${body}` : ''}`, {
        type: { waiting: 'warning', error: 'error' }[kind] || 'success',
        autoClose: kind === 'waiting' ? false : 8000,
        onClick: () => navigate(url),
        className: 'toast-simple',
      });
      if (document.hidden && 'Notification' in window && Notification.permission === 'granted') {
        try {
          const n = new Notification(heading, { body, tag: `claude-${module_id}` });
          n.onclick = () => { window.focus(); navigate(url); n.close(); };
        } catch { /* notifications unavailable */ }
      }
    });
    return () => { socketService.unsubscribeTopic(topic); unsub(); };
  }, [user?.id, navigate]);

  // Load user pages
  useEffect(() => {
    if (!user || mode !== 'user') return;

    fetchWithAuth('/api/v1/users/me/pages')
      .then((res) => {
        if (!res.ok) throw new Error(res.statusText);
        return res.json();
      })
      .then((data) => {
        setPages(Array.isArray(data) ? data : []);
      })
      .catch(console.error);
  }, [mode, user]);

  // Initialize theme from localStorage ('dark' | 'light')
  useEffect(() => {
    initTheme();
  }, []);

  const shouldRedirectToLogin = userLoaded && !user && mode === 'user' && path !== '/login';
  if (shouldRedirectToLogin) {
    const nextDest = encodeURIComponent(location.pathname + (location.search || ''));
    const loginTarget = nextDest ? `/login?next=${nextDest}` : '/login';
    return <Navigate to={loginTarget} replace />;
  }

  return (
    <div className="app-container">
      {showSidebar && (
        <Sidebar
          currentPage={path}
          currentSlug={slug}
          user={user}
          pages={pages}
          onModuleSelect={() => {}} // no longer needed
        />
      )}

      <main className="main-content">
        <Routes>
          <Route path="/modules" element={<ModulePage pages={pages} user={user} />} />
          <Route path="/modules/:slug" element={<ModulePage pages={pages} user={user} />} />
          <Route path="/dashboard" element={<DashboardView />} />
          <Route path="/settings" element={<UserSettingsPage pages={pages} user={user} />} />
          <Route path="/legal/terms" element={<TermsOfUse />} />
          <Route path="/legal/privacy" element={<PrivacyPolicy />} />
          <Route path="/admin/modules" element={<Modules onSort="name" />} />
          <Route path="/admin/modules/:moduleId" element={<ModuleDetails />} />
          <Route path="/admin/redirections" element={<Redirections />} />
          <Route path="/admin/roles" element={<Roles onSort="name" />} />
          <Route path="/admin/roles/:roleId" element={<RoleDetail />} />
          <Route path="/admin/roles/:roleId/rule-builder" element={<RoleRuleBuilder />} />
          <Route path="/admin/users" element={<Users onSort="-last_seen" />} />
          <Route path="/admin/users/:identifier" element={<UserDetail />} />
          <Route path="/admin/ssh-keys" element={<SSHKeys />} />
          <Route path="/admin/stats" element={<Stats />} />
          <Route path="/admin/stats/:moduleId" element={<ModuleStats />} />
          <Route path="/admin/containers" element={<ContainersPage />} />
          <Route path="/admin/dashboard" element={<DashboardEditor />} />
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="*"
            element={
              path.startsWith("/admin/")
                ? <Navigate to="/admin/modules" replace />
                : <Navigate to="/modules" replace />
            }
          />
        </Routes>
      </main>

      <ToastContainer
        position="bottom-right"
        autoClose={2000}
        pauseOnHover={true}
        newestOnTop={true}
        limit={5}
      />
    </div>
  );
}

export default function App() {
  return (
    <Router>
      <TourProvider>
        <Main />
      </TourProvider>
    </Router>
  );
}
