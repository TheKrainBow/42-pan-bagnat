import React, { useEffect, useState } from 'react';
import Button from 'Global/Button/Button';
import RoleBadge from 'Global/RoleBadge/RoleBadge';
import './DockerPageSection.css';
import { fetchWithAuth } from 'Global/utils/Auth';
import PageIconModal from 'Pages/Modules/Components/ModuleIconModal/ModuleIconModal';
import { getModulesDomain } from '../../../../../utils/modules';
import { getModulePageMode, pageModeToFlags } from '../../../../../utils/modulePageMode';
import ModulePageRolesModal from '../../ModulePageRolesModal/ModulePageRolesModal';

export default function ModulePageSection({ moduleId }) {
  const [pages, setPages] = useState([]);            // holds both existing & new rows
  const [edits, setEdits] = useState({});            // keyed by row.id
  const [isSaving, setIsSaving] = useState(false);
  const [iconTarget, setIconTarget] = useState(null);
  const [rolesTarget, setRolesTarget] = useState(null);
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

  const getContainerPorts = (containerName) => {
    if (!containerName) return [];
    const container = containers.find((c) => c.name === containerName);
    if (!container || !Array.isArray(container.ports)) return [];
    const normalizeProto = (value) => {
      const proto = String(value || '').trim().toLowerCase();
      return proto || 'tcp';
    };
    const unique = new Map();
    for (const port of container.ports) {
      if (!Number.isInteger(port?.container_port) || port.container_port <= 0) continue;
      const normalized = {
        ...port,
        protocol: normalizeProto(port.protocol),
      };
      const key = `${normalized.container_port}/${normalized.protocol}`;
      if (!unique.has(key)) {
        unique.set(key, normalized);
      }
    }
    return Array.from(unique.values());
  };

  const formatPortLabel = (port) => {
    if (!port) return '';
    const proto = port.protocol ? `/${port.protocol}` : '';
    if (port.scope === 'host' && port.host_port) {
      return `${port.container_port}${proto} • host ${port.host_port}`;
    }
    return `${port.container_port}${proto} • network`;
  };

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

  const handleContainerSelect = (id, containerName) => {
    const availablePorts = getContainerPorts(containerName);
    setEdits((prev) => {
      const existing = prev[id] || {};
      const keepCurrent =
        availablePorts.find((p) => p.container_port === existing.targetPort)?.container_port ??
        null;
      const fallbackPort =
        keepCurrent !== null ? keepCurrent : (availablePorts[0]?.container_port ?? null);
      return {
        ...prev,
        [id]: {
          ...existing,
          targetContainer: containerName,
          targetPort: containerName ? fallbackPort : null,
          dirty: true,
        },
      };
    });
  };

  const handlePortSelect = (id, value) => {
    const parsed = value === '' ? null : Number(value);
    setEdits(prev => ({
      ...prev,
      [id]: {
        ...prev[id],
        targetPort: Number.isNaN(parsed) ? null : parsed,
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
              <th>Container</th>
              <th>Port</th>
              <th>Network</th>
              <th>Mode</th>
              <th>Auth</th>
              <th className="roles-col">Roles</th>
              <th>Visible</th>
              <th className="actions-col">Actions</th>
            </tr>
          </thead>
          <tbody>
          {pages.map(({ id }) => {
            const edit = edits[id] || {};
            const containerPorts = getContainerPorts(edit.targetContainer);
            const hasKnownContainer = !!containers.find(c => c.name === edit.targetContainer);
            const missingContainer = !(edit.targetContainer || '').trim();
            const missingPort = typeof edit.targetPort !== 'number';
            const missingNetwork = !(edit.network || '').trim();
            const missingName = !(edit.name || '').trim();
            const missingSlug = !(edit.slug || '').trim();
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
                  <div className="select-with-warning">
                    <select
                      className="page-select"
                      value={edit.targetContainer || ''}
                      onChange={e => handleContainerSelect(id, e.target.value)}
                    >
                      <option value="">Select container</option>
                      {containers.map(container => (
                        <option key={container.name} value={container.name}>
                          {container.name}
                        </option>
                      ))}
                      {edit.targetContainer && !hasKnownContainer && (
                        <option value={edit.targetContainer} className="missing-option">
                          {edit.targetContainer} (!)
                        </option>
                      )}
                    </select>
                    {missingContainer && (
                      <span className="field-warning" title="Container not set">!</span>
                    )}
                  </div>
                </td>
                <td className="page-cell">
                  <div className="select-with-warning">
                    <select
                      className="page-select"
                      value={typeof edit.targetPort === 'number' ? String(edit.targetPort) : ''}
                      onChange={e => handlePortSelect(id, e.target.value)}
                      disabled={!edit.targetContainer || containerPorts.length === 0}
                    >
                      <option value="">
                        {!edit.targetContainer
                          ? 'Pick a container first'
                          : containerPorts.length === 0
                            ? 'No ports detected'
                            : 'Select a port'}
                      </option>
                      {containerPorts.map((port, idx) => (
                        <option
                          key={`${port.container_port}-${port.host_port || 0}-${port.protocol || 'tcp'}-${idx}`}
                          value={port.container_port}
                        >
                          {formatPortLabel(port)}
                        </option>
                      ))}
                    </select>
                    {missingPort && (
                      <span className="field-warning" title="Port not set">!</span>
                    )}
                  </div>
                </td>
                <td className="page-cell">
                  <div className="select-with-warning">
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
                    {missingNetwork && (
                      <span className="field-warning" title="Network not set">!</span>
                    )}
                  </div>
                </td>
                <td className="page-cell page-flag-cell">
                  <select
                    className="page-select"
                    value={edit.pageMode || 'both'}
                    onChange={e => handleChange(id, 'pageMode', e.target.value)}
                  >
                    <option value="iframe_only">Iframe only</option>
                    <option value="both">Both</option>
                    <option value="page_only">Page only</option>
                  </select>
                </td>
                <td className="page-cell page-flag-cell">
                  <label className="page-access-toggle">
                    <input
                      type="checkbox"
                      checked={edit.needAuth || false}
                      onChange={e => handleChange(id, 'needAuth', e.target.checked)}
                    />
                  </label>
                </td>
                <td className="page-cell">
                  <div className="page-roles-cell">
                    <div className="page-roles-preview">
                      {(edit.roles || []).length === 0 ? (
                        <span className="page-roles-empty">Public</span>
                      ) : (
                        (edit.roles || []).slice(0, 3).map((role) => (
                          <RoleBadge key={role.id} role={role}>
                            {role.name}
                          </RoleBadge>
                        ))
                      )}
                    </div>
                    <Button
                      label="Set roles"
                      color="gray"
                      onClick={() => setRolesTarget(edit)}
                    />
                  </div>
                </td>
                <td className="page-cell page-flag-cell">
                  <label className="page-access-toggle">
                    <input
                      type="checkbox"
                      checked={edit.isVisible !== false}
                      onChange={e => handleChange(id, 'isVisible', e.target.checked)}
                    />
                  </label>
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
    </div>
  );
}
