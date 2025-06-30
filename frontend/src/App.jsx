import {React, useEffect} from 'react';
import './App.css';
import Users from './Pages/Users/Users';
import Roles from './Pages/Roles/Roles';
import Modules from './Pages/Modules/Modules';
import ModuleDetails from './Pages/Modules/ModuleDetails';
import { BrowserRouter as Router, Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom';

function Sidebar({ currentPage }) {
  const navigate = useNavigate();

  useEffect(() => {
    document.body.classList.add('theme-dark');
    document.body.classList.remove('theme-light');
    // document.body.classList.add('theme-light');
    // document.body.classList.remove('theme-dark');
  }, []);

  return (
    <aside className="sidebar"> 
      <div className="sidebar-header" onClick={() => navigate('/admin/modules')} style={{ cursor: 'pointer' }}>
        <img src="/icons/panbagnat.png" alt="Logo" className="sidebar-logo" />
        <span className="sidebar-title">Pan Bagnat</span>
      </div>
      <div className={`sidebar-item ${currentPage.startsWith('admin/modules') ? 'active' : 'inactive'}`} onClick={() => navigate('/admin/modules')}>
        <img src="/icons/modules.png" alt="Roles" className="sidebar-icon" />
        Modules
      </div>
      <div className={`sidebar-item ${currentPage.startsWith('admin/roles') ? 'active' : 'inactive'}`} onClick={() => navigate('/admin/roles')}>
        <img src="/icons/roles.png" alt="Roles" className="sidebar-icon" />
        Roles
      </div>
      <div className={`sidebar-item ${currentPage.startsWith('admin/users') ? 'active' : 'inactive'}`} onClick={() => navigate('/admin/users')}>
        <img src="/icons/users.png" alt="Users" className="sidebar-icon" />
        Users
      </div>
    </aside>
  );
}

function Main() {
  const location = useLocation();
  var path = location.pathname.slice(1) || 'modules';

  path = path.startsWith('/admin/modules') ? 'modules' : path;
  return (
    <div className="app-container">
      <Sidebar currentPage={path} />
      <main className="main-content">
        <Routes>
          <Route path="/admin/modules" element={<Modules onSort="name" />} />
          <Route path="/admin/modules/:moduleId" element={<ModuleDetails />} />
          <Route path="/admin/roles" element={<Roles onSort="name" />} />
          <Route path="/admin/users" element={<Users onSort="-last_seen" />} />
          <Route path="/" element={<Navigate to="/modules" replace />} />
        </Routes>
      </main>
    </div>
  );
}

function App() {
  return (
    <Router>
      <Main />
    </Router>
  );
}

export default App;
