import React, { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { fetchWithAuth } from "Global/utils/Auth";
import Field from "Global/Field/Field";
import Button from "Global/Button/Button";
import ModuleSimpleBadge from 'Global/ModuleSimpleBadge/ModuleSimpleBadge';
import UserBadge from "Global/UserBadge/UserBadge";
import RoleBadge from 'Global/RoleBadge/RoleBadge';
import { Wheel } from '@uiw/react-color';
import { toast } from 'react-toastify';
import './RoleDetail.css';

export default function RoleDetail() {
  const { roleId } = useParams();
  const [role, setRole] = useState(null);
  const [modules, setModules] = useState([]); // modules impacted by at least one page
  const [users, setUsers] = useState([]);     // assigned users

  const [allUsers, setAllUsers] = useState([]);

  const [showUserSearch, setShowUserSearch] = useState(false);
  const [userSearchTerm, setUserSearchTerm] = useState("");
  const userDropdownRef = useRef();

  // form fields
  const [name, setName] = useState("");
  const [color, setColor] = useState('#7c3aed');
  const [isDefault, setIsDefault] = useState(false);
  const nameRef = useRef();
  const colorRef = useRef();

  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const navigate = useNavigate();

  // load role details
  useEffect(() => {
    async function load() {
      try {
        const res = await fetchWithAuth(`/api/v1/admin/roles/${roleId}`);
        const data = await res.json();
        setRole(data);
        setName(data.name);
        setColor(data.color);
        setIsDefault(data.is_default);
        setModules(data.modules || []);
        setUsers(data.users || []);
      } catch (err) {
        console.error(err);
      }
    }
    load();
  }, [roleId]);

  const PROTECTED_ROLE_IDS = useMemo(
    () => new Set(['roles_admin', 'roles_blacklist', 'roles_default']),
    []
  );
  const isProtectedRole = role ? PROTECTED_ROLE_IDS.has(role.id) : false;

  // Only allow editing name/color/advanced rules when NOT assigned by default
  const canEditBasics = !isDefault; // false when the checkbox is ticked

  // load users on mount
  useEffect(() => {
    fetchWithAuth('/api/v1/admin/users?limit=1000')
      .then(res => res.json())
      .then(data => setAllUsers(data.users || []))
      .catch(console.error);
  }, []);

  useEffect(() => {
    function handleClickOutside(e) {
      if (showUserSearch && userDropdownRef.current && !userDropdownRef.current.contains(e.target)) {
        setShowUserSearch(false);
        setUserSearchTerm("");
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [showUserSearch]);

  const filteredUsers = useMemo(() => allUsers.filter(u =>
    u.ft_login.toLowerCase().includes(userSearchTerm.toLowerCase()) &&
    !users.some(assigned => assigned.id === u.id)
  ), [allUsers, userSearchTerm, users]);

  const handleAddUser = async u => {
    try {
      const res = await fetchWithAuth(`/api/v1/admin/users/${u.id}/roles/${roleId}`, { method: 'POST' });
      if (!res.ok) throw new Error();
      setUsers(prev => {
        const updated = [...prev, u];
        // Recalculate filteredRoles manually
        const remaining = allUsers.filter(u =>
            u.ft_login.toLowerCase().includes(userSearchTerm.toLowerCase()) &&
            !updated.some(assigned => assigned.id === u.id)
        );

        if (remaining.length === 0) {
          setShowUserSearch(false);
          setUserSearchTerm("");
        }

        return updated;
      });
    } catch (err) { console.error(err); }
  };
  const handleRemoveUser = async u => {
    try {
      const res = await fetchWithAuth(`/api/v1/admin/users/${u.id}/roles/${roleId}`, { method: 'DELETE' });
      if (!res.ok) throw new Error();
      setUsers(prev => prev.filter(x => x.id !== u.id));
    } catch (err) { console.error(err); }
  };

  // save updates
  const handleSave = async () => {
    setError('');
    const validName = nameRef.current.isValid(true);
    const validColor = colorRef.current.isValid(true);
    if (!validName || !validColor) {
      if (!validName) nameRef.current.triggerShake();
      if (!validColor) colorRef.current.triggerShake();
      return;
    }
    setLoading(true);
    try {
      const res = await fetchWithAuth(`/api/v1/admin/roles/${roleId}`, {
        method: 'PATCH',
        headers: {'Content-Type':'application/json'},
        body: JSON.stringify({ name: name.trim(), color: color, is_default: isDefault })
      });
      if (!res.ok) throw new Error(await res.text());
      // modules & users already updated via individual calls
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
      toast.success("Role saved")
    }
  };


  const handleDelete = async () => {
    setError('');
    setLoading(true);
    try {
      const res = await fetchWithAuth(`/api/v1/admin/roles/${roleId}`, {
        method: 'DELETE',
        headers: {'Content-Type':'application/json'},
      });
      if (!res.ok) throw new Error(await res.text());
    } catch (err) {
      setError(err.message);
      console.log(err.message);
    } finally {
      setLoading(false);
      toast(`🗑️ Role ${name} was deleted`, {
        className: 'toast-simple',
        autoClose: 10000,
        onClick: () => {
          if (window.location.pathname === `/admin/roles/${roleId}`) {
            window.location.href = '/admin/roles';
          }
        },
        onClose: () => {
          if (window.location.pathname === `/admin/roles/${roleId}`) {
            window.location.href = '/admin/roles';
          }
        }
      });
    }
  };

  if (!role) return <div>Loading...</div>;
  return (
    <div className="role-detail-wrapper">
      <section className="assign-section">
        <div className="section-header">
          <h2 className="role-title">
            Edit{" "}
            <RoleBadge role={{ id: role.id, color: color }}>
              {name}
            </RoleBadge>
          </h2>
          <div className="button-wrapper">
            <Button
              label={'🔧 Advanced rules'}
              color="gray"
              disabled={!canEditBasics || loading}
              disabledMessage={"Disable 'Assign this role to new users' to edit rules"}
              onClick={() => navigate(`/admin/roles/${roleId}/rule-builder`)}
            />
            <Button
              label={'Save Changes'}
              color="blue"
              disabled={loading}
              onClick={handleSave}
            />
            <Button
              label={'🗑️ Delete Role'}
              color="red"
              disabled={isProtectedRole || loading}
              onClick={handleDelete}
              disabledMessage={"This system role cannot be deleted"}
            />
          </div>
        </div>

        <div className="role-form">
          <Field
            ref={nameRef}
            label="Name"
            value={name}
            onChange={e => canEditBasics && setName(e.target.value)}
            required
            disabled={!canEditBasics}
          />

          <Field
            ref={colorRef}
            label="Color"
            value={color}
            backgroundColor={color}
            onChange={e => canEditBasics && setColor(e.target.value)}
            required
            disabled={!canEditBasics}
            validator={val => /^#[0-9A-Fa-f]{6}$/.test(val) ? null : 'Invalid hex'}
          />

          <div
            className="color-wheel"
            style={{ opacity: canEditBasics ? 1 : 0.5, pointerEvents: canEditBasics ? 'auto' : 'none' }}
          >
            <Wheel
              color={color}
              onChange={c => setColor(c.hex?.toLowerCase())}
              width={120}
              height={120}
            />
          </div>

          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={isDefault}
              onChange={e => !isProtectedRole && setIsDefault(e.target.checked)}
              disabled={isProtectedRole}
              title={isProtectedRole ? "This system role's assignment is locked" : undefined}
            />
            Assign this role to new users
          </label>
        </div>
      </section>

      {error && <div className="form-error">{error}</div>}

      {/* Modules */}
      <section className="assign-section">
        <div className="section-header">
          <label>Modules</label>
        </div>
        <p className="section-help">
          These modules are derived from the pages that grant this role.
        </p>
        <div className="assigned-list modules-list">
          {modules.length === 0 ? (
            <div className="empty-message">No modules assigned</div>
          ) : (
            modules.map(m => (
              <ModuleSimpleBadge
                key={m.id}
                module={m}
                onClick={() => navigate(`/admin/modules/${m.id}?tab=settings`)}
              />
            ))
          )}
        </div>
      </section>

      {/* Users */}
      <section className="assign-section">
        <div className="section-header">
          <label>Users</label>
          <Button
            label={"+ Add User"}
            color="blue"
            disabled={loading}
            onClick={() => setShowUserSearch(true)}
          />
        </div>

        <div className="assigned-list users-list">
          {users.length === 0 ? (
            <div className="empty-message">No users assigned</div>
          ) : (
            users.map(u => (
              <div key={u.id} className="assigned-item">
                <UserBadge user={u} onClick={() => navigate(`/admin/users/${u.id}`)} onDelete={() => handleRemoveUser(u)} />
              </div>
            ))
          )}
        </div>

        {showUserSearch && (
          <div className="role-search-dropdown" ref={userDropdownRef}>
            <input
              type="text"
              placeholder="Search users..."
              value={userSearchTerm}
              onChange={e => setUserSearchTerm(e.target.value)}
            />
            <ul className="role-search-list">
              {filteredUsers.length === 0 ? (
                <li className="role-line no-result">No more users available</li>
              ) : (
                filteredUsers.map(u => (
                  <li key={u.id}>
                    <div
                      className="role-line role-line-clickable"
                      onClick={() => handleAddUser(u)}
                    >
                      <UserBadge user={u} disableClick={true} />
                    </div>
                  </li>
                ))
              )}
            </ul>
          </div>
        )}
      </section>
    </div>
  );
}
