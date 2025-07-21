import React, { useEffect, useState } from 'react';
import Button from 'Global/Button';
import './ModulePageSettings.css';

export default function ModulePageSettings({ moduleId }) {
  const [pages, setPages] = useState([]);            // holds both existing & new rows
  const [edits, setEdits] = useState({});            // keyed by row.id
  const [isSaving, setIsSaving] = useState(false);
  const hasUnsaved = Object.values(edits).some(e => e.dirty);

  // load existing pages
  const fetchPages = async () => {
    try {
      const res = await fetch(`
        http://localhost:8080/api/v1/modules/${moduleId}/pages
      `);
      const data = await res.json();
      const list = (data.pages || []).map(p => ({
        id: p.name,        // use name as stable id for existing
        name: p.name,
        url: p.url,
        isPublic: p.isPublic,
        isNew: false
      }));
      setPages(list);

      // initialize edit state
      const initial = {};
      list.forEach(p => {
        initial[p.id] = { ...p, dirty: false };
      });
      setEdits(initial);
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
    const newRow = { id: tempId, name: '', url: '', isPublic: false, isNew: true };

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
      if (isNew) {
        await fetch(
          `http://localhost:8080/api/v1/modules/${moduleId}/pages`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, url, isPublic })
          }
        );
      } else {
        await fetch(
          `http://localhost:8080/api/v1/modules/${moduleId}/pages/${encodeURIComponent(id)}`,
          {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, url, isPublic })
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
        await fetch(
          `http://localhost:8080/api/v1/modules/${moduleId}/pages/${encodeURIComponent(id)}`,
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
          {pages.map(({ id }) => {
            const edit = edits[id] || {};
            return (
              <li
                key={id}
                className={`page-item${edit.dirty ? ' dirty' : ''}`}
              >
                <div className="page-info">
                  <input
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
                  <label>
                    <input
                      type="checkbox"
                      checked={edit.isPublic || false}
                      onChange={e => handleChange(id, 'isPublic', e.target.checked)}
                    />
                    Public
                  </label>
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
    </div>
  );
}
