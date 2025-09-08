import { useState, useRef, forwardRef, useImperativeHandle } from "react";
import './ModuleWarningSection.css';
import Button from "ui/atoms/Button/Button";
import { fetchWithAuth } from 'Global/utils/Auth';

const ModuleWarningSection = forwardRef(({ sshKey, moduleID, onRetrySuccess, onRetry }, ref) => {
  const [copied, setCopied] = useState(false);
  const retryButton = useRef();

  useImperativeHandle(ref, () => ({
    callToAction() {
      retryButton.current?.callToAction();
    }
  }));

  const handleCopy = () => {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(sshKey)
        .then(() => {
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        })
        .catch((err) => {
          console.error("Copy failed", err);
        });
    } else {
      console.warn("Clipboard API not supported");
    }
  };

  const handleRetryClone = async () => {
    try {
      const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleID}/git/clone`, {
        method: 'POST'
      });
      if (res.ok && onRetrySuccess) {
        onRetrySuccess();
      } else {
        retryButton.current?.triggerShake();
      }
    } catch (err) {
      retryButton.current?.triggerShake();
    }
  };

  return (
    <div className="module-warning-section">
      <div className="warning-header">
        <div className="warning-icon">⚠️</div>
        <div className="warning-text">
          This module couldn’t be cloned. Please add the following SSH public key to your repository’s deploy keys.
        </div>
      </div>
      <div className="public-key-wrapper">
        <pre className="public-key-display">{sshKey}</pre>
        <div className="copy-container">
          <Button
            label="📋 Copy"
            color="warning"
            onClick={() => handleCopy()}
          />
          {copied && <div className="copy-tooltip">Copied!</div>}
        </div>
        <div className="retry-clone-container">
          <Button
            ref={retryButton}
            label="🔁 Retry Clone"
            color="warning"
            onClick={() => handleRetryClone()}
          />
        </div>
      </div>
    </div>
  );
});

export default ModuleWarningSection;
