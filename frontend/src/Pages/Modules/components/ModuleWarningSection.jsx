import { useState } from "react";
import './ModuleWarningSection.css';

const ModuleWarningSection = ({ sshKey, moduleID, onRetrySuccess, onRetry }) => {
  const [copied, setCopied] = useState(false);
  const [retrying, setRetrying] = useState(false);
  const [retrySuccess, setRetrySuccess] = useState(null);
  const [retry, setRetry] = useState(null);

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
    setRetrying(true);
    setRetrySuccess(null);
    try {
      const res = await fetch(`/api/v1/modules/${moduleID}/git/clone`, {
        method: 'POST'
      });
      setRetrySuccess(res.ok);
      setRetry(true);
      if (res.ok && onRetrySuccess) {
        onRetrySuccess();
      }
    } catch (err) {
      setRetrySuccess(false);
      setRetry(true);
    } finally {
      onRetry();
      setRetrying(false);
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
          <button className="copy-button" onClick={handleCopy}>📋 Copy</button>
          {copied && <div className="copy-tooltip">Copied!</div>}
        </div>
        <div className="retry-clone-container">
          <button
            className="copy-button"
            onClick={handleRetryClone}
            disabled={retrying}
          >
            🔁 Retry Clone
          </button>
        </div>
        {retrySuccess === true && <div className="retry-status success">✅ Clone triggered</div>}
        {retrySuccess === false && <div className="retry-status error">❌ Clone failed</div>}
      </div>
    </div>
  );
};

export default ModuleWarningSection;