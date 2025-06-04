// components/ModuleSettings.jsx
import React from 'react';
import RoleBadge from '../../../Global/RoleBadge';
import Button from '../../../Global/Button';
import './ModuleSettings.css';

const ModuleSettings = ({ roles, status, onToggleStatus, statusUpdating }) => {
  const handleFetchData = () => {
    console.log('Fetching data...');
  };
  const handleUpdate = () => {
    console.log('Handling update...');
  };
  const handleUninstall = () => {
    console.log('Handling uninstall...');
  };
  return (
    <div className="module-settings-container">
      <div className="module-settings-actions">
        <Button label="Fetch Data" color="gray" onClick={handleFetchData} />
        <Button label="Update" color="blue" onClick={handleUpdate} />
        <Button label="Uninstall" color="red" onClick={handleUninstall} />
      </div>

      <div className="module-settings-toggle">
        <strong>Enabled:</strong>
        <label className="switch-label">
          <label className="switch">
            <input
              type="checkbox"
              checked={status === 'enabled'}
              onChange={onToggleStatus}
              disabled={statusUpdating}
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
              <RoleBadge key={role.id} hexColor={role.color}>{role.name}</RoleBadge>
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
