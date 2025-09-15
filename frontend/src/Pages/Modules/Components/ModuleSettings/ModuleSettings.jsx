// src/components/ModuleSettings.jsx
import React, { useEffect, useState, useRef } from "react";
import RoleBadge from 'Global/RoleBadge/RoleBadge';
import Button from 'Global/Button/Button';
import { fetchWithAuth } from "Global/utils/Auth";
import ModuleAboutSection from '../ModuleAboutSection/ModuleAboutSection';
import { useNavigate } from 'react-router-dom'
import './ModuleSettings.css';

export default function ModuleSettings({
  module,
  statusUpdating,
  onToggleStatus,
  onFetchData,
  onUpdate,
  onUninstall
}) {
  // Local state for the roles assigned to this module
  const [moduleRoles, setModuleRoles] = useState(module.roles || []);

  // All possible roles for the picker
  const [availableRoles, setAvailableRoles] = useState([]);

  // Search/dropdown state
  const [showRoleSearch, setShowRoleSearch] = useState(false);
  const [searchRoleTerm, setSearchRoleTerm] = useState("");
  const dropdownRef = useRef();
  const navigate = useNavigate();

  // Load available roles once (or whenever module changes)
  useEffect(() => {
    fetchWithAuth(`/api/v1/admin/roles`)
      .then(res => res.json())
      .then(data => setAvailableRoles(data.roles || []))
      .catch(console.error);
  }, [module.id]);

  // Close dropdown if clicking outside
  useEffect(() => {
    function onOutside(e) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target)) {
        setShowRoleSearch(false);
        setSearchRoleTerm("");
      }
    }
    if (showRoleSearch) document.addEventListener("mousedown", onOutside);
    else document.removeEventListener("mousedown", onOutside);
    return () => document.removeEventListener("mousedown", onOutside);
  }, [showRoleSearch]);

  // Filter out already-assigned roles and apply search
  const filteredRoles = availableRoles.filter(r =>
    r.name.toLowerCase().includes(searchRoleTerm.toLowerCase()) &&
    !moduleRoles.some(mr => mr.id === r.id)
  );

  // Add a role to this module
  const handleAddRole = async role => {
    try {
      const res = await fetchWithAuth(
        `/api/v1/admin/modules/${module.id}/roles/${role.id}`,
        { method: "POST" }
      );
      if (!res.ok) throw new Error("Failed to add role");
      // Update local state
      setModuleRoles(prev => {
        const updated = [...prev, role];

        // Recalculate filteredRoles manually
        const remaining = availableRoles.filter(r =>
          r.name.toLowerCase().includes(searchRoleTerm.toLowerCase()) &&
          !updated.some(mr => mr.id === r.id)
        );

        if (remaining.length === 0) {
          setShowRoleSearch(false);
          setSearchRoleTerm("");
        }

        return updated;
      });
    } catch (err) {
      console.error(err);
    }
  };

  // Remove a role from this module
  const handleRoleRemove = async role => {
    try {
      const res = await fetchWithAuth(
        `/api/v1/admin/modules/${module.id}/roles/${role.id}`,
        { method: "DELETE" }
      );
      if (!res.ok) throw new Error("Failed to remove role");
      setModuleRoles(prev => prev.filter(r => r.id !== role.id));
    } catch (err) {
      console.error(err);
    }
  };

  return (
    <div className="module-settings-container">
      <ModuleAboutSection module={module} />

      <div className="module-settings-actions">
        <Button
          label="🗑️ Delete Module"
          color="red"
          onClick={onUninstall}
        />
      </div>

      <div className="module-settings-toggle">
        <strong>Enabled:</strong>
        <label className="switch-label">
          <label className={`switch ${module.status === 'waiting_for_action' ? 'waiting' : ''}`}>
            <input
              type="checkbox"
              checked={module.status === 'enabled'}
              onChange={onToggleStatus}
              disabled={statusUpdating || module.status === 'waiting_for_action'}
            />
            <span className="slider round"></span>
          </label>
        </label>
      </div>

      <div className="user-modules-panel">
        <div className="role-panel-header">
          <label>Roles:</label>
          <button
            className="add-role-btn"
            onClick={() => setShowRoleSearch(show => !show)}
          >
            + Add Role
          </button>
        </div>

        <div className="user-role-tags">
          {moduleRoles.length === 0
            ? <i>No roles assigned</i>
            : moduleRoles.map(role => (
                <RoleBadge
                  key={role.id}
                  role={role}
                  onClick={() => navigate(`/admin/roles/${role.id}`)}
                  onDelete={() => handleRoleRemove(role)}
                >
                  {role.name}
                </RoleBadge>
              ))
          }
        </div>

        {showRoleSearch && (
          <div className="role-search-dropdown" ref={dropdownRef}>
            <input
              type="text"
              placeholder="Search roles..."
              value={searchRoleTerm}
              onChange={e => setSearchRoleTerm(e.target.value)}
              autoFocus
            />
            <ul className="role-search-list">
              {filteredRoles.length === 0 ? (
                <li className="role-line no-result">No more roles available.</li>
              ) : (
                filteredRoles.map(role => (
                  <li key={role.id}>
                    <div
                      className="role-line role-line-clickable"
                      onClick={() => handleAddRole(role)}
                    >
                      <RoleBadge role={role}>
                        {role.name}
                      </RoleBadge>
                    </div>
                  </li>
                ))
              )}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}
