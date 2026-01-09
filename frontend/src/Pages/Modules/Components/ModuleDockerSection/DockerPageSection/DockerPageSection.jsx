import React, { useEffect, useState } from 'react';
import Button from 'Global/Button/Button';
import './DockerPageSection.css';
import { fetchWithAuth } from 'Global/utils/Auth';
import PageIconModal from 'Pages/Modules/Components/ModuleIconModal/ModuleIconModal';

export default function ModulePageSection({ moduleId }) {
  const [pages, setPages] = useState([]);            // holds both existing & new rows
  const [edits, setEdits] = useState({});            // keyed by row.id
  const [isSaving, setIsSaving] = useState(false);
  const [iconTarget, setIconTarget] = useState(null);
  const [proxyStatuses, setProxyStatuses] = useState({});
  const [networks, setNetworks] = useState([]);
  const hasUnsaved = Object.values(edits).some(e => e.dirty);

  const fetchNetworks = async () => {
    try {
      const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/networks`);
      if (!res) return;
      const data = await res.json();
      setNetworks(data.networks || []);
    } catch (err) {
      console.error('Failed to fetch module networks:', err);
      setNetworks([]);
    }
  };

  useEffect(() => {
    fetchNetworks();
  }, [moduleId]);

  const fetchProxyStatus = async (slug) => {
    if (!slug) return;
    setProxyStatuses(prev => ({ ...prev, [slug]: { loading: true } }));
    try {
      const res = await fetchWithAuth(`/module-page/_status/${encodeURIComponent(slug)}`);
      if (!res) return;
      if (!res.ok) {
        setProxyStatuses(prev => ({ ...prev, [slug]: { loading: false, error: `HTTP ${res.status}` } }));
        return;
      }
      const data = await res.json();
      setProxyStatuses(prev => ({ ...prev, [slug]: { loading: false, ...data } }));
    } catch (err) {
      setProxyStatuses(prev => ({ ...prev, [slug]: { loading: false, error: err.message } }));
    }
  };

  const handleProxyReconnect = async (slug) => {
    if (!slug) return;
    try {
      const res = await fetchWithAuth(`/module-page/_status/${encodeURIComponent(slug)}`, {
        method: 'POST'
      });
      if (!res) return;
      await fetchProxyStatus(slug);
    } catch (err) {
      console.error('Reconnect failed:', err);
    }
  };

  // load existing pages
  const fetchPages = async () => {
    try {
      setProxyStatuses({});
      const res = await fetchWithAuth(`
        /api/v1/admin/modules/${moduleId}/pages
      `);
      if (!res) return;
      const data = await res.json();
      const list = (data.pages || []).map(p => ({
        id: p.id,
        slug: p.slug,
        name: p.name,
        url: p.url,
        isPublic: p.is_public,
        icon_url: p.icon_url,
        network: p.network_name || '',
        isNew: false,
        moduleCheck: p.module_check || null
      }));
      setPages(list);

      // initialize edit state
      const initial = {};
      list.forEach(p => {
        initial[p.id] = { ...p, dirty: false };
      });
      setEdits(initial);

      list.filter(p => p.slug).forEach(p => fetchProxyStatus(p.slug));
    } catch (err) {
      console.error('Fetch failed:', err);
    }
  };

  // effect to fetch pages on mount or moduleId change
  useEffect(() => {
    // wrap async call inside effect
    fetchPages();
  }, [moduleId]);

  // begin a new blank row
  const handleAddRow = () => {
    const tempId = `new-${Date.now()}`;
    const newRow = { id: tempId, slug: '', name: '', url: '', isPublic: false, icon_url: '', network: '', isNew: true, moduleCheck: null };

    setPages(ps => [...ps, newRow]);
    setEdits(e => ({
      ...e,
      [tempId]: { ...newRow, dirty: true }
    }));
  };

  // on any field change
  const handleChange = (id, field, value) => {
    setEdits(e => ({
      ...e,
      [id]: {
        ...e[id],
        [field]: value,
        dirty: true
      }
    }));
  };

  // save either POST (new) or PATCH (existing)
  const handleSave = async (id) => {
    const { name, url, isPublic, isNew } = edits[id];
    if (!name || !url) return;

    setIsSaving(true);
    try {
      const payload = {
        name,
        url,
        is_public: isPublic,
        network_name: (edits[id].network || '').trim() || null,
      };
      if (isNew) {
        await fetchWithAuth(
          `/api/v1/admin/modules/${moduleId}/pages`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
          }
        );
      } else {
        await fetchWithAuth(
          `/api/v1/admin/modules/${moduleId}/pages/${encodeURIComponent(id)}`,
          {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
          }
        );
      }
      await fetchPages();
    } catch (err) {
      console.error('Save failed:', err);
    } finally {
      setIsSaving(false);
    }
  };

  // delete row
  const handleDelete = async (id) => {
    if (!window.confirm('Delete this page?')) return;

    const row = pages.find(p => p.id === id);
    if (row.isNew) {
      setPages(ps => ps.filter(p => p.id !== id));
      setEdits(e => { const copy = { ...e }; delete copy[id]; return copy; });
    } else {
      try {
        await fetchWithAuth(
          `/api/v1/admin/modules/${moduleId}/pages/${encodeURIComponent(id)}`,
          { method: 'DELETE' }
        );
        await fetchPages();
      } catch (err) {
        console.error('Delete failed:', err);
      }
    }
  };

  // 1) beforeunload: catches reloads / tab closes
  useEffect(() => {
    const handleBeforeUnload = e => {
      if (!hasUnsaved) return;
      e.preventDefault();
      // Chrome requires returnValue to be set
      e.returnValue = '';
    };
    window.addEventListener('beforeunload', handleBeforeUnload);
    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload);
    };
  }, [hasUnsaved]);

  return (
    <div className="front-pages-panel">
      <div className="front-pages-header">
        <h3>Front Pages</h3>
        <Button label="Add Page" color="green" onClick={handleAddRow} />
      </div>
    
      {pages.length === 0 ? (
        <div className="no-pages">No pages added yet.</div>
      ) : (
        <ul className="page-list">
          {pages.map(({ id, slug, moduleCheck }) => {
            const edit = edits[id] || {};
            const proxy = slug ? proxyStatuses[slug] : null;
            const moduleStatus = moduleCheck
              ? (moduleCheck.ok ? 'ok' : 'ko')
              : 'pending';
            const proxyStatus = proxy
              ? (proxy.loading ? 'pending' : proxy.ok ? 'ok' : 'ko')
              : 'pending';

            return (
              <li
                key={id}
                className={`page-item${edit.dirty ? ' dirty' : ''}`}
              >
                <div className="page-info">
                  <img src={edit.icon_url || '/icons/modules.png'} className="page-icon-preview" alt="icon" title="Click to change icon" onClick={()=> setIconTarget(id)} />
                  <input
                    className="page-name-input"
                    type="text"
                    placeholder="Name"
                    value={edit.name || ''}
                    onChange={e => handleChange(id, 'name', e.target.value)}
                  />
                  <input
                    className="page-url-input"
                    type="text"
                    placeholder="URL"
                    value={edit.url || ''}
                    onChange={e => handleChange(id, 'url', e.target.value)}
                  />
                  <select
                    className={`page-network-select${edit.network && !networks.includes(edit.network) ? ' missing-network' : ''}`}
                    value={edit.network || ''}
                    onChange={e => handleChange(id, 'network', e.target.value)}
                  >
                    <option value="">No network</option>
                    {networks.map(net => (
                      <option key={net} value={net}>{net}</option>
                    ))}
                    {edit.network && !networks.includes(edit.network) && (
                      <option value={edit.network} className="missing-option">
                        {edit.network} (!)
                      </option>
                    )}
                  </select>
                  <div className="page-status">
                    <span
                      className={`status-chip ${moduleStatus}`}
                      title={moduleCheck?.details || (moduleCheck?.ok ? 'Reachable' : 'Awaiting containers')}
                    >
                      Module {moduleStatus === 'ok' ? 'OK' : moduleStatus === 'ko' ? 'KO' : '...'}
                    </span>
                    <button
                      type="button"
                      className={`status-chip proxy-button ${proxyStatus}`}
                      title={proxy?.error || proxy?.message || (proxy?.loading ? 'Checking…' : 'Awaiting response')}
                      onClick={() => proxyStatus === 'ko' && slug ? handleProxyReconnect(slug) : null}
                      disabled={proxyStatus !== 'ko' || !slug}
                    >
                      Proxy {proxyStatus === 'ok' ? 'OK' : proxyStatus === 'ko' ? 'KO' : '...'}
                    </button>
                    <label className="page-public-toggle">
                      <input
                        type="checkbox"
                        checked={edit.isPublic || false}
                        onChange={e => handleChange(id, 'isPublic', e.target.checked)}
                      />
                      Public
                    </label>
                  </div>
                </div>
                <div className="page-actions">
                  <Button
                    label="Save"
                    color="green"
                    onClick={() => handleSave(id)}
                    disabled={!edit.dirty || isSaving}
                  />
                  <Button
                    label="Delete"
                    color="red"
                    onClick={() => handleDelete(id)}
                  />
                </div>
              </li>
            );
          })}
        </ul>
      )}
      {iconTarget && (
        <PageIconModal
          moduleId={moduleId}
          pageId={iconTarget}
          currentIcon={(edits[iconTarget] && edits[iconTarget].icon_url) || ''}
          onClose={()=> setIconTarget(null)}
          onUpdated={()=> { setIconTarget(null); fetchPages(); }}
        />
      )}
    </div>
  );
}
