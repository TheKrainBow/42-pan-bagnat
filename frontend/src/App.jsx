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
import RoleDetail from 'Pages/Roles/RoleDetail/RoleDetail';
import UserDetail from './Pages/Users/UserDetail';
import ModulePage from './Pages/Modules/ModulePage/ModulePage';
import LoginPage from "./Pages/Login/Login";
import Sidebar from 'Global/Sidebar/Sidebar';
import { socketService } from 'Global/SocketService/SocketService';
import { ToastContainer } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import "./Notifications.css";

function Main() {
  const location = useLocation();
  const { slug } = useParams();
  const navigate = useNavigate();
  const path = location.pathname;
  const mode = path.startsWith('/admin/') ? 'admin' : 'user';

  const [user, setUser] = useState(null);
  const [userLoaded, setUserLoaded] = useState(false);
  const [pages, setPages] = useState([]);

  const showSidebar = path !== "/login";

  // Load current user
  useEffect(() => {
    if (showSidebar) {
      fetchWithAuth('/api/v1/users/me')
        .then((res) => {
          if (!res.ok) throw new Error(res.statusText);
          return res.json();
        })
        .then(setUser)
        .catch(() => setUser(null))
        .finally(() => setUserLoaded(true));
    }
  }, []);

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

  // Redirect to login if unauthenticated user in user mode
  if (userLoaded && !user && mode === 'user') {
    return <Navigate to="/login" replace />;
  }

  document.body.classList.add('theme-dark');

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
