// ModuleImport.jsx
import React, { useState, useEffect } from 'react';
import './ModuleImport.css';
import Button from 'Global/Button';

const ModuleImport = ({ onClose, onSubmit }) => {
  const [gitUrl, setGitUrl] = useState('');
  const [sshKey, setSshKey] = useState('');

  // Track validation state:
  const [gitUrlError, setGitUrlError] = useState('');
  const [sshKeyError, setSshKeyError] = useState('');

  // Shake
  const [shakeGit, setShakeGit] = useState(false);
  const [shakeSsh, setShakeSsh] = useState(false);

  // 1) Only accept SSH-style Git URLs: ssh://[user@]host[:port]/path/to/repo.git
  const sshUrlRegex = /^ssh:\/\/(?:[^\s@]+@)?[^\s/:]+(?::\d+)?\/[^\s]+\.git$/;

  // Helper: basic SSH key regex (ssh-rsa, ssh-ed25519, etc.)
  // This checks that it starts with "ssh-(rsa|ed25519|ecdsa|dss)" followed by a space and a base64 blob.
  // It does NOT check the contents of the key, only the general shape.
  const sshRegex = /^(ssh-(rsa|ed25519|ecdsa|dss)) [A-Za-z0-9+/]+=*( [^\s]+)?$/;

  // Helper: validate a URL string by attempting new URL(...)
  useEffect(() => {
    if (gitUrl === '') {
      setGitUrlError('');
    } else if (!sshUrlRegex.test(gitUrl.trim())) {
      setGitUrlError('Please enter a valid SSH-style Git URL (e.g. ssh://git@host:port/path.git).');
    } else {
      setGitUrlError('');
    }
  }, [gitUrl]);

  // Validate SSH Key whenever it changes
  useEffect(() => {
    if (sshKey === '') {
      setSshKeyError('');
    } else if (!sshRegex.test(sshKey.trim())) {
      setSshKeyError('SSH key must start with "ssh-rsa", "ssh-ed25519", etc. and be valid base64.');
    } else {
      setSshKeyError('');
    }
  }, [sshKey]);

  // Only enable “Submit” if both fields have no error AND are non‐empty
  const isFormValid = () => {
    return (
      gitUrl !== '' &&
      sshKey !== '' &&
      gitUrlError === '' &&
      sshKeyError === ''
    );
  };

  
  const handleSubmit = () => {
    const trimmedGit = gitUrl.trim();
    const trimmedSsh = sshKey.trim();
    let didShake = false;

    // If Git URL is empty or has an error, shake that field
    if (trimmedGit === '' || gitUrlError) {
      setShakeGit(true);
      setTimeout(() => setShakeGit(false), 300);
      didShake = true;
    }
    // If SSH Key is empty or has an error, shake that field
    if (trimmedSsh === '' || sshKeyError) {
      setShakeSsh(true);
      setTimeout(() => setShakeSsh(false), 300);
      didShake = true;
    }

    if (didShake) {
      return;
    }
    // All good: submit and reset
    onSubmit({ gitUrl: trimmedGit, sshKey: trimmedSsh });
    setGitUrl('');
    setSshKey('');
  };

  return (
    <div className="mi-overlay">
      <div className="mi-content">
        <h3>Import Module</h3>

        {/* Git URL Field */}
        <div className="mi-field">
          <label htmlFor="mi-git-url">Git SSH URL:</label>
          <input
            id="mi-git-url"
            type="text"
            placeholder="ssh://git@host:port/path.git"
            value={gitUrl}
            onChange={(e) => setGitUrl(e.target.value)}
            className={`
              mi-input
              ${gitUrlError   ? 'invalid' : ''}
              ${shakeGit     ? 'shake'   : ''}
            `.trim()}
          />
          {gitUrlError && <p className="mi-error-text">{gitUrlError}</p>}
        </div>

        {/* SSH Key Field */}
        <div className="mi-field">
          <label htmlFor="mi-ssh-key">SSH Key:</label>
          <textarea
            id="mi-ssh-key"
            placeholder="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."
            value={sshKey}
            onChange={(e) => setSshKey(e.target.value)}
            rows={4}
            className={`mi-textarea ${sshKeyError ? 'invalid' : ''} ${
              shakeSsh ? 'shake' : ''
            }`.trim()}
          />
          {sshKeyError && <p className="mi-error-text">{sshKeyError}</p>}
        </div>

        {/* Actions */}
        <div className="mi-actions">
          <Button
            label="Cancel"
            color="gray"
            onClick={onClose}
          />
          <Button
            label="Submit"
            color="blue"
            onClick={handleSubmit}
            disabled={!isFormValid()}
          />
        </div>
      </div>
    </div>
  );
};

export default ModuleImport;
