// src/components/Sidebar.jsx
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchWithAuth } from 'Global/utils/Auth';
import './Sidebar.css';

export default function Sidebar({ currentPage, currentSlug, user, onModuleSelect, pages }) {
  const navigate = useNavigate();
  const mode = currentPage.startsWith('/admin/') ? 'admin' : 'user';

  const [selectedPage, setSelectedPage] = useState(null);

  // Sync selected page based on slug or fallback
  useEffect(() => {
    if (pages.length === 0) return;

    const found = pages.find((p) => p.slug === currentSlug);
    const selected = found || pages[0];

    setSelectedPage(selected);
    onModuleSelect(selected);
    // Do NOT navigate here — redirection happens in App.jsx
  }, [pages, currentSlug, onModuleSelect]);

  // On click
  const handleSelect = (page) => {
    setSelectedPage(page);
    onModuleSelect(page);
    navigate(`/modules/${page.slug}`);
  };

  // For admin tab highlighting
  const isActive = (path) =>
    currentPage.startsWith(path) ? 'active' : 'inactive';

  return (
    <aside className="sidebar">
      {/* Header / logo */}
      <div
        className="sidebar-header"
        onClick={() => navigate(mode === 'admin' ? '/admin/modules' : '/')}
        style={{ cursor: 'pointer' }}
      >
        <img src="/icons/panbagnat.png" alt="Logo" className="sidebar-logo" />
        <span className="sidebar-title">Pan Bagnat</span>
      </div>

      {mode === 'admin' ? (
        <>
          <div className={`sidebar-item ${isActive('/admin/modules')}`} onClick={() => navigate('/admin/modules')}>
            <img src="/icons/modules.png" alt="Modules" className="sidebar-icon" />
            Modules
          </div>
          <div className={`sidebar-item ${isActive('/admin/roles')}`} onClick={() => navigate('/admin/roles')}>
            <img src="/icons/roles.png" alt="Roles" className="sidebar-icon" />
            Roles
          </div>
          <div className={`sidebar-item ${isActive('/admin/users')}`} onClick={() => navigate('/admin/users')}>
            <img src="/icons/users.png" alt="Users" className="sidebar-icon" />
            Users
          </div>
          <div className="sidebar-footer">
            <div className="sidebar-item" onClick={() => navigate('/modules')}>
              🔧 Switch to User
            </div>
          </div>
        </>
      ) : (
        <>
          <ul className="sidebar-user-modules">
            {Array.isArray(pages) &&
              pages.map((page) => (
                <li
                  key={page.name}
                  className={`sidebar-item ${selectedPage?.name === page.name ? 'active' : 'inactive'}`}
                  onClick={() => handleSelect(page)}
                >
                  {page.name}
                </li>
              ))}
          </ul>

          {user && user.is_staff && (
            <div className="sidebar-footer">
              <div className="sidebar-item" onClick={() => navigate('/admin/modules')}>
                🔧 Switch to Admin
              </div>
            </div>
          )}
        </>
      )}
    </aside>
  );
}
