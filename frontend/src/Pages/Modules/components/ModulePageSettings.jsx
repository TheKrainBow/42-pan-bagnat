import React, { useEffect, useState, useRef } from 'react';
import Button from 'Global/Button';
import './ModulePageSettings.css';

export default function ModulePageSettings({ moduleId }) {
  const [pages, setPages] = useState([]);
  const [newPage, setNewPage] = useState({ name: '', url: '' });
  const [isSaving, setIsSaving] = useState(false);
  const fetchedRef = useRef(false);

  const fetchPages = () => {
    if (fetchedRef.current) return;
    fetchedRef.current = true;
    fetch(`http://localhost:8080/api/v1/modules/${moduleId}/pages`)
      .then(r => r.json())
      .then(data => {
        const pageList = data.pages || [];
        setPages(pageList);
        const editMap = {};
        pageList.forEach(p => editMap[p.name] = p.url);
        setUrlEdits(editMap);
      })
      .catch(console.error);
  };

  useEffect(fetchPages, [moduleId]);

  const handleAdd = async () => {
    if (!newPage.name || !newPage.url) return;
    setIsSaving(true);
    try {
      await fetch(`http://localhost:8080/api/v1/modules/${moduleId}/pages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newPage),
      });
      setNewPage({ name: '', url: '' });
      fetchPages();
    } catch (err) {
      console.error('Add failed:', err);
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async (name) => {
    if (!confirm(`Delete page "${name}"?`)) return;
    await fetch(`http://localhost:8080/api/v1/modules/${moduleId}/pages/${name}`, {
      method: 'DELETE',
    });
    fetchPages();
  };

  return (
    <div className="front-pages-panel">
      <h3>Front Pages</h3>
      <ul className="page-list">
        {pages.map(({ name, url }) => (
          <li key={name} className="page-item">
            <div className="page-info">
              <span className="page-name">{name}</span>
              <span className="page-url">{url}</span>
            </div>
            <Button label="Delete" color="red" onClick={() => handleDelete(name)} />
          </li>
        ))}
      </ul>

      <div className="add-page-form">
        <input
          type="text"
          placeholder="Page name"
          value={newPage.name}
          onChange={e => setNewPage({ ...newPage, name: e.target.value })}
        />
        <input
          type="text"
          placeholder="Page URL"
          value={newPage.url}
          onChange={e => setNewPage({ ...newPage, url: e.target.value })}
        />
        <Button
          label={isSaving ? 'Adding...' : 'Add Page'}
          color="green"
          onClick={handleAdd}
          disabled={isSaving}
        />
      </div>
    </div>
  );
}
