import React, { useState } from 'react';
import './App.css';
import User from './User';
import Roles from './Roles';

function App() {
  const [page, setPage] = useState('users');

  return (
    <div className="app-container">
      <aside className="sidebar"> 
      <div className="sidebar-header">
        <img src="/icons/panbagnat.png" alt="Logo" className="sidebar-logo" />
        <span className="sidebar-title">Pan Bagnat</span>
      </div>
      <div className={`sidebar-item ${page === 'modules' ? 'active' : 'inactive'}`} onClick={() => setPage('modules')}>
        <img src="/icons/modules.png" alt="Modules" className="sidebar-icon" />
        Modules
      </div>
      <div className={`sidebar-item ${page === 'roles' ? 'active' : 'inactive'}`} onClick={() => setPage('roles')}>
        <img src="/icons/roles.png" alt="Roles" className="sidebar-icon" />
        Roles
      </div>
      <div className={`sidebar-item ${page === 'users' ? 'active' : 'inactive'}`} onClick={() => setPage('users')}>
        <img src="/icons/users.png" alt="Users" className="sidebar-icon" />
        Users
      </div>
      </aside>
      <main className="main-content">
        {page === 'modules' && <h2>Modules (coming soon)</h2>}
        {page === 'roles' && <Roles onSort="name" />}
        {page === 'users' && <User onSort="-last_seen" />}
      </main>
    </div>
  );
}

export default App;
