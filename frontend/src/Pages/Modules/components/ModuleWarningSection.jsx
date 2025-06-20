import { useState } from "react";
import './ModuleWarningSection.css';

const ModuleWarningSection = ({ sshKey }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(sshKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
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
      </div>
    </div>
  );
}

export default ModuleWarningSection;
