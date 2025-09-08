import { useEffect, useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import './Sidebar.css';

export default function Sidebar({ currentPage, user, pages }) {
  const navigate = useNavigate();
  const mode = currentPage.startsWith('/admin/') ? 'admin' : 'user';

  const [selectedPage, setSelectedPage] = useState(null);
  const location = useLocation();
  const match = location.pathname.match(/(?<!\/admin)\/modules\/([^/]+)/);
  const currentSlug = match ? match[1] : null;

  useEffect(() => {
    if (pages.length === 0) return;
    const found = pages.find((p) => p.slug === currentSlug);
    setSelectedPage(found || pages[0]);
  }, [pages, currentSlug]);

  const handleSelect = (page) => {
    setSelectedPage(page);
    navigate(`/modules/${page.slug}`);
  };

  const isActive = (path) => currentPage.startsWith(path) ? 'active' : 'inactive';

  return (
    <aside className="sidebar">
      <div className="sidebar-header" onClick={() => navigate(mode === 'admin' ? '/admin/modules' : '/')} style={{ cursor: 'pointer' }}>
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
            <div className="sidebar-item last" onClick={() => navigate('/modules')}>
              🔧 Switch to User
            </div>
          </div>
        </>
      ) : (
        <>
          <ul className="sidebar-user-modules">
            {pages.map((page) => (
              <li key={page.name} className={`sidebar-item ${currentSlug === page.slug ? 'active' : 'inactive'}`} onClick={() => handleSelect(page)}>
                {page.name}
              </li>
            ))}
          </ul>

          {user?.is_staff && (
            <div className="sidebar-footer">
              <div className="sidebar-item last" onClick={() => navigate('/admin/modules')}>
                🔧 Switch to Admin
              </div>
            </div>
          )}
        </>
      )}
    </aside>
  );
}
