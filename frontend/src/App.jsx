import React from 'react';
import './App.css';
import User from './pages/User';
import Roles from './pages/Roles';
import Modules from './pages/Modules';
import { BrowserRouter as Router, Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom';

function Sidebar({ currentPage }) {
  const navigate = useNavigate();

  return (
    <aside className="sidebar"> 
      <div className="sidebar-header" onClick={() => navigate('/modules')} style={{ cursor: 'pointer' }}>
        <img src="/icons/panbagnat.png" alt="Logo" className="sidebar-logo" />
        <span className="sidebar-title">Pan Bagnat</span>
      </div>
      <div className={`sidebar-item ${currentPage === 'modules' ? 'active' : 'inactive'}`} onClick={() => navigate('/modules')}>
        <img src="/icons/modules.png" alt="Modules" className="sidebar-icon" />
        Modules
      </div>
      <div className={`sidebar-item ${currentPage === 'roles' ? 'active' : 'inactive'}`} onClick={() => navigate('/roles')}>
        <img src="/icons/roles.png" alt="Roles" className="sidebar-icon" />
        Roles
      </div>
      <div className={`sidebar-item ${currentPage === 'users' ? 'active' : 'inactive'}`} onClick={() => navigate('/users')}>
        <img src="/icons/users.png" alt="Users" className="sidebar-icon" />
        Users
      </div>
    </aside>
  );
}

function Main() {
  const location = useLocation();
  const path = location.pathname.slice(1) || 'modules';

  return (
    <div className="app-container">
      <Sidebar currentPage={path} />
      <main className="main-content">
        <Routes>
          <Route path="/modules" element={<Modules onSort="name" />} />
          <Route path="/roles" element={<Roles onSort="name" />} />
          <Route path="/users" element={<User onSort="-last_seen" />} />
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
