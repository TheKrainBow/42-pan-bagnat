// src/App.jsx (or wherever Main lives)
import React, { useState, useRef, useEffect, createContext } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate, useLocation } from 'react-router-dom';

import './App.css';
import Users from './Pages/Users/Users';
import Roles from './Pages/Roles/Roles';
import Modules from './Pages/Modules/Modules';
import ModuleDetails from './Pages/Modules/ModuleDetails';
import ModulePage from './Pages/Modules/ModulePage';
import Sidebar from 'Global/Sidebar';
import { socketService } from 'Global/SocketService';

function Main() {
  const location = useLocation();
  const path = location.pathname;
  const [activePage, setActivePage] = useState(null);

  return (
    <div className="app-container">
      <Sidebar
        currentPage={path}
        onModuleSelect={(page) => setActivePage(page)}
      />

      <main className="main-content">
        <Routes>
          {/* user‐mode home shows the ModulePage */}
          <Route path="/modules" element={<ModulePage page={activePage} />} />

          {/* admin screens */}
          <Route path="/admin/modules" element={<Modules onSort="name" />} />
          <Route path="/admin/modules/:moduleId" element={<ModuleDetails />} />
          <Route path="/admin/roles" element={<Roles onSort="name" />} />
          <Route path="/admin/users" element={<Users onSort="-last_seen" />} />

          {/* catch‐all: redirect unknown to either admin or user home */}
          <Route path="*" element={
              path.startsWith('/admin/')
                ? <Navigate to="/admin/modules" replace />
                : <Navigate to="/modules" replace />
            }
          />
        </Routes>
      </main>
    </div>
  );
}

console.log('SocketService ID:', socketService.id);

export default function App() {
  return (
    <Router>
      <Main />
    </Router>
  );
}