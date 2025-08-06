// src/App.jsx (or wherever Main lives)
import React, { useState, useEffect } from 'react';
import { fetchWithAuth } from 'Global/utils/Auth';
import {
  BrowserRouter as Router,
  Routes,
  Route,
  Navigate,
  useLocation,
  useParams,
  useNavigate,
} from 'react-router-dom';

import './App.css';
import Users from './Pages/Users/Users';
import Roles from './Pages/Roles/Roles';
import Modules from './Pages/Modules/Modules';
import ModuleDetails from './Pages/Modules/ModuleDetails/ModuleDetails';
import RoleDetail from 'Pages/Roles/RoleDetail/RoleDetail';
import UserDetail from './Pages/Users/UserDetail';
import ModulePage from './Pages/Modules/ModulePage/ModulePage';
import LoginPage from "./Pages/Login/Login";
import Sidebar from 'Global/Sidebar/Sidebar';
import { socketService } from 'Global/SocketService/SocketService';
import { ToastContainer } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import "./Notifications.css"

function Main() {
  const location = useLocation();
  const { slug } = useParams();
  const navigate = useNavigate();
  const path = location.pathname;
  const mode = path.startsWith('/admin/') ? 'admin' : 'user';
  const [user, setUser] = useState(null);
  const [userLoaded, setUserLoaded] = useState(false);
  
  const [activePage, setActivePage] = useState(null);
  const [pages, setPages] = useState([]);

  const showSidebar = path !== "/login";

  useEffect(() => {
    if (showSidebar) {
      fetchWithAuth('/api/v1/users/me')
        .then((res) => {
          if (!res.ok) throw new Error(res.statusText);
          return res.json();
        })
        .then((data) => {
          setUser(data);
        })
        .catch(() => {
          setUser(null); // unauthenticated
        })
        .finally(() => {
          setUserLoaded(true);
        });
    }
  }, []);

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
  }, [mode, slug, user]);

  useEffect(() => {
    if (!user || mode !== "user") return;

    if (!slug) {
      if (pages.length > 0) {
        navigate(`/modules/${pages[0].slug}`, { replace: true });
      } else {
        navigate('/modules', { replace: true });
      }
      return;
    }

    const found = pages.find((p) => p.slug === slug);
    if (!found && pages.length > 0) {
      navigate(`/modules/${pages[0].slug}`, { replace: true });
    }
  }, [pages, slug, navigate, mode, user]);

  if (userLoaded && !user && mode === "user") {
    return <Navigate to="/login" replace />;
  }

  // Auto-redirect from /modules → /modules/:firstSlug
  // if (path === "/modules" && pages.length > 0) {
  //   return <Navigate to={`/modules/${pages[0].slug}`} replace />;
  // }

  document.body.classList.add('theme-dark');

  return (
    <div className="app-container">
      {showSidebar && (
        <Sidebar
          currentPage={path}
          currentSlug={slug}
          user={user}
          onModuleSelect={setActivePage}
          pages={pages}
        />
      )}

      <main className="main-content">
        <Routes>
          <Route path="/modules" element={<ModulePage/>} />
          <Route path="/modules/:slug" element={<ModulePage page={activePage} />} />
          <Route path="/admin/modules" element={<Modules onSort="name" />} />
          <Route path="/admin/modules/:moduleId" element={<ModuleDetails />} />
          <Route path="/admin/roles" element={<Roles onSort="name" />} />
          <Route path="/admin/roles/:roleId" element={<RoleDetail />} />
          <Route path="/admin/users" element={<Users onSort="-last_seen" />} />
          <Route path="/admin/users/:identifier" element={<UserDetail />} />
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
      <Main />
    </Router>
  );
}