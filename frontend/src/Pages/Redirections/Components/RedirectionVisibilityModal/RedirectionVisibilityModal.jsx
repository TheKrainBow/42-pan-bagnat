import React, { useEffect, useState } from 'react';
import Button from 'Global/Button/Button';
import './RedirectionVisibilityModal.css';

export default function RedirectionVisibilityModal({ open, value, onClose, onSave }) {
  const [isVisible, setIsVisible] = useState(value?.isVisible !== false);
  const [needAuth, setNeedAuth] = useState(!!value?.needAuth);

  useEffect(() => {
    if (!open) return;
    setIsVisible(value?.isVisible !== false);
    setNeedAuth(!!value?.needAuth);
  }, [open, value]);

  if (!open) return null;

  const handleSave = () => {
    onSave?.({ isVisible, needAuth });
  };

  return (
    <div className="modal-backdrop" onMouseDown={onClose}>
      <div className="modal redirection-visibility-modal" onMouseDown={(e) => e.stopPropagation()}>
        <div className="modal-title">Visibility</div>
        <div className="redirection-visibility-summary">
          Controls sidebar visibility and login requirement. Clicking this redirection always sends the browser straight to its target URL.
        </div>

        <div className="redirection-visibility-grid">
          <label className="redirection-visibility-check">
            <input
              type="checkbox"
              checked={isVisible}
              onChange={(e) => setIsVisible(e.target.checked)}
            />
            <span>Visible in sidebar</span>
          </label>

          <label className="redirection-visibility-check">
            <input
              type="checkbox"
              checked={needAuth}
              onChange={(e) => setNeedAuth(e.target.checked)}
            />
            <span>Require login to see this link</span>
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
