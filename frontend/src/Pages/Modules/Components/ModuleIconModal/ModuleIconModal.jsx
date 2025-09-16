import React, { useState } from 'react';
import './ModuleIconModal.css';
import { fetchWithAuth } from 'Global/utils/Auth';

export default function ModuleIconModal({ moduleId, pageId, currentIcon, onClose, onUpdated }) {
  const isPage = !!pageId;
  const [mode, setMode] = useState(isPage ? 'inherit' : 'upload'); // 'inherit'|'upload'|'url'|'repo'
  const [busy, setBusy] = useState(false);
  const [url, setUrl] = useState('');
  const [repoPath, setRepoPath] = useState('');

  const base = isPage
    ? `/api/v1/admin/modules/${moduleId}/pages/${pageId}/icon`
    : `/api/v1/admin/modules/${moduleId}/icon`;

  const handleFile = async (file) => {
    if (!file) return;
    setBusy(true);
    try {
      const fd = new FormData();
      fd.append('file', file);
      const res = await fetchWithAuth(`${base}/upload`, { method: 'POST', body: fd });
      if (res && res.ok) {
        onUpdated?.();
        onClose?.();
      }
    } finally {
      setBusy(false);
    }
  };

  const handleUrl = async () => {
    if (!url) return;
    setBusy(true);
    try {
      const res = await fetchWithAuth(`${base}/url`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ url })
      });
      if (res && res.ok) { onUpdated?.(); onClose?.(); }
    } finally { setBusy(false); }
  };

  const handleRepo = async () => {
    if (!repoPath) return;
    setBusy(true);
    try {
      const res = await fetchWithAuth(`${base}/from-repo`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ path: repoPath })
      });
      if (res && res.ok) { onUpdated?.(); onClose?.(); }
    } finally { setBusy(false); }
  };

  const handleInherit = async () => {
    if (!isPage) return;
    setBusy(true);
    try {
      const res = await fetchWithAuth(`${base}/clear`, { method: 'POST' });
      if (res && res.ok) { onUpdated?.(); onClose?.(); }
    } finally { setBusy(false); }
  };

  const onDrop = async (e) => {
    e.preventDefault();
    e.stopPropagation();
    const dt = e.dataTransfer;
    if (dt.files && dt.files.length > 0) { await handleFile(dt.files[0]); return; }
    const uri = dt.getData('text/uri-list') || dt.getData('text/plain');
    if (uri) { setMode('url'); setUrl(uri.trim()); }
  };

  return (
    <div className="modal-backdrop" onMouseDown={onClose}>
      <div className="modal" onMouseDown={e=>e.stopPropagation()}>
        <h3 className="modal-title">Change Module Icon</h3>
        <div className="mi-body">
          <div className="mi-preview">
            <img src={currentIcon || '/icons/modules.png'} alt="icon" />
          </div>
          <div className="mi-controls" onDragOver={e=>{e.preventDefault();}} onDrop={onDrop}>
            <div className="mi-row">
              <label>Source</label>
              <select value={mode} onChange={e=>setMode(e.target.value)}>
                {isPage && <option value="inherit">Use Module Icon</option>}
                <option value="upload">Upload file</option>
                <option value="url">From URL</option>
                <option value="repo">From repo path</option>
              </select>
            </div>
            {mode === 'inherit' && isPage && (
              <div className="mi-row">
                <button className="btn-secondary" disabled={busy} onClick={handleInherit}>Use Module Icon</button>
              </div>
            )}
            {mode === 'upload' && (
              <div className="mi-row">
                <input type="file" accept="image/*" disabled={busy} onChange={e=>handleFile(e.target.files?.[0])} />
              </div>
            )}
            {mode === 'url' && (
              <div className="mi-row">
                <input type="text" placeholder="https://…/icon.png" value={url} onChange={e=>setUrl(e.target.value)} />
                <button disabled={busy || !url} onClick={handleUrl}>Use</button>
              </div>
            )}
            {mode === 'repo' && (
              <div className="mi-row">
                <input type="text" placeholder="assets/icon.png" value={repoPath} onChange={e=>setRepoPath(e.target.value)} />
                <button disabled={busy || !repoPath} onClick={handleRepo}>Use</button>
              </div>
            )}
            <div className="mi-drop">Drag & drop an image or link here (max 50 MB)</div>
          </div>
        </div>
        <div className="modal-actions">
          <button className="btn-secondary" onClick={onClose} disabled={busy}>Close</button>
        </div>
      </div>
    </div>
  );
}
