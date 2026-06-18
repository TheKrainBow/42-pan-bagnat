import React, { useEffect, useState } from 'react';
import Button from 'Global/Button/Button';
import './ModuleVisibilityModal.css';

export default function ModuleVisibilityModal({ open, value, onClose, onSave }) {
  const [pageMode, setPageMode] = useState(value?.pageMode || 'both');
  const [isVisible, setIsVisible] = useState(value?.isVisible !== false);
  const [needAuth, setNeedAuth] = useState(!!value?.needAuth);

  useEffect(() => {
    if (!open) return;
    setPageMode(value?.pageMode || 'both');
    setIsVisible(value?.isVisible !== false);
    setNeedAuth(!!value?.needAuth);
  }, [open, value]);

  if (!open) return null;

  const handleSave = () => {
    onSave?.({
      pageMode,
      isVisible,
      needAuth,
    });
  };

  return (
    <div className="modal-backdrop" onMouseDown={onClose}>
      <div className="modal module-visibility-modal" onMouseDown={(e) => e.stopPropagation()}>
        <div className="modal-title">Visibility</div>
        <div className="module-visibility-summary">
          Controls the display mode, sidebar visibility and login requirement.
        </div>

        <div className="module-visibility-grid">
          <label className="module-visibility-field">
            <span>Mode</span>
            <select value={pageMode} onChange={(e) => setPageMode(e.target.value)}>
              <option value="iframe_only">Iframe only</option>
              <option value="both">Iframe and Page</option>
              <option value="page_only">Page only</option>
            </select>
          </label>

          <label className="module-visibility-check">
            <input
              type="checkbox"
              checked={isVisible}
              onChange={(e) => setIsVisible(e.target.checked)}
            />
            <span>Visible in sidebar</span>
          </label>

          <label className="module-visibility-check">
            <input
              type="checkbox"
              checked={needAuth}
              onChange={(e) => setNeedAuth(e.target.checked)}
            />
            <span>Require login for direct access</span>
          </label>
        </div>

        <div className="modal-actions">
          <Button label="Cancel" color="gray" onClick={onClose} />
          <Button label="Apply" color="green" onClick={handleSave} />
        </div>
      </div>
    </div>
  );
}
