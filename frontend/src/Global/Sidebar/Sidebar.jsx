// src/components/Sidebar.jsx
import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import './Sidebar.css';
import { fetchWithAuth } from 'Global/utils/Auth';

function loadPrefsFor(login) {
  try {
    const raw = localStorage.getItem(`pb:sidebar:${login}`);
    return raw ? JSON.parse(raw) : { order: [], hidden: {} };
  } catch {
    return { order: [], hidden: {} };
  }
}

export default function Sidebar({ currentPage, user, pages }) {
  const navigate = useNavigate();
  const mode = currentPage.startsWith('/admin/') ? 'admin' : 'user';

  const [selectedPage, setSelectedPage] = useState(null);
  const [prefs, setPrefs] = useState(() => (user?.ft_login ? loadPrefsFor(user.ft_login) : { order: [], hidden: {} }));
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem('pb:sidebar:collapsed') === '1');
  const [anim, setAnim] = useState(''); // '' | 'collapsing' | 'expanding'
  const location = useLocation();
  const match = location.pathname.match(/(?<!\/admin)\/modules\/([^/]+)/);
  const currentSlug = match ? match[1] : null;

  // Sync selected page based on slug or fallback
  useEffect(() => {
    if (pages.length === 0) return;

    const found = pages.find((p) => p.slug === currentSlug);
    setSelectedPage(found || pages[0]);
  }, [pages, currentSlug]);

  // Reload prefs when user changes
  useEffect(() => {
    if (user?.ft_login) setPrefs(loadPrefsFor(user.ft_login));
  }, [user?.ft_login]);

  // Reload prefs on navigation (so returning from /settings reflects changes)
  useEffect(() => {
    if (user?.ft_login) setPrefs(loadPrefsFor(user.ft_login));
  }, [location.pathname, user?.ft_login]);

  // Live updates from settings page
  useEffect(() => {
    function onPrefsChanged(e) {
      if (!user?.ft_login) return;
      const login = e?.detail?.login;
      if (!login || login === user.ft_login) {
        setPrefs(loadPrefsFor(user.ft_login));
      }
    }
    window.addEventListener('pb:prefs:sidebarChanged', onPrefsChanged);
    return () => window.removeEventListener('pb:prefs:sidebarChanged', onPrefsChanged);
  }, [user?.ft_login]);


  // Load remote prefs on first login (cross-device persistence)
  useEffect(() => {
    if (!user?.ft_login) return;
    let cancelled = false;
    fetchWithAuth('/api/v1/users/me/prefs/sidebar')
      .then(r => r && r.ok ? r.json() : null)
      .then(remote => {
        if (!remote || cancelled) return;
        setPrefs(remote);
        try { localStorage.setItem(`pb:sidebar:${user.ft_login}`, JSON.stringify(remote)); } catch {}
      })
      .catch(() => {});
    return () => { cancelled = true; };
  }, [user?.ft_login]);

  // Always re-fetch remote prefs when entering settings (pages might have changed)
  useEffect(() => {
    if (!user?.ft_login) return;
    if (!location || location.pathname !== '/settings') return;
    let cancelled = false;
    fetchWithAuth('/api/v1/users/me/prefs/sidebar')
      .then(r => r && r.ok ? r.json() : null)
      .then(remote => {
        if (!remote || cancelled) return;
        setPrefs(remote);
        try { localStorage.setItem(`pb:sidebar:${user.ft_login}`, JSON.stringify(remote)); } catch {}
      })
      .catch(() => {});
    return () => { cancelled = true; };
  }, [location.pathname, user?.ft_login]);

  // Apply prefs to pages (order + hidden)
  const displayPages = useMemo(() => {
    if (mode !== 'user') return pages;
    const order = Array.isArray(prefs.order) ? prefs.order : [];
    const hidden = prefs.hidden || {};
    const bySlug = new Map(pages.map(p => [p.slug, p]));
    const ordered = [];
    for (const s of order) {
      if (bySlug.has(s)) ordered.push(bySlug.get(s));
      bySlug.delete(s);
    }
    for (const p of pages) {
      if (!ordered.find(x => x.slug === p.slug)) ordered.push(p);
    }
    return ordered.filter(p => !hidden[p.slug]);
  }, [pages, prefs, mode]);

  // On click
  const handleSelect = (page) => {
    setSelectedPage(page);
    navigate(`/modules/${page.slug}`);
  };

  const isActive = (path) =>
    currentPage.startsWith(path) ? 'active' : 'inactive';

  // Timings (ms). Keep these in sync with CSS in Sidebar.css
  const WIDTH_ANIM_MS = 0;     // .sidebar width transition
  const CENTER_DELAY_MS = WIDTH_ANIM_MS; // when to center chevron after collapse

  const toggleCollapsed = () => {
    if (!collapsed) {
      // Collapse immediately (width + label fade start together).
      setAnim('collapsing');
      setCollapsed(true);
      try { localStorage.setItem('pb:sidebar:collapsed', '1'); } catch {}
      // Keep chevron pinned until width transition completes.
      window.setTimeout(() => setAnim(''), CENTER_DELAY_MS);
    } else {
      // Expand: width first, then reveal labels.
      setAnim('expanding');
      setCollapsed(false);
      try { localStorage.setItem('pb:sidebar:collapsed', '0'); } catch {}
      window.setTimeout(() => setAnim(''), WIDTH_ANIM_MS);
    }
  };

  return (
    <aside className={`sidebar ${collapsed ? 'collapsed' : ''} ${anim}`}>
      {/* Brand header + collapse */}
      <div className="sidebar-header">
        <div className="sidebar-topbar">
          <div className="sidebar-brand">
            <img src="/icons/panbagnat.png" alt="Pan Bagnat" className="sidebar-brand-icon" />
            <span className="sidebar-brand-title">Pan Bagnat</span>
          </div>
          <button className="sidebar-collapse-btn header" onClick={toggleCollapsed} title={collapsed ? 'Expand' : 'Collapse'}>
            <img src={collapsed ? '/icons/chevron-right.svg' : '/icons/chevron-left.svg'} alt="" className="sidebar-icon themed-icon" />
          </button>
        </div>
      </div>
      <div className="sidebar-sep" />

      {mode === 'admin' ? (
        <>
          <div className="sidebar-section-title">{collapsed ? 'ADM' : 'Admin'}</div>
          <ul className="sidebar-user-modules">
            <li className={`sidebar-item ${isActive('/admin/modules')}`} onClick={() => navigate('/admin/modules')} title={collapsed ? 'Modules' : undefined}>
              <img src="/icons/modules.png" alt="" className="sidebar-icon" />
              <span className="sidebar-label">Modules</span>
            </li>
            <li className={`sidebar-item ${isActive('/admin/roles')}`} onClick={() => navigate('/admin/roles')} title={collapsed ? 'Roles' : undefined}>
              <img src="/icons/roles.png" alt="" className="sidebar-icon" />
              <span className="sidebar-label">Roles</span>
            </li>
            <li className={`sidebar-item ${isActive('/admin/users')}`} onClick={() => navigate('/admin/users')} title={collapsed ? 'Users' : undefined}>
              <img src="/icons/users.png" alt="" className="sidebar-icon" />
              <span className="sidebar-label">Users</span>
            </li>
            <li className={`sidebar-item ${isActive('/admin/ssh-keys')}`} onClick={() => navigate('/admin/ssh-keys')} title={collapsed ? 'SSH Keys' : undefined}>
              <span className="sidebar-icon" role="img" aria-label="SSH Keys">🔑</span>
              <span className="sidebar-label">SSH Keys</span>
            </li>
          </ul>
          <div className="sidebar-footer">
            <div className="sidebar-sep" />
            <div className="sidebar-item" onClick={() => navigate('/modules')} title={collapsed ? 'User Dashboard' : undefined}>
              <img src="/icons/42.svg" className="sidebar-icon builtin-icon" alt="" />
              <span className="sidebar-label">User Dashboard</span>
            </div>
            <div className="sidebar-sep" />
            {user && (
              <div className="sidebar-userfooter" onClick={() => navigate('/settings')} title={collapsed ? user.ft_login : undefined}>
                <div className="avatar-wrap">
                  <img src={user.ft_photo} alt={user.ft_login} className="sidebar-avatar" />
                  <span className="sidebar-status online" />
                </div>
                <span className="sidebar-userlogin">{user.ft_login}</span>
              </div>
            )}
          </div>
        </>
      ) : (
        <>
          <div className="sidebar-section-title">{collapsed ? 'MODS' : 'Modules'}</div>
          <ul className="sidebar-user-modules">
            {displayPages.map((page) => (
              <li
                key={page.slug}
                className={`sidebar-item ${currentSlug === page.slug ? 'active' : 'inactive'}`}
                onClick={() => handleSelect(page)}
                title={collapsed ? page.name : undefined}
              >
                <img className="sidebar-icon" src={page.icon_url || '/icons/modules.png'} alt="" />
                <span className="sidebar-label">{page.name}</span>
              </li>
            ))}
          </ul>
          <div className="sidebar-footer">
            {user?.is_staff && (
              <>
                <div className="sidebar-sep" />
                <div className="sidebar-item" onClick={() => navigate('/admin/modules')} title={collapsed ? 'Admin Dashboard' : undefined}>
                  <img src="/icons/admin-dashboard.svg" className="sidebar-icon themed-icon" alt="" />
                  <span className="sidebar-label">Admin Dashboard</span>
                </div>
              </>
            )}
            <div className="sidebar-sep" />
            {user && (
              <div className="sidebar-userfooter" onClick={() => navigate('/settings')} title={collapsed ? user.ft_login : undefined}>
                <div className="avatar-wrap">
                  <img src={user.ft_photo} alt={user.ft_login} className="sidebar-avatar" />
                  <span className="sidebar-status online" />
                </div>
                <span className="sidebar-userlogin">{user.ft_login}</span>
              </div>
            )}
          </div>
        </>
      )}
    </aside>
  );
}
