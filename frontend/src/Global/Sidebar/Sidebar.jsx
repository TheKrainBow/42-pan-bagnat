// src/components/Sidebar.jsx
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import './Sidebar.css';

export default function Sidebar({ currentPage, onModuleSelect }) {
  const navigate = useNavigate();
  const mode = currentPage.startsWith('/admin/') ? 'admin' : 'user';

  // selectedPage will be one of the page objects (or null)
  const [selectedPage, setSelectedPage] = useState(null);

  // pages is the array of { name, display_name, url, is_public, module_id }
  const [pages, setPages] = useState([]);

  // Fetch your pages list when in user mode
  useEffect(() => {
    document.body.classList.add('theme-dark');
    document.body.classList.remove('theme-light');

    if (mode === 'user') {
      fetch('/api/v1/modules/pages')
        .then((res) => {
          if (!res.ok) throw new Error(res.statusText);
          return res.json();
        })
        .then((data) => {
          // pull out the array
          setPages(data.pages);
        })
        .catch(console.error);
    }
  }, [mode]);

  // On first load, auto-select the very first page
  useEffect(() => {
    if (pages.length > 0 && selectedPage === null) {
      setSelectedPage(pages[0]);
      onModuleSelect(pages[0]);
    }
  }, [pages, selectedPage, onModuleSelect]);

  // called when user clicks a page
  const handleSelect = (page) => {
    setSelectedPage(page);
    onModuleSelect(page);
    navigate(page.url);
  };

  // helper for admin nav highlighting
  const isActive = (path) =>
    currentPage.startsWith(path) ? 'active' : 'inactive';

  return (
    <aside className="sidebar">
      {/* header/logo */}
      <div
        className="sidebar-header"
        onClick={() =>
          navigate(mode === 'admin' ? '/admin/modules' : '/')
        }
        style={{ cursor: 'pointer' }}
      >
        <img src="/icons/panbagnat.png" alt="Logo" className="sidebar-logo" />
        <span className="sidebar-title">Pan Bagnat</span>
      </div>

      {mode === 'admin' ? (
        <>
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
          {/* user pages list */}
          <ul className="sidebar-user-modules">
            {pages.map((page) => (
              <li
                key={page.name}
                className={`sidebar-item ${
                  selectedPage?.name === page.name ? 'active' : 'inactive'
                }`}
                onClick={() => handleSelect(page)}
              >
                {page.name}
              </li>
            ))}
          </ul>

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
