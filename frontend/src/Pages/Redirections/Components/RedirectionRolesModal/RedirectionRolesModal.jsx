import React, { useEffect, useMemo, useState } from 'react';
import Button from 'Global/Button/Button';
import RoleBadge from 'Global/RoleBadge/RoleBadge';
import { fetchWithAuth } from 'Global/utils/Auth';
import './RedirectionRolesModal.css';

export default function RedirectionRolesModal({ open, redirection, onClose, onUpdated }) {
  const [availableRoles, setAvailableRoles] = useState([]);
  const [roles, setRoles] = useState(redirection?.roles || []);
  const [forbiddenRoles, setForbiddenRoles] = useState(redirection?.forbiddenRoles || []);
  const [search, setSearch] = useState('');
  const [forbiddenSearch, setForbiddenSearch] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setRoles(redirection?.roles || []);
    setForbiddenRoles(redirection?.forbiddenRoles || []);
    setSearch('');
    setForbiddenSearch('');
  }, [redirection]);

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
      if (roles.some((assigned) => assigned.id === role.id)) return false;
      if (!term) return true;
      return role.name.toLowerCase().includes(term);
    });
  }, [availableRoles, roles, search]);

  const filteredForbiddenRoles = useMemo(() => {
    const term = forbiddenSearch.trim().toLowerCase();
    return availableRoles.filter((role) => {
      if (forbiddenRoles.some((assigned) => assigned.id === role.id)) return false;
      if (!term) return true;
      return role.name.toLowerCase().includes(term);
    });
  }, [availableRoles, forbiddenRoles, forbiddenSearch]);

  if (!open || !redirection) return null;

  const mutateRole = async (role, method) => {
    setLoading(true);
    try {
      const res = await fetchWithAuth(
        `/api/v1/admin/redirections/${redirection.id}/roles/${role.id}`,
        { method }
      );
      if (!res.ok) {
        throw new Error(await res.text());
      }
      setRoles((prev) => {
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

  const mutateForbiddenRole = async (role, method) => {
    setLoading(true);
    try {
      const res = await fetchWithAuth(
        `/api/v1/admin/redirections/${redirection.id}/forbidden-roles/${role.id}`,
        { method }
      );
      if (!res.ok) {
        throw new Error(await res.text());
      }
      setForbiddenRoles((prev) => {
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
  const removeForbiddenRole = (role) => mutateForbiddenRole(role, 'DELETE');
  const addForbiddenRole = (role) => mutateForbiddenRole(role, 'POST');

  return (
    <div className="page-roles-backdrop" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="page-roles-modal">
        <div className="page-roles-header">
          <div>
            <h3>Set roles</h3>
            <p>{redirection.name}</p>
          </div>
          <Button label="Close" color="gray" onClick={onClose} />
        </div>

        <div className="page-roles-group">
          <div className="page-roles-section">
            <label>Allowed roles</label>
            <div className="page-roles-assigned">
              {roles.length === 0 ? (
                <i>No roles assigned</i>
              ) : (
                roles.map((role) => (
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

        <div className="page-roles-group page-roles-group-forbidden">
          <div className="page-roles-section">
            <label>Forbidden roles</label>
            <p className="page-roles-hint">Users with any of these roles are denied access, even if they also have an allowed role.</p>
            <div className="page-roles-assigned">
              {forbiddenRoles.length === 0 ? (
                <i>No roles forbidden</i>
              ) : (
                forbiddenRoles.map((role) => (
                  <RoleBadge key={role.id} role={role} onDelete={() => removeForbiddenRole(role)}>
                    {role.name}
                  </RoleBadge>
                ))
              )}
            </div>
          </div>

          <div className="page-roles-section">
            <label>Add forbidden role</label>
            <input
              className="page-roles-search"
              type="text"
              placeholder="Search roles..."
              value={forbiddenSearch}
              onChange={(e) => setForbiddenSearch(e.target.value)}
            />
            <div className="page-roles-list">
              {filteredForbiddenRoles.length === 0 ? (
                <div className="page-roles-empty">No more roles available</div>
              ) : (
                filteredForbiddenRoles.map((role) => (
                  <button
                    key={role.id}
                    className="page-roles-item"
                    onClick={() => addForbiddenRole(role)}
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
    </div>
  );
}
