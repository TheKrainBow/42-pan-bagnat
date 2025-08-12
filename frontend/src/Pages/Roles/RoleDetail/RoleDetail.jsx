import React, { useEffect, useState, useRef, useMemo } from "react";
import { useParams } from "react-router-dom";
import { fetchWithAuth } from "Global/utils/Auth";
import Field from "Global/Field/Field";
import Button from "Global/Button/Button";
import ModuleSimpleBadge from 'Global/ModuleSimpleBadge/ModuleSimpleBadge';
import UserBadge from "Global/UserBadge/UserBadge";
import RoleBadge from 'Global/RoleBadge/RoleBadge';
import { Wheel } from '@uiw/react-color';
import { useNavigate } from 'react-router-dom';
import { toast } from 'react-toastify';
import './RoleDetail.css';

export default function RoleDetail() {
  const { roleId } = useParams();
  const [role, setRole] = useState(null);
  const [modules, setModules] = useState([]); // assigned modules
  const [users, setUsers] = useState([]);     // assigned users

  const [allModules, setAllModules] = useState([]);
  const [allUsers, setAllUsers] = useState([]);

  const [showModuleSearch, setShowModuleSearch] = useState(false);
  const [moduleSearchTerm, setModuleSearchTerm] = useState("");
  const moduleDropdownRef = useRef();

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

  // load all modules & users on mount
  useEffect(() => {
    fetchWithAuth('/api/v1/admin/modules?limit=1000')
      .then(res => res.json())
      .then(data => setAllModules(data.modules || []))
      .catch(console.error);
    fetchWithAuth('/api/v1/admin/users?limit=1000')
      .then(res => res.json())
      .then(data => setAllUsers(data.users || []))
      .catch(console.error);
  }, []);

  // close clicks
  useEffect(() => {
    function handleClickOutside(e) {
      if (showModuleSearch && moduleDropdownRef.current && !moduleDropdownRef.current.contains(e.target)) {
        setShowModuleSearch(false);
        setModuleSearchTerm("");
      }
      if (showUserSearch && userDropdownRef.current && !userDropdownRef.current.contains(e.target)) {
        setShowUserSearch(false);
        setUserSearchTerm("");
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [showModuleSearch, showUserSearch]);

  // filtered lists
  const filteredModules = useMemo(() => allModules.filter(m =>
    m.name.toLowerCase().includes(moduleSearchTerm.toLowerCase()) &&
    !modules.some(assigned => assigned.id === m.id)
  ), [allModules, moduleSearchTerm, modules]);

  const filteredUsers = useMemo(() => allUsers.filter(u =>
    u.ft_login.toLowerCase().includes(userSearchTerm.toLowerCase()) &&
    !users.some(assigned => assigned.id === u.id)
  ), [allUsers, userSearchTerm, users]);

  // add/remove handlers
  const handleAddModule = async mod => {
    try {
      const res = await fetchWithAuth(`/api/v1/admin/modules/${mod.id}/roles/${roleId}`, { method: 'POST' });
      if (!res.ok) throw new Error();
      setModules(prev => {
        const updated = [...prev, mod];
        // Recalculate filteredRoles manually
        const remaining = allModules.filter(m =>
            m.name.toLowerCase().includes(moduleSearchTerm.toLowerCase()) &&
            !updated.some(assigned => assigned.id === m.id)
        );

        if (remaining.length === 0) {
          setShowModuleSearch(false);
          setModuleSearchTerm("");
        }

        return updated;
      });
    } catch (err) { console.error(err); }
  };

  const handleRemoveModule = async mod => {
    try {
      const res = await fetchWithAuth(`/api/v1/admin/modules/${mod.id}/roles/${roleId}`, { method: 'DELETE' });
      if (!res.ok) throw new Error();
      setModules(prev => prev.filter(m => m.id !== mod.id));
    } catch (err) { console.error(err); }
  };
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
            Edit {" "}
            <RoleBadge role={{ id: role.id, color: color }}>
              {name}
            </RoleBadge>
          </h2>
          <div className="button-wrapper">
              <Button
                label={'🔧 Advanced rules'}
                color="gray"
                onClick={() => navigate(`/admin/roles/${roleId}/rule-builder`)}
              />
            <Button
              label={loading ? 'Saving…' : 'Save Changes'}
              color="blue"
              onClick={handleSave}
              disabled={false}
              disabledMessage={"Not implemented (sorry)"}
            />

            <Button
              label={'🗑️ Delete Role'}
              color="red"
              onClick={handleDelete}
              disabled={false}
              disabledMessage={"Not implemented (sorry)"}
            />
          </div>
        </div>
        <div className="role-form">
          <Field ref={nameRef} label="Name" value={name} onChange={e=>setName(e.target.value)} required />
          <Field
            ref={colorRef}
            label="Color"
            value={color}
            backgroundColor={color}
            onChange={e => setColor(e.target.value)}
            required
            validator={val => /^#[0-9A-Fa-f]{6}$/.test(val) ? null : 'Invalid hex'}
          />
          <div className="color-wheel">
            <Wheel color={color} onChange={c=>setColor(c.hex?.toLowerCase())} width={120} height={120} />
          </div>
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={isDefault}
              onChange={e => setIsDefault(e.target.checked)}
            />
            Assign this role to new users
          </label>
        </div>
      </section>

      {error && <div className="form-error">{error}</div>}
      <div className="actions">
      </div>
      
      <section className="assign-section">
        <div className="section-header">
          <label>Modules</label>
          <Button
            label={"+ Add Module"}
            color="blue"
            onClick={() => setShowModuleSearch(true)}
          />
        </div>
        <div className="assigned-list modules-list">
          {modules.length === 0 ? (
            <div className="empty-message">No modules assigned</div>
          ) : (
            modules.map(m => (
              <ModuleSimpleBadge
                key={m.id}
                module={m}
                onClick={() => navigate(`/admin/modules/${m.id}?tab=settings`)}
                onDelete={() => handleRemoveModule(m)}
              />
            ))
          )}
        </div>

        {showModuleSearch && (
          <div className="role-search-dropdown" ref={moduleDropdownRef}>
            <input
              type="text"
              placeholder="Search modules..."
              value={moduleSearchTerm}
              onChange={e => setModuleSearchTerm(e.target.value)}
            />
            <ul className="role-search-list">
              {filteredModules.length === 0 ? (
                <li className="role-line no-result">No more modules available</li>
              ) : (
                filteredModules.map(m => (
                  <li key={m.id}>
                    <div
                      className="role-line role-line-clickable"
                      onClick={() => handleAddModule(m)}
                    >
                      <ModuleSimpleBadge module={m} />
                    </div>
                  </li>
                ))
              )}
            </ul>
          </div>
        )}
      </section>


      <section className="assign-section">
        <div className="section-header">
          <label>Users</label>
          <Button
            label={"+ Add User"}
            color="blue"
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