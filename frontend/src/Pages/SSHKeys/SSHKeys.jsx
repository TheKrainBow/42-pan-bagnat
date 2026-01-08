import React, { useEffect, useRef, useState } from 'react';
import Button from 'Global/Button/Button';
import Field from 'Global/Field/Field';
import ModuleSimpleBadge from 'Global/ModuleSimpleBadge/ModuleSimpleBadge';
import { fetchWithAuth } from 'Global/utils/Auth';
import { toast } from 'react-toastify';
import { useNavigate } from 'react-router-dom';
import './SSHKeys.css';

const SSHKeys = () => {
  const [keys, setKeys] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [newName, setNewName] = useState('');
  const [newPrivate, setNewPrivate] = useState('');
  const [showPrivateConfirm, setShowPrivateConfirm] = useState(false);
  const [revealed, setRevealed] = useState({});
  const [creating, setCreating] = useState(false);
  const [modulesModal, setModulesModal] = useState({ open: false, key: null, modules: [], loading: false, error: '' });
  const [historyModal, setHistoryModal] = useState({ open: false, key: null, events: [], loading: false, error: '' });
  const historyConsoleRef = useRef(null);
  const navigate = useNavigate();

  const loadKeys = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await fetchWithAuth('/api/v1/admin/ssh-keys');
      if (!res.ok) throw new Error('Failed to fetch SSH keys');
      const data = await res.json();
      setKeys(Array.isArray(data?.ssh_keys) ? data.ssh_keys : []);
    } catch (err) {
      setError(err.message || 'Failed to load SSH keys');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadKeys();
  }, []);

  const toggleReveal = (id) => {
    setRevealed(prev => ({ ...prev, [id]: !prev[id] }));
  };

  const handleDelete = async (id) => {
    if (!window.confirm('Delete this SSH key? Modules using it will fail to deploy.')) return;
    try {
      const res = await fetchWithAuth(`/api/v1/admin/ssh-keys/${id}`, { method: 'DELETE' });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to delete key');
      }
      toast.success('SSH key deleted');
      loadKeys();
    } catch (err) {
      toast.error(err.message || 'Unable to delete key');
    }
  };

  const handleRegenerate = async (id) => {
    if (!window.confirm('Regenerate this SSH key? You will need to update every repository using it.')) return;
    try {
      const res = await fetchWithAuth(`/api/v1/admin/ssh-keys/${id}/regenerate`, { method: 'POST' });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to regenerate key');
      }
      toast.success('SSH key regenerated');
      loadKeys();
    } catch (err) {
      toast.error(err.message || 'Unable to regenerate key');
    }
  };

  const createKey = async () => {
    if (!newName.trim()) {
      toast.error('Name is required');
      return;
    }
    setCreating(true);
    try {
      const res = await fetchWithAuth('/api/v1/admin/ssh-keys', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newName.trim(), ssh_private_key: newPrivate.trim() }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to create key');
      }
      toast.success('SSH key created');
      setShowModal(false);
      setShowPrivateConfirm(false);
      setNewName('');
      setNewPrivate('');
      loadKeys();
    } catch (err) {
      toast.error(err.message || 'Unable to create key');
    } finally {
      setCreating(false);
    }
  };

  const handleCreate = () => {
    if (!newName.trim()) {
      toast.error('Name is required');
      return;
    }
    if (newPrivate.trim()) {
      setShowPrivateConfirm(true);
      return;
    }
    createKey();
  };

  const openModulesModal = async (key) => {
    setModulesModal({ open: true, key, modules: [], loading: true, error: '' });
    try {
      const res = await fetchWithAuth(`/api/v1/admin/ssh-keys/${key.id}/modules`);
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to load modules');
      }
      const mods = await res.json();
      setModulesModal({ open: true, key, modules: Array.isArray(mods?.modules) ? mods.modules : [], loading: false, error: '' });
    } catch (err) {
      setModulesModal(prev => ({ ...prev, loading: false, error: err.message || 'Failed to load modules' }));
    }
  };

  const openHistoryModal = async (key) => {
    setHistoryModal({ open: true, key, events: [], loading: true, error: '' });
    try {
      const eventsRes = await fetchWithAuth(`/api/v1/admin/ssh-keys/${key.id}/events`);
      if (!eventsRes.ok) {
        const text = await eventsRes.text();
        throw new Error(text || 'Failed to load history');
      }
      const eventsData = await eventsRes.json();
      setHistoryModal({ open: true, key, events: Array.isArray(eventsData?.events) ? eventsData.events : [], loading: false, error: '' });
    } catch (err) {
      setHistoryModal(prev => ({ ...prev, loading: false, error: err.message || 'Failed to load history' }));
    }
  };

  useEffect(() => {
    if (!historyModal.open) return;
    if (historyConsoleRef.current) {
      historyConsoleRef.current.scrollTop = historyConsoleRef.current.scrollHeight;
    }
  }, [historyModal.events, historyModal.open]);

  const closeModulesModal = () => setModulesModal({ open: false, key: null, modules: [], loading: false, error: '' });
  const closeHistoryModal = () => setHistoryModal({ open: false, key: null, events: [], loading: false, error: '' });

  return (
    <div className="ssh-page">
      <div className="ssh-header">
        <div>
          <h2>SSH Keys</h2>
          <p>Manage reusable deploy keys for your modules.</p>
        </div>
        <Button label="Add Key" color="blue" onClick={() => setShowModal(true)} />
      </div>

      {loading ? (
        <div className="ssh-empty">Loading SSH keys…</div>
      ) : error ? (
        <div className="ssh-error">{error}</div>
      ) : keys.length === 0 ? (
        <div className="ssh-empty">No SSH keys yet. Create one to reuse across modules.</div>
      ) : (
        <div className="ssh-list">
          {keys.map(key => (
            <div className="ssh-card" key={key.id}>
              <div className="ssh-card-header">
                <div className="ssh-card-title">
                  <strong>{key.name}</strong>
                  {key.created_by && (
                    <div className="ssh-created-by">
                      <span>Created by</span>
                      {key.created_by.type === 'user' && key.created_by.user ? (
                        <div className="ssh-creator" title={key.created_by.user.login}>
                          <img src={key.created_by.user.photo_url || '/icons/users.png'} alt="" />
                          <span>{key.created_by.user.login}</span>
                        </div>
                      ) : key.created_by.type === 'module' && key.created_by.module ? (
                        <div className="ssh-creator" title={key.created_by.module.name}>
                          <img src={key.created_by.module.icon_url || '/icons/modules.png'} alt="" />
                          <span>{key.created_by.module.name}</span>
                        </div>
                      ) : (
                        <div className="ssh-creator"><span>Unknown</span></div>
                      )}
                    </div>
                  )}
                </div>
                <div className="ssh-card-actions">
                  <button className="ssh-usage-link" onClick={() => openModulesModal(key)}>
                    Used by {key.usage_count || 0} module{(key.usage_count || 0) === 1 ? '' : 's'}
                  </button>
                  <button className="ssh-history-link" onClick={() => openHistoryModal(key)}>
                    Last used at {key.last_used_at ? new Date(key.last_used_at).toLocaleString() : 'Never'}
                  </button>
                  <Button label="Regenerate" color="blue" onClick={() => handleRegenerate(key.id)} />
                  <Button
                    label="Delete"
                    color="red"
                    onClick={() => handleDelete(key.id)}
                    disabled={(key.usage_count || 0) >= 1}
                    disabledMessage={`SSH key is still assigned to ${key.usage_count || 0} modules`}
                  />
                </div>
              </div>
              <div className="ssh-card-body">
                <span className="ssh-label">Public key</span>
                <div className="ssh-key-row">
                  <code>{revealed[key.id] ? key.public_key : '••••••••••••••••••••••••••••••••'}</code>
                  <button className="ssh-eye" onClick={() => toggleReveal(key.id)}>
                    {revealed[key.id] ? 'Hide' : 'Show'}
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {showModal && (
        <div className="ssh-modal-overlay">
          <div className="ssh-modal">
            <h3>New SSH Key</h3>
            <Field
              label="Name"
              value={newName}
              onChange={e => setNewName(e.target.value)}
              placeholder="shared-deploy-key"
              required={true}
            />
            <Field
              label="Private Key (optional)"
              value={newPrivate}
              onChange={e => setNewPrivate(e.target.value)}
              placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
              multiline={true}
              rows={5}
            />
            <p className="ssh-modal-tip">Leave the private key empty to let Pan Bagnat generate a new pair.</p>
            <div className="ssh-modal-actions">
              <Button label="Cancel" color="gray" onClick={() => { setShowModal(false); setShowPrivateConfirm(false); setNewName(''); setNewPrivate(''); }} />
              <Button label={creating ? 'Creating…' : 'Create'} color="blue" onClick={handleCreate} disabled={creating} />
            </div>
          </div>
        </div>
      )}

      {showPrivateConfirm && (
        <div className="ssh-modal-overlay" onClick={() => !creating && setShowPrivateConfirm(false)}>
          <div className="ssh-confirm-modal" onClick={e => e.stopPropagation()}>
            <p className="ssh-confirm-title">
              <span className="ssh-confirm-emoji">⚠️</span>
              Shareable private keys
              <span className="ssh-confirm-emoji">⚠️</span>
            </p>
            <p className="ssh-confirm-text">
              Private keys saved in Pan Bagnat are shared across every admin. Other admins can view the generated public
              key and use this credential for future module imports.
            </p>
            <p className="ssh-confirm-note">Only continue if this key is meant to be shared.</p>
            <div className="ssh-confirm-buttons">
              <Button label="Cancel" color="gray" onClick={() => setShowPrivateConfirm(false)} disabled={creating} />
              <Button
                label={creating ? 'Creating…' : 'Confirm'}
                color="blue"
                onClick={() => { setShowPrivateConfirm(false); createKey(); }}
                disabled={creating}
              />
            </div>
          </div>
        </div>
      )}

      {modulesModal.open && (
        <div className="ssh-modal-overlay" onClick={closeModulesModal}>
          <div className="ssh-modal ssh-modules-modal" onClick={e => e.stopPropagation()}>
            <div className="ssh-modals-header">
              <h3>Modules using {modulesModal.key?.name}</h3>
              <button className="ssh-eye" onClick={closeModulesModal}>✕</button>
            </div>
            {modulesModal.loading ? (
              <div className="ssh-empty">Loading modules…</div>
            ) : modulesModal.error ? (
              <div className="ssh-error">{modulesModal.error}</div>
            ) : (
              <div className="ssh-modules-list">
                {modulesModal.modules.length === 0 ? (
                  <div className="ssh-empty">No modules linked to this key.</div>
                ) : (
                  modulesModal.modules.map(mod => (
                    <ModuleSimpleBadge
                      key={mod.id}
                      module={mod}
                      onClick={() => {
                        navigate(`/admin/modules/${mod.id}?tab=settings`);
                        closeModulesModal();
                      }}
                    />
                  ))
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {historyModal.open && (
        <div className="ssh-modal-overlay" onClick={closeHistoryModal}>
          <div className="ssh-modal ssh-history-modal" onClick={e => e.stopPropagation()}>
            <div className="ssh-modals-header">
              <h3>Usage history — {historyModal.key?.name}</h3>
              <button className="ssh-eye" onClick={closeHistoryModal}>✕</button>
            </div>
            {historyModal.loading ? (
              <div className="ssh-empty">Loading history…</div>
            ) : historyModal.error ? (
              <div className="ssh-error">{historyModal.error}</div>
            ) : (
              <div className="ssh-history-console" ref={historyConsoleRef}>
                {historyModal.events.length === 0 ? (
                  <div className="ssh-empty">No events recorded.</div>
                ) : (
                  [...historyModal.events].reverse().map(ev => {
                    const msg = ev.message || '';
                    const lower = msg.toLowerCase();
                    const lineClass = lower.includes('used') ? 'ssh-line-action'
                      : lower.includes('created') ? 'ssh-line-create'
                      : lower.includes('regenerated') ? 'ssh-line-regenerate'
                      : 'ssh-line-default';
                    const userLabel = ev.actor_user?.login || 'system';
                    const moduleLabel = ev.actor_module?.name || 'Deleted';
                    const moduleDeleted = !ev.actor_module?.name;
                    return (
                      <div key={ev.id} className={`ssh-log-line ${lineClass}`}>
                        <span className="ssh-log-time">[{new Date(ev.created_at).toLocaleString()}]</span>
                        <span className="ssh-log-user"> [{userLabel}]</span>
                        <span className={`ssh-log-module${moduleDeleted ? ' deleted' : ''}`}> [{moduleLabel}]</span>
                        <span className="ssh-log-text"> {msg}</span>
                      </div>
                    );
                  })
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default SSHKeys;
