// src/components/ModuleSettings.jsx
import React, { useEffect, useState } from "react";
import Button from 'Global/Button/Button';
import { fetchWithAuth } from "Global/utils/Auth";
import ModuleAboutSection from '../ModuleAboutSection/ModuleAboutSection';
import { toast } from 'react-toastify';
import './ModuleSettings.css';

export default function ModuleSettings({
  module,
  statusUpdating,
  onToggleStatus,
  onUpdate,
  onUninstall
}) {

  const [sshKeys, setSshKeys] = useState([]);
  const [sshKeysLoading, setSshKeysLoading] = useState(true);

  useEffect(() => {
    const loadSSHKeys = async () => {
      setSshKeysLoading(true);
      try {
        const res = await fetchWithAuth('/api/v1/admin/ssh-keys');
        if (!res.ok) throw new Error('Failed to fetch SSH keys');
        const data = await res.json();
        setSshKeys(Array.isArray(data?.ssh_keys) ? data.ssh_keys : []);
      } catch (err) {
        console.error(err);
        setSshKeys([]);
      } finally {
        setSshKeysLoading(false);
      }
    };
    loadSSHKeys();
  }, [module.id]);

  const handleSSHKeyChange = async (sshKeyId) => {
    if (!sshKeyId || sshKeyId === module.ssh_key_id) return;
    try {
      const res = await fetchWithAuth(`/api/v1/admin/modules/${module.id}/git/ssh-key`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ssh_key_id: sshKeyId })
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to update SSH key');
      }
      toast.success('SSH key updated');
      onUpdate?.();
    } catch (err) {
      toast.error(err.message || 'Unable to update SSH key');
    }
  };

  return (
    <div className="module-settings-container">
      <ModuleAboutSection
        module={module}
        sshKeys={sshKeys}
        sshKeysLoading={sshKeysLoading}
        onSSHKeyChange={handleSSHKeyChange}
      />

      {/* Icon editor moved to header modal (click icon) */}

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

    </div>
  );
}
