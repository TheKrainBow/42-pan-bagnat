import React, { useEffect, useState } from 'react';
import Button from 'Global/Button/Button';
import RoleBadge from 'Global/RoleBadge/RoleBadge';
import './DockerPageSection.css';
import { fetchWithAuth } from 'Global/utils/Auth';
import PageIconModal from 'Pages/Modules/Components/ModuleIconModal/ModuleIconModal';
import { getModulesDomain } from '../../../../../utils/modules';
import { getModulePageMode, pageModeToFlags } from '../../../../../utils/modulePageMode';
import ModulePageRolesModal from '../../ModulePageRolesModal/ModulePageRolesModal';
import ModuleContainerModal from '../ModuleContainerModal/ModuleContainerModal';
import ModuleVisibilityModal from '../ModuleVisibilityModal/ModuleVisibilityModal';

export default function ModulePageSection({ moduleId }) {
  const [pages, setPages] = useState([]);            // holds both existing & new rows
  const [edits, setEdits] = useState({});            // keyed by row.id
  const [isSaving, setIsSaving] = useState(false);
  const [iconTarget, setIconTarget] = useState(null);
  const [rolesTarget, setRolesTarget] = useState(null);
  const [containerTarget, setContainerTarget] = useState(null);
  const [visibilityTarget, setVisibilityTarget] = useState(null);
  const [networks, setNetworks] = useState([]);
  const [containers, setContainers] = useState([]);
  const modulesDomain = getModulesDomain();
  const hasUnsaved = Object.values(edits).some(e => e.dirty);

  const slugify = (value) => String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');

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

  const fetchContainers = async () => {
    try {
      const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/docker/ls`);
      if (!res) return;
      const data = await res.json();
      setContainers(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to fetch module containers:', err);
      setContainers([]);
    }
  };

  useEffect(() => {
    fetchContainers();
  }, [moduleId]);

  // load existing pages
  const fetchPages = async () => {
    try {
      const res = await fetchWithAuth(`
        /api/v1/admin/modules/${moduleId}/pages
      `);
      if (!res) return;
      const data = await res.json();
      const list = (data.pages || []).map(p => ({
        id: p.id,
        slug: p.slug,
        name: p.name,
        targetContainer: p.target_container || '',
        targetPort: typeof p.target_port === 'number' ? p.target_port : null,
        pageMode: getModulePageMode(p),
        needAuth: !!p.need_auth,
        isVisible: p.is_visible !== false,
        icon_url: p.icon_url,
        network: p.network_name || '',
        roles: Array.isArray(p.roles) ? p.roles : [],
        isNew: false,
        slugAuto: false,
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
    const newRow = {
      id: tempId,
      slug: '',
      name: '',
      targetContainer: '',
      targetPort: null,
      pageMode: 'iframe_only',
      needAuth: true,
      isVisible: true,
      icon_url: '',
      network: '',
      roles: [],
      isNew: true,
      slugAuto: true,
    };

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

  // save either POST (new) or PATCH (existing)
  const handleSave = async (id) => {
    const { name, slug, targetContainer, targetPort, pageMode, needAuth, isVisible, isNew } = edits[id];
    if (!name || !slug) return;
    const { iframeOnly, pageOnly } = pageModeToFlags(pageMode);

    setIsSaving(true);
    try {
      const trimmedContainer = (targetContainer || '').trim();
      const hasTarget = trimmedContainer !== '' && typeof targetPort === 'number';
      const payload = {
        name,
        slug,
        target_container: hasTarget ? trimmedContainer : null,
        target_port: hasTarget ? targetPort : null,
        iframe_only: !!iframeOnly,
        page_only: !!pageOnly,
        need_auth: !!needAuth,
        is_visible: !!isVisible,
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

  const handleContainerSave = (id, next) => {
    setEdits((prev) => {
      const current = prev[id] || {};
      return {
        ...prev,
        [id]: {
          ...current,
          targetContainer: next.targetContainer || '',
          targetPort: typeof next.targetPort === 'number' ? next.targetPort : null,
          network: next.network || '',
          dirty: true,
        },
      };
    });
  };

  const handleVisibilitySave = (id, next) => {
    setEdits((prev) => {
      const current = prev[id] || {};
      return {
        ...prev,
        [id]: {
          ...current,
          pageMode: next.pageMode || 'both',
          isVisible: next.isVisible !== false,
          needAuth: !!next.needAuth,
          dirty: true,
        },
      };
    });
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
        <div className="page-table-wrap">
        <table className="page-table">
          <thead>
            <tr>
              <th className="icon-col">Icon</th>
              <th>Nom</th>
              <th>Slug</th>
              <th className="container-col">Container</th>
              <th className="visibility-col">Visibility</th>
              <th className="roles-col">Role</th>
              <th className="actions-col">Actions</th>
            </tr>
          </thead>
          <tbody>
          {pages.map(({ id }) => {
            const edit = edits[id] || {};
            const missingContainer = !(edit.targetContainer || '').trim();
            const missingPort = typeof edit.targetPort !== 'number';
            const missingName = !(edit.name || '').trim();
            const missingSlug = !(edit.slug || '').trim();
            const portProtocol = edit.targetPort
              ? (containers
                .find((container) => container.name === edit.targetContainer)
                ?.ports
                ?.find((port) => port.container_port === edit.targetPort)
                ?.protocol || 'tcp')
              : '';
            const containerLabel = missingContainer
              ? 'Pas de conteneur'
              : edit.targetPort
                ? `${edit.targetContainer}:${edit.targetPort}/${portProtocol}`
                : edit.targetContainer;
            const containerHasWarning = !missingContainer && (missingPort || !(edit.network || '').trim());
            const visibilitySummary = [
              edit.pageMode === 'iframe_only' ? 'Iframe only' : edit.pageMode === 'page_only' ? 'Page only' : 'Both',
              edit.isVisible !== false ? 'Sidebar on' : 'Sidebar off',
              edit.needAuth ? 'Login on' : 'Login off',
            ].join(' • ');
            return (
              <tr
                key={id}
                className={`page-item${edit.dirty ? ' dirty' : ''}`}
              >
                <td className="page-cell page-icon-cell">
                  <img src={edit.icon_url || '/icons/modules.png'} className="page-icon-preview" alt="icon" title="Click to change icon" onClick={()=> setIconTarget(id)} />
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
                  <button
                    type="button"
                    className={`page-container-button${containerHasWarning ? ' has-warning' : ''}`}
                    onClick={() => setContainerTarget(edit)}
                    title="Configure container, port and network"
                  >
                    {missingContainer ? (
                      <span className="page-container-empty">Pas de conteneur</span>
                    ) : (
                      <span className="page-container-summary">{containerLabel}</span>
                    )}
                    {containerHasWarning && (
                      <span className="field-warning page-container-warning" title="Container, port or network missing">!</span>
                    )}
                  </button>
                </td>
                <td className="page-cell">
                  <button
                    type="button"
                    className="page-visibility-button"
                    onClick={() => setVisibilityTarget(edit)}
                    title="Configure mode, sidebar and page access"
                  >
                    <span className="page-visibility-summary">{visibilitySummary}</span>
                  </button>
                </td>
                <td className="page-cell">
                  <button
                    type="button"
                    className="page-role-cell"
                    onClick={() => setRolesTarget(edit)}
                    title="Open role editor"
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
                      disabled={!edit.dirty || isSaving || missingName || missingSlug}
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
        <PageIconModal
          moduleId={moduleId}
          pageId={iconTarget}
          currentIcon={(edits[iconTarget] && edits[iconTarget].icon_url) || ''}
          onClose={()=> setIconTarget(null)}
          onUpdated={()=> { setIconTarget(null); fetchPages(); }}
        />
      )}
      {rolesTarget && (
        <ModulePageRolesModal
          open={!!rolesTarget}
          moduleId={moduleId}
          page={rolesTarget}
          onClose={() => setRolesTarget(null)}
          onUpdated={fetchPages}
        />
      )}
      {containerTarget && (
        <ModuleContainerModal
          open={!!containerTarget}
          containers={containers}
          networks={networks}
          value={containerTarget}
          onClose={() => setContainerTarget(null)}
          onSave={(next) => {
            handleContainerSave(containerTarget.id, next);
            setContainerTarget(null);
          }}
        />
      )}
      {visibilityTarget && (
        <ModuleVisibilityModal
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
