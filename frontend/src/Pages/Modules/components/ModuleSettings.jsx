import React from 'react';
import RoleBadge from 'Global/RoleBadge';
import Button from 'Global/Button';
import './ModuleSettings.css';

const ModuleSettings = ({
  roles,
  status,
  statusUpdating,
  onToggleStatus,
  onFetchData,
  onUpdate,
  onUninstall
}) => {
  return (
    <div className="module-settings-container">
      <div className="module-settings-actions">
        <Button
          label="Fetch Data"
          color="gray"
          onClick={onFetchData}
          disabled={status === 'waiting_for_action'}
        />
        <Button
          label="Update"
          color="blue"
          onClick={onUpdate}
          disabled={status === 'waiting_for_action'}
        />
        <Button
          label="Uninstall"
          color="red"
          onClick={onUninstall}
        />
      </div>

      <div className="module-settings-toggle">
        <strong>Enabled:</strong>
        <label className="switch-label">
          <label className={`switch ${status === 'waiting_for_action' ? 'waiting' : ''}`}>
            <input
              type="checkbox"
              checked={status === 'enabled'}
              onChange={onToggleStatus}
              disabled={statusUpdating || status === 'waiting_for_action'}
            />
            <span className="slider round"></span>
          </label>
        </label>
      </div>

      <div className="module-settings-roles">
        <strong>Linked Roles:</strong>
        <div className="role-badges">
          {Array.isArray(roles) && roles.length > 0 ? (
            roles.map(role => (
              <RoleBadge key={role.id} hexColor={role.color}>
                {role.name}
              </RoleBadge>
            ))
          ) : (
            <span style={{ opacity: 0.5 }}>No roles assigned</span>
          )}
        </div>
      </div>
    </div>
  );
};

export default ModuleSettings;
