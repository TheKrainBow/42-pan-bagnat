import React, { useEffect } from 'react';
import './ModuleUninstallModal.css';
import Button from 'Global/Button/Button';

const ModuleUninstallModal = ({ onConfirm, onCancel }) => {
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'Escape') onCancel();
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onCancel]);

  return (
    <div
      className="modal-backdrop"
      onClick={(e) => {
        if (e.target === e.currentTarget) onCancel();
      }}
    >
      <div className="modal">
        <p className="modal-strong">
          <span className="modal-emoji">⚠️</span> Are you sure you want to uninstall this module? <span className="modal-emoji">⚠️</span>
        </p>
        <p>This action is irreversible and will permanently delete:</p>
        <ul className="modal-description-list">
          <li>All containers associated with the module</li>
          <li>The local cloned module repository (including its <code>.git</code> folder)</li>
          <li>All related database entries (module metadata and role associations)</li>
        </ul>
        <p className="modal-note">
          All data linked to this module will be permanently removed.
        </p>
        <div className="modal-buttons">
          <Button label="Cancel" onClick={onCancel} />
          <Button label="Yes, Uninstall" color="red" onClick={onConfirm} />
        </div>
      </div>
    </div>
  );
};

export default ModuleUninstallModal;
