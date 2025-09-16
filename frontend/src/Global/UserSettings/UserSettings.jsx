import React, { useEffect, useMemo, useState } from 'react';
import './UserSettings.css';
import { fetchWithAuth } from 'Global/utils/Auth';

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

export default function UserSettings({ open, onClose, user, pages, onPrefsChange }) {
  const login = user?.ft_login || '';
  const [prefs, setPrefs] = useState(() => loadPrefs(login));

  useEffect(() => {
    if (open) {
      setPrefs(loadPrefs(login));
    }
  }, [open, login]);

  const combined = useMemo(() => {
    const order = Array.isArray(prefs.order) ? prefs.order : [];
    const hidden = prefs.hidden || {};
    const bySlug = new Map(pages.map(p => [p.slug, p]));
    const ordered = [];
    for (const s of order) {
      if (bySlug.has(s)) ordered.push(bySlug.get(s));
      bySlug.delete(s);
    }
    // Append any new/unknown pages at the end using original order
    for (const p of pages) {
      if (!ordered.find(x => x.slug === p.slug)) ordered.push(p);
    }
    return ordered.map(p => ({ ...p, _hidden: !!hidden[p.slug] }));
  }, [pages, prefs]);

  if (!open) return null;

  const move = (slug, dir) => {
    setPrefs(prev => {
      const order = Array.isArray(prev.order) ? [...prev.order] : [];
      const effective = combined.map(p => p.slug); // current visual order
      const idx = effective.indexOf(slug);
      if (idx < 0) return prev;
      const swapWith = dir === 'up' ? idx - 1 : idx + 1;
      if (swapWith < 0 || swapWith >= effective.length) return prev;
      const newEffective = [...effective];
      [newEffective[idx], newEffective[swapWith]] = [newEffective[swapWith], newEffective[idx]];
      // Persist only the order list
      return { ...prev, order: newEffective };
    });
  };

  const toggleHidden = (slug) => {
    setPrefs(prev => {
      const hidden = { ...(prev.hidden || {}) };
      hidden[slug] = !hidden[slug];
      return { ...prev, hidden };
    });
  };

  const onSave = () => {
    savePrefs(login, prefs);
    onPrefsChange && onPrefsChange(prefs);
    onClose && onClose();
  };

  const onReset = () => {
    const p = { order: [], hidden: {} };
    setPrefs(p);
    savePrefs(login, p);
    onPrefsChange && onPrefsChange(p);
  };

  const onDisconnect = async () => {
    await fetchWithAuth('/auth/logout', { method: 'POST' });
    // Hard redirect to login
    window.location.assign('/login');
  };

  const onDeleteAccount = async () => {
    if (!window.confirm('Delete your account permanently? This action cannot be undone.')) return;
    const res = await fetchWithAuth('/api/v1/users/me', { method: 'DELETE' });
    if (res && (res.status === 204 || res.ok)) {
      window.location.assign('/login');
    }
  };

  return (
    <div className="modal-backdrop" onMouseDown={onClose}>
      <div className="modal user-settings" onMouseDown={e => e.stopPropagation()} role="dialog" aria-modal="true">
        <h3 className="modal-title">User Settings</h3>

        <section className="us-section">
          <h4 className="us-section-title">Sidebar pages</h4>
          <p className="us-section-hint">Reorder and hide pages you don’t want to see.</p>
          <ul className="us-pages">
            {combined.map(p => (
              <li key={p.slug} className="us-page-row">
                <div className="us-page-controls">
                  <button className="us-btn" onClick={() => move(p.slug, 'up')} aria-label={`Move ${p.name} up`}>▲</button>
                  <button className="us-btn" onClick={() => move(p.slug, 'down')} aria-label={`Move ${p.name} down`}>▼</button>
                </div>
                <label className="us-page-label">
                  <input type="checkbox" checked={!p._hidden} onChange={() => toggleHidden(p.slug)} />
                  <span className="us-page-name">{p.name}</span>
                </label>
              </li>
            ))}
          </ul>
        </section>

        <section className="us-section">
          <h4 className="us-section-title">Account</h4>
          <div className="us-actions">
            <button className="us-danger-outline" onClick={onDisconnect}>Disconnect</button>
            <button className="us-danger" onClick={onDeleteAccount}>Delete account</button>
          </div>
        </section>

        <div className="modal-actions">
          <button className="us-secondary" onClick={onClose}>Cancel</button>
          <button className="us-secondary" onClick={onReset}>Reset</button>
          <button className="us-primary" onClick={onSave}>Save</button>
        </div>
      </div>
    </div>
  );
}

