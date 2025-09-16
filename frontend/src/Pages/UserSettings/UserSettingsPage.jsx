import React, { useEffect, useMemo, useRef, useState } from 'react';
import './UserSettingsPage.css';
import { fetchWithAuth } from 'Global/utils/Auth';
import RoleBadge from 'Global/RoleBadge/RoleBadge';
import { getReadableStyles } from 'Global/utils/ColorUtils';

function loadPrefs(login) {
  try {
    const raw = localStorage.getItem(`pb:sidebar:${login}`);
    return raw ? JSON.parse(raw) : { order: [], hidden: {} };
  } catch {
    return { order: [], hidden: {} };
  }
}

function savePrefs(login, prefs) {
  try {
    localStorage.setItem(`pb:sidebar:${login}`, JSON.stringify(prefs));
  } catch {}
}

export default function UserSettingsPage({ pages: initialPages, user: initialUser }) {
  const [user, setUser] = useState(initialUser || null);
  const [pages, setPages] = useState(initialPages || []);
  const [prefs, setPrefs] = useState(() => (initialUser?.ft_login ? loadPrefs(initialUser.ft_login) : { order: [], hidden: {} }));
  const [sessions, setSessions] = useState([]);
  const [loadingSessions, setLoadingSessions] = useState(true);
  const [saving, setSaving] = useState(false);
  const saveTimerRef = useRef(null);
  const [roles, setRoles] = useState([]);
  const [loadingRoles, setLoadingRoles] = useState(false);

  // Fetch user if missing
  useEffect(() => {
    if (user) return;
    fetchWithAuth('/api/v1/users/me')
      .then(r => r && r.ok ? r.json() : null)
      .then(setUser)
      .catch(() => setUser(null));
  }, []);

  // Fetch pages if missing
  useEffect(() => {
    if (!user || (initialPages && initialPages.length)) return;
    fetchWithAuth('/api/v1/users/me/pages')
      .then(r => r && r.ok ? r.json() : [])
      .then(setPages)
      .catch(() => setPages([]));
  }, [user]);

  // Load sessions
  useEffect(() => {
    if (!user) return;
    setLoadingSessions(true);
    fetchWithAuth('/api/v1/users/me/sessions')
      .then(r => r && r.ok ? r.json() : [])
      .then((arr) => { setSessions(Array.isArray(arr) ? arr : []); })
      .finally(() => setLoadingSessions(false));
  }, [user]);

  // Load roles
  useEffect(() => {
    if (!user) return;
    setLoadingRoles(true);
    fetchWithAuth('/api/v1/users/me/roles')
      .then(r => r && r.ok ? r.json() : [])
      .then(arr => setRoles(Array.isArray(arr) ? arr : []))
      .finally(() => setLoadingRoles(false));
  }, [user]);

  // Load prefs from server and sync local
  useEffect(() => {
    if (!user?.ft_login) return;
    fetchWithAuth('/api/v1/users/me/prefs/sidebar')
      .then(r => r && r.ok ? r.json() : null)
      .then(remote => {
        if (!remote) return;
        setPrefs(remote);
        try { localStorage.setItem(`pb:sidebar:${user.ft_login}`, JSON.stringify(remote)); } catch {}
        window.dispatchEvent(new CustomEvent('pb:prefs:sidebarChanged', { detail: { login: user.ft_login } }));
      })
      .catch(() => {});
  }, [user?.ft_login]);

  // Keep prefs synced with user
  useEffect(() => {
    if (user?.ft_login) setPrefs(loadPrefs(user.ft_login));
  }, [user?.ft_login]);

  // Compute ordered pages with visibility
  const orderedPages = useMemo(() => {
    const order = Array.isArray(prefs.order) ? prefs.order : [];
    const hidden = prefs.hidden || {};
    const bySlug = new Map(pages.map(p => [p.slug, p]));
    const out = [];
    for (const s of order) { if (bySlug.has(s)) { out.push(bySlug.get(s)); bySlug.delete(s); } }
    for (const p of pages) { if (!out.find(x => x.slug === p.slug)) out.push(p); }
    return out.map(p => ({ ...p, _hidden: !!hidden[p.slug] }));
  }, [pages, prefs]);

  // Drag and drop reorder
  const dragSlugRef = useRef(null);
  const onDragStart = (slug) => (e) => { dragSlugRef.current = slug; e.dataTransfer.effectAllowed = 'move'; };
  const onDragOver = (slug) => (e) => { e.preventDefault(); e.dataTransfer.dropEffect = 'move'; };
  const onDrop = (slug) => (e) => {
    e.preventDefault();
    const from = dragSlugRef.current;
    const to = slug;
    if (!from || !to || from === to) return;
    const newOrder = orderedPages.map(p => p.slug);
    const fromIdx = newOrder.indexOf(from);
    const toIdx = newOrder.indexOf(to);
    if (fromIdx < 0 || toIdx < 0) return;
    newOrder.splice(toIdx, 0, newOrder.splice(fromIdx, 1)[0]);
    pushPrefs({ ...prefs, order: newOrder });
  };

  function pushPrefs(next) {
    if (!user?.ft_login) return;
    setPrefs(next);
    try { localStorage.setItem(`pb:sidebar:${user.ft_login}`, JSON.stringify(next)); } catch {}
    window.dispatchEvent(new CustomEvent('pb:prefs:sidebarChanged', { detail: { login: user.ft_login } }));
    setSaving(true);
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    saveTimerRef.current = setTimeout(async () => {
      await fetchWithAuth('/api/v1/users/me/prefs/sidebar', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(next),
      });
      setSaving(false);
    }, 400);
  }

  const toggleVisibility = (slug) => {
    const next = (() => {
      const hidden = { ...(prefs.hidden || {}) };
      hidden[slug] = !hidden[slug];
      return { ...prefs, hidden };
    })();
    pushPrefs(next);
  };

  const onSave = () => {};
  const onReset = () => {
    if (!user?.ft_login) return;
    const p = { order: [], hidden: {} };
    setPrefs(p);
    savePrefs(user.ft_login, p);
  };

  const deleteAccount = async () => {
    if (!window.confirm('Delete your account permanently? This action cannot be undone.')) return;
    const res = await fetchWithAuth('/api/v1/users/me', { method: 'DELETE' });
    if (res && (res.status === 204 || res.ok)) window.location.assign('/login');
  };

  const revokeSession = async (sid) => {
    const res = await fetchWithAuth(`/api/v1/users/me/sessions/${encodeURIComponent(sid)}`, { method: 'DELETE' });
    if (!res) return;
    // If current session, backend cleared cookie; redirect to login
    if (sessions.find(s => s.id === sid)?.is_current) {
      window.location.assign('/login');
      return;
    }
    // Otherwise refresh list
    const r = await fetchWithAuth('/api/v1/users/me/sessions');
    const arr = r && r.ok ? await r.json() : [];
    setSessions(Array.isArray(arr) ? arr : []);
  };

  const revokeAllSessions = async () => {
    const res = await fetchWithAuth(`/api/v1/users/me/sessions`, { method: 'DELETE' });
    if (!res) return;
    window.location.assign('/login');
  };

  const parseUA = (ua) => {
    if (!ua) return { label: 'Unknown device' };
    const OS = /Windows|Mac OS X|Linux|Android|iPhone|iPad/.exec(ua)?.[0] || '';
    const brMatch = /(Chrome|Firefox|Safari|Edg|Opera)\/([\d\.]+)/.exec(ua);
    const Browser = brMatch ? `${brMatch[1]} ${brMatch[2]}` : '';
    const label = [Browser, OS].filter(Boolean).join(' on ');
    return { label: label || ua };
  };

  const displayRoleName = (name) => (/admin/i.test(name) ? 'Admin' : name);

  return (
    <div className="user-settings-page">
      <header className="usp-header">
        {user && (
          <>
            <div className="usp-header-left">
              <img className="usp-avatar" src={user.ft_photo} alt={user.ft_login} />
              <div className="usp-identity">
                <div className="usp-login">{user.ft_login}</div>
                <div className="usp-roles-chips">
                  {roles.map(r => (
                    <span key={r.id} className="role-chip" style={getReadableStyles(r.color)}>
                      {displayRoleName(r.name)}
                    </span>
                  ))}
                </div>
              </div>
            </div>
            <div className="usp-header-actions">
              <button className="btn-danger" onClick={deleteAccount}>Delete account</button>
            </div>
          </>
        )}
      </header>

      <section className="usp-card">
        <div className="usp-card-title">Sidebar</div>
        <div className="usp-card-sub">Drag to reorder; toggle eye to hide/show.</div>
        <ul className="usp-pages">
          {orderedPages.map(p => (
            <li key={p.slug}
                className={`usp-page ${p._hidden ? 'is-hidden' : ''}`}
                draggable
                onDragStart={onDragStart(p.slug)}
                onDragOver={onDragOver(p.slug)}
                onDrop={onDrop(p.slug)}>
              <span className="usp-handle" aria-hidden>≡</span>
              <span className={`usp-page-name ${p._hidden ? 'hidden' : ''}`}>{p.name}</span>
              {p._hidden && <span className="usp-chip usp-chip-hidden">Hidden</span>}
              <button
                className={`usp-eye ${p._hidden ? 'off' : 'on'}`}
                title={p._hidden ? 'Show' : 'Hide'}
                onClick={() => toggleVisibility(p.slug)}>
                {p._hidden ? '🙈' : '👁️'}
              </button>
            </li>
          ))}
        </ul>
        <div className="usp-actions">
          <span className={`usp-save-indicator ${saving ? 'saving' : 'saved'}`}>{saving ? 'Saving…' : 'Saved'}</span>
          <div className="usp-actions-spacer" />
          <button className="btn-secondary" onClick={onReset}>Reset</button>
        </div>
      </section>

      {/* Account section removed; roles and delete action moved to header */}

      <section className="usp-card">
        <div className="usp-card-title">Sessions</div>
        {loadingSessions ? (
          <div className="usp-muted">Loading sessions…</div>
        ) : sessions.length === 0 ? (
          <div className="usp-muted">No sessions</div>
        ) : (
          <div className="usp-sessions">
            {sessions.map(s => (
              <div key={s.id} className={`usp-session ${s.is_current ? 'current' : ''}`}>
                <div className="usp-session-main">
                  <div className="usp-session-ua" title={s.user_agent}>{parseUA(s.user_agent).label}</div>
                  <div className="usp-session-meta">
                    <span>IP: {s.ip || '—'}</span>
                    <span>Last seen: {s.last_seen ? new Date(s.last_seen).toLocaleString() : '—'}</span>
                    <span>Expires: {s.expires_at ? new Date(s.expires_at).toLocaleString() : '—'}</span>
                  </div>
                </div>
                <div className="usp-session-actions">
                  {s.is_current && <span className="usp-current-badge">Current</span>}
                  <button className="btn-danger" onClick={() => revokeSession(s.id)}>Revoke</button>
                </div>
              </div>
            ))}
            <div className="usp-actions">
              <button className="btn-outline-danger" onClick={revokeAllSessions}>Revoke all sessions</button>
            </div>
          </div>
        )}
      </section>
    </div>
  );
}
