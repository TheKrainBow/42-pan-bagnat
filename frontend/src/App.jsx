// src/App.jsx (or wherever Main lives)
import React, { useState, useRef, useEffect, createContext } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate, useLocation } from 'react-router-dom';

import './App.css';
import Users from './Pages/Users/Users';
import Roles from './Pages/Roles/Roles';
import Modules from './Pages/Modules/Modules';
import ModuleDetails from './Pages/Modules/ModuleDetails/ModuleDetails';
import ModulePage from './Pages/Modules/ModulePage/ModulePage';
import LoginPage from "./Pages/Login/Login";
import Sidebar from 'Global/Sidebar/Sidebar';
import { socketService } from 'Global/SocketService/SocketService';
import { ToastContainer } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import "./Notifications.css"

function Main() {
  const location = useLocation();
  const path = location.pathname;
  const [activePage, setActivePage] = useState(null);

  const showSidebar = path !== "/login";

  document.body.classList.add('theme-dark');
  return (
    <div className="app-container">
      {showSidebar && (
        <Sidebar
          currentPage={path}
          onModuleSelect={(page) => setActivePage(page)}
        />
      )}

      <main className="main-content">
        <Routes>
          <Route path="/modules" element={<ModulePage page={activePage} />} />
          <Route path="/admin/modules" element={<Modules onSort="name" />} />
          <Route path="/admin/modules/:moduleId" element={<ModuleDetails />} />
          <Route path="/admin/roles" element={<Roles onSort="name" />} />
          <Route path="/admin/users" element={<Users onSort="-last_seen" />} />
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

console.log('SocketService ID:', socketService.id);

export default function App() {
  return (
    <Router>
      <Main />
    </Router>
  );
}