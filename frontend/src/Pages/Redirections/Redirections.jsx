import React, { useCallback, useEffect, useState } from 'react';
import Button from 'Global/Button/Button';
import RoleBadge from 'Global/RoleBadge/RoleBadge';
import './Redirections.css';
import { fetchWithAuth } from 'Global/utils/Auth';
import { getModulesDomain } from '../../utils/modules';
import RedirectionIconModal from './Components/RedirectionIconModal/RedirectionIconModal';
import RedirectionRolesModal from './Components/RedirectionRolesModal/RedirectionRolesModal';
import RedirectionVisibilityModal from './Components/RedirectionVisibilityModal/RedirectionVisibilityModal';

export default function Redirections() {
  const [redirections, setRedirections] = useState([]);
  const [edits, setEdits] = useState({});
  const [isSaving, setIsSaving] = useState(false);
  const [iconTarget, setIconTarget] = useState(null);
  const [rolesTarget, setRolesTarget] = useState(null);
  const [visibilityTarget, setVisibilityTarget] = useState(null);
  const modulesDomain = getModulesDomain();
  const hasUnsaved = Object.values(edits).some(e => e.dirty);

  const slugify = (value) => String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');

  const fetchRedirections = useCallback(async () => {
    try {
      const res = await fetchWithAuth('/api/v1/admin/redirections');
      const data = await res.json();
      const list = (Array.isArray(data) ? data : []).map(r => ({
        id: r.id,
        slug: r.slug,
        name: r.name,
        targetURL: r.target_url || '',
        needAuth: !!r.need_auth,
        isVisible: r.is_visible !== false,
        icon_url: r.icon_url,
        roles: Array.isArray(r.roles) ? r.roles : [],
        forbiddenRoles: Array.isArray(r.forbidden_roles) ? r.forbidden_roles : [],
        isNew: false,
        slugAuto: false,
      }));
      setRedirections(list);

      const initial = {};
      list.forEach(r => {
        initial[r.id] = { ...r, dirty: false };
      });
      setEdits(initial);
    } catch (err) {
      console.error('Fetch failed:', err);
    }
  }, []);

  useEffect(() => {
    fetchRedirections();
  }, [fetchRedirections]);

  const handleAddRow = () => {
    const tempId = `new-${Date.now()}`;
    const newRow = {
      id: tempId,
      slug: '',
      name: '',
      targetURL: '',
      needAuth: true,
      isVisible: true,
      icon_url: '',
      roles: [],
      forbiddenRoles: [],
      isNew: true,
      slugAuto: true,
    };

    setRedirections(rs => [...rs, newRow]);
    setEdits(e => ({
      ...e,
      [tempId]: { ...newRow, dirty: true }
    }));
  };

  const handleChange = (id, field, value) => {
    setEdits(e => ({
      ...e,
      [id]: {
        ...e[id],
        [field]: value,
        dirty: true,
      }
    }));
  };

  const handleNameChange = (id, value) => {
    setEdits((prev) => {
      const current = prev[id] || {};
      const next = {
        ...current,
        name: value,
        dirty: true,
      };
      if (current.isNew && current.slugAuto !== false) {
        next.slug = slugify(value);
      }
      return {
        ...prev,
        [id]: next,
      };
    });
  };

  const handleSlugChange = (id, value) => {
    setEdits((prev) => ({
      ...prev,
      [id]: {
        ...prev[id],
        slug: slugify(value),
        slugAuto: false,
        dirty: true,
      },
    }));
  };

  const handleVisibilitySave = (id, next) => {
    setEdits((prev) => {
      const current = prev[id] || {};
      return {
        ...prev,
        [id]: {
          ...current,
          isVisible: next.isVisible !== false,
          needAuth: !!next.needAuth,
          dirty: true,
        },
      };
    });
  };

  const handleSave = async (id) => {
    const { name, slug, targetURL, needAuth, isVisible, isNew } = edits[id];
    if (!name || !slug || !targetURL) return;

    setIsSaving(true);
    try {
      const payload = {
        name,
        slug,
        target_url: targetURL.trim(),
        need_auth: !!needAuth,
        is_visible: !!isVisible,
      };
      if (isNew) {
        await fetchWithAuth('/api/v1/admin/redirections', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
      } else {
        await fetchWithAuth(`/api/v1/admin/redirections/${encodeURIComponent(id)}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
      }
      await fetchRedirections();
    } catch (err) {
      console.error('Save failed:', err);
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('Delete this redirection?')) return;

    const row = redirections.find(r => r.id === id);
    if (row.isNew) {
      setRedirections(rs => rs.filter(r => r.id !== id));
      setEdits(e => { const copy = { ...e }; delete copy[id]; return copy; });
    } else {
      try {
        await fetchWithAuth(`/api/v1/admin/redirections/${encodeURIComponent(id)}`, { method: 'DELETE' });
        await fetchRedirections();
      } catch (err) {
        console.error('Delete failed:', err);
      }
    }
  };

  useEffect(() => {
    const handleBeforeUnload = e => {
      if (!hasUnsaved) return;
      e.preventDefault();
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
        <h3>Redirections</h3>
        <Button label="Add Redirection" color="green" onClick={handleAddRow} />
      </div>

      {redirections.length === 0 ? (
        <div className="no-pages">No redirections added yet.</div>
      ) : (
        <div className="page-table-wrap">
          <table className="page-table">
            <thead>
              <tr>
                <th className="icon-col">Icon</th>
                <th>Nom</th>
                <th>Slug</th>
                <th className="url-col">Target URL</th>
                <th className="visibility-col">Visibility</th>
                <th className="roles-col">Role</th>
                <th className="actions-col">Actions</th>
              </tr>
            </thead>
            <tbody>
              {redirections.map(({ id }) => {
                const edit = edits[id] || {};
                const missingName = !(edit.name || '').trim();
                const missingSlug = !(edit.slug || '').trim();
                const missingURL = !(edit.targetURL || '').trim();
                const visibilitySummary = [
                  edit.isVisible !== false ? 'Sidebar on' : 'Sidebar off',
                  edit.needAuth ? 'Login on' : 'Login off',
                ].join(' • ');
                return (
                  <tr
                    key={id}
                    className={`page-item${edit.dirty ? ' dirty' : ''}`}
                  >
                    <td className="page-cell page-icon-cell">
                      <img
                        src={edit.icon_url || '/icons/modules.png'}
                        className="page-icon-preview"
                        alt="icon"
                        title={edit.isNew ? 'Save the redirection first to set an icon' : 'Click to change icon'}
                        onClick={() => !edit.isNew && setIconTarget(edit)}
                      />
                    </td>
                    <td className="page-cell">
                      <input
                        className={`page-text-input${missingName ? ' missing-field' : ''}`}
                        type="text"
                        placeholder="Display name"
                        value={edit.name || ''}
                        onChange={e => handleNameChange(id, e.target.value)}
                      />
                    </td>
                    <td className="page-cell">
                      <div className={`slug-input${missingSlug ? ' missing-field' : ''}`}>
                        <input
                          className="page-text-input slug-input-field"
                          type="text"
                          placeholder="slug"
                          value={edit.slug || ''}
                          onChange={e => handleSlugChange(id, e.target.value)}
                        />
                        <span className="slug-suffix">.{modulesDomain}</span>
                      </div>
                    </td>
                    <td className="page-cell">
                      <input
                        className={`page-text-input${missingURL ? ' missing-field' : ''}`}
                        type="text"
                        placeholder="https://exam-master.42nice.fr"
                        value={edit.targetURL || ''}
                        onChange={e => handleChange(id, 'targetURL', e.target.value)}
                      />
                    </td>
                    <td className="page-cell">
                      <button
                        type="button"
                        className="page-visibility-button"
                        onClick={() => setVisibilityTarget(edit)}
                        title="Configure sidebar visibility and login requirement"
                      >
                        <span className="page-visibility-summary">{visibilitySummary}</span>
                      </button>
                    </td>
                    <td className="page-cell">
                      <button
                        type="button"
                        className="page-role-cell"
                        onClick={() => !edit.isNew && setRolesTarget(edit)}
                        disabled={edit.isNew}
                        title={edit.isNew ? 'Save the redirection first to assign roles' : 'Open role editor'}
                      >
                        <div className="page-role-preview">
                          {(edit.roles || []).length === 0 ? (
                            <span className="page-role-empty">Public</span>
                          ) : (
                            (edit.roles || []).map((role) => (
                              <RoleBadge key={role.id} role={role}>
                                {role.name}
                              </RoleBadge>
                            ))
                          )}
                        </div>
                      </button>
                    </td>
                    <td className="page-cell">
                      <div className="page-actions">
                        <Button
                          label="Save"
                          color="green"
                          onClick={() => handleSave(id)}
                          disabled={!edit.dirty || isSaving || missingName || missingSlug || missingURL}
                        />
                        <Button
                          label="Delete"
                          color="red"
                          onClick={() => handleDelete(id)}
                        />
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {iconTarget && (
        <RedirectionIconModal
          redirectionId={iconTarget.id}
          currentIcon={(edits[iconTarget.id] && edits[iconTarget.id].icon_url) || ''}
          onClose={() => setIconTarget(null)}
          onUpdated={() => { setIconTarget(null); fetchRedirections(); }}
        />
      )}
      {rolesTarget && (
        <RedirectionRolesModal
          open={!!rolesTarget}
          redirection={rolesTarget}
          onClose={() => setRolesTarget(null)}
          onUpdated={fetchRedirections}
        />
      )}
      {visibilityTarget && (
        <RedirectionVisibilityModal
          open={!!visibilityTarget}
          value={visibilityTarget}
          onClose={() => setVisibilityTarget(null)}
          onSave={(next) => {
            handleVisibilitySave(visibilityTarget.id, next);
            setVisibilityTarget(null);
          }}
        />
      )}
    </div>
  );
}
