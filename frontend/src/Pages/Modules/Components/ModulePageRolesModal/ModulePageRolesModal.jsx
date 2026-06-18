import React, { useEffect, useMemo, useState } from 'react';
import Button from 'Global/Button/Button';
import RoleBadge from 'Global/RoleBadge/RoleBadge';
import { fetchWithAuth } from 'Global/utils/Auth';
import './ModulePageRolesModal.css';

export default function ModulePageRolesModal({ open, moduleId, page, onClose, onUpdated }) {
  const [availableRoles, setAvailableRoles] = useState([]);
  const [pageRoles, setPageRoles] = useState(page?.roles || []);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setPageRoles(page?.roles || []);
    setSearch('');
  }, [page]);

  useEffect(() => {
    if (!open) return;
    fetchWithAuth('/api/v1/admin/roles?limit=1000')
      .then(res => res.json())
      .then(data => setAvailableRoles(Array.isArray(data.roles) ? data.roles : []))
      .catch(err => {
        console.error('Failed to load roles', err);
        setAvailableRoles([]);
      });
  }, [open]);

  const filteredRoles = useMemo(() => {
    const term = search.trim().toLowerCase();
    return availableRoles.filter((role) => {
      if (pageRoles.some((assigned) => assigned.id === role.id)) return false;
      if (!term) return true;
      return role.name.toLowerCase().includes(term);
    });
  }, [availableRoles, pageRoles, search]);

  if (!open || !page) return null;

  const mutateRole = async (role, method) => {
    setLoading(true);
    try {
      const res = await fetchWithAuth(
        `/api/v1/admin/modules/${moduleId}/pages/${page.id}/roles/${role.id}`,
        { method }
      );
      if (!res.ok) {
        throw new Error(await res.text());
      }
      setPageRoles((prev) => {
        if (method === 'POST') return [...prev, role];
        return prev.filter((item) => item.id !== role.id);
      });
      onUpdated?.();
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const removeRole = (role) => mutateRole(role, 'DELETE');
  const addRole = (role) => mutateRole(role, 'POST');

  return (
    <div className="page-roles-backdrop" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="page-roles-modal">
        <div className="page-roles-header">
          <div>
            <h3>Set roles</h3>
            <p>{page.name}</p>
          </div>
          <Button label="Close" color="gray" onClick={onClose} />
        </div>

        <div className="page-roles-section">
          <label>Allowed roles</label>
          <div className="page-roles-assigned">
            {pageRoles.length === 0 ? (
              <i>No roles assigned</i>
            ) : (
              pageRoles.map((role) => (
                <RoleBadge key={role.id} role={role} onDelete={() => removeRole(role)}>
                  {role.name}
                </RoleBadge>
              ))
            )}
          </div>
        </div>

        <div className="page-roles-section">
          <label>Add role</label>
          <input
            className="page-roles-search"
            type="text"
            placeholder="Search roles..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <div className="page-roles-list">
            {filteredRoles.length === 0 ? (
              <div className="page-roles-empty">No more roles available</div>
            ) : (
              filteredRoles.map((role) => (
                <button
                  key={role.id}
                  className="page-roles-item"
                  onClick={() => addRole(role)}
                  disabled={loading}
                >
                  <RoleBadge role={role}>{role.name}</RoleBadge>
                </button>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
