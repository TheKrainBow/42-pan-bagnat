// src/components/Sidebar.jsx
import { useEffect, useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import './Sidebar.css'

export default function Sidebar({ currentPage, onModuleSelect }) {
  const navigate = useNavigate();
  const mode = currentPage.startsWith('/admin/') ? 'admin' : 'user';
  const [selectedModule, setSelectedModule] = useState(null);

  const [modules, setModules] = useState([]);

  useEffect(() => {
    document.body.classList.add('theme-dark');
    document.body.classList.remove('theme-light');

    if (mode === 'user') {
      fetch('http://localhost/__register')
        .then((res) => {
          if (!res.ok) throw new Error(res.statusText);
          return res.json();
        })
        .then(setModules)
        .catch(console.error);
    }
  }, [mode]);

  useEffect(() => {
    if (modules.length > 0 && selectedModule === null) {
      setSelectedModule(modules[0]);
      onModuleSelect(modules[0]);
    }
  }, [modules, selectedModule, onModuleSelect]);

  // any user action should update selectedId, e.g.:
  const handleSelect = (mod) => {
    setSelectedModule(mod);
    onModuleSelect(mod);
  };

  const isActive = (path) =>
    currentPage.startsWith(path) ? 'active' : 'inactive';

  return (
    <aside className="sidebar">
      {/* header */}
      <div
        className="sidebar-header"
        onClick={() =>
          navigate(mode === 'admin' ? '/admin/modules' : '/')
        }
        style={{ cursor: 'pointer' }}
      >
        <img
          src="/icons/panbagnat.png"
          alt="Logo"
          className="sidebar-logo"
        />
        <span className="sidebar-title">Pan Bagnat</span>
      </div>

      {mode === 'admin' ? (
        <>
          {/* admin nav items */}
          <div
            className={`sidebar-item ${isActive('/admin/modules')}`}
            onClick={() => navigate('/admin/modules')}
          >
            <img
              src="/icons/modules.png"
              alt="Modules"
              className="sidebar-icon"
            />
            Modules
          </div>
          <div
            className={`sidebar-item ${isActive('/admin/roles')}`}
            onClick={() => navigate('/admin/roles')}
          >
            <img
              src="/icons/roles.png"
              alt="Roles"
              className="sidebar-icon"
            />
            Roles
          </div>
          <div
            className={`sidebar-item ${isActive('/admin/users')}`}
            onClick={() => navigate('/admin/users')}
          >
            <img
              src="/icons/users.png"
              alt="Users"
              className="sidebar-icon"
            />
            Users
          </div>

          {/* switch back to admin */}
          <div className="sidebar-footer">
            <div
              className="sidebar-item"
              onClick={() => navigate('/modules')}
            >
              🔧 Switch to User
            </div>
          </div>
        </>
      ) : (
        <>
          {/* user list */}
          <ul className="sidebar-user-modules">
            {modules.map((modName) => (
              <li
                key={modName}
                className={`sidebar-item ${selectedModule === modName ? 'active' : 'inactive'}`}
                onClick={() => handleSelect(modName)}
              >
                {modName}
              </li>
            ))}
          </ul>

          {/* switch back to admin */}
          <div className="sidebar-footer">
            <div
              className="sidebar-item"
              onClick={() => navigate('/admin/modules')}
            >
              🔧 Switch to Admin
            </div>
          </div>
        </>
      )}
    </aside>
  );
}
