// src/components/Sidebar.jsx
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import './Sidebar.css'

export default function Sidebar({ currentPage, onModuleSelect }) {
  const navigate = useNavigate();
  const mode = currentPage.startsWith('/admin/') ? 'admin' : 'user';

  const [modules, setModules] = useState([]);

  useEffect(() => {
    document.body.classList.add('theme-dark');
    document.body.classList.remove('theme-light');

    if (mode === 'user') {
      fetch('/__register')
        .then((res) => {
          if (!res.ok) throw new Error(res.statusText);
          return res.json();
        })
        .then(setModules)
        .catch(console.error);
    }
  }, [mode]);

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
            className={`sidebar-item ${isActive('admin/modules')}`}
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
            className={`sidebar-item ${isActive('admin/roles')}`}
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
            className={`sidebar-item ${isActive('admin/users')}`}
            onClick={() => navigate('/admin/users')}
          >
            <img
              src="/icons/users.png"
              alt="Users"
              className="sidebar-icon"
            />
            Users
          </div>
        </>
      ) : (
        <>
          {/* user list */}
          <ul className="sidebar-user-modules">
            {modules.map((modName) => (
              <li
                key={modName}
                className="sidebar-item inactive"
                onClick={() => onModuleSelect(modName)}
              >
                {modName}
              </li>
            ))}
          </ul>

          {/* switch back to admin */}
          <div className="sidebar-footer">
            <button
              className="sidebar-item switch-to-admin"
              onClick={() => navigate('/admin/modules')}
            >
              🔧 Switch to Admin
            </button>
          </div>
        </>
      )}
    </aside>
  );
}
