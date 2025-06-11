// ModuleImport.jsx
import React, { useState, useEffect, useRef } from 'react';
import './ModuleImport.css';
import Button from 'Global/Button';
import Field from 'Global/Field';

const ModuleImport = ({ onClose, onSubmit }) => {
  const gitInputRef = useRef();
  const sshInputRef = useRef();

  const [gitUrl, setGitUrl] = useState('ssh://git@someexample/yes.git');
  const [sshKey, setSshKey] = useState('ssh-rsa a');

  // 1) Only accept SSH-style Git URLs: ssh://[user@]host[:port]/path/to/repo.git
  const sshUrlRegex = /^ssh:\/\/(?:[^\s@]+@)?[^\s/:]+(?::\d+)?\/[^\s]+\.git$/;

  // Helper: basic SSH key regex (ssh-rsa, ssh-ed25519, etc.)
  // This checks that it starts with "ssh-(rsa|ed25519|ecdsa|dss)" followed by a space and a base64 blob.
  // It does NOT check the contents of the key, only the general shape.
  const sshRegex = /^(ssh-(rsa|ed25519|ecdsa|dss)) [A-Za-z0-9+/]+=*( [^\s]+)?$/;
  
  const handleSubmit = () => {
    const isGitValid = gitInputRef.current.isValid(true);
    const isSshValid = sshInputRef.current.isValid(true);

    if (!isGitValid) gitInputRef.current.triggerShake();
    if (!isSshValid) sshInputRef.current.triggerShake();

    if (!isGitValid || !isSshValid) return;

    // All good: submit and reset
    onSubmit({ gitUrl, sshKey });
  };

  const GitURLValidator = (value) => {
    if (!sshUrlRegex.test(value)) {
      return 'Please enter a valid SSH-style Git URL (e.g. ssh://git@host:port/path.git).';
    }
    return null;
  }

  const SSHKeyValidator = (value) => {
    if (!sshRegex.test(value)) {
      return ['SSH key must start with "ssh-rsa", "ssh-ed25519", etc. and be valid base64.'];
    }
    return null;
  }

  return (
    <div className="mi-overlay">
      <div className="mi-content">
        <h3>Import Module</h3>

        {/* Git URL Field */}
        <Field
          label="Git SSH URL:"
          ref={gitInputRef}
          value={gitUrl}
          onChange={e => setGitUrl(e.target.value)}
          placeholder="ssh://git@host:port/namespace/repo.git"
          required={true}
          validator={GitURLValidator}
        />

        {/* SSH Key Field */}
        <Field
          label="SSH Key:"
          ref={sshInputRef}
          value={sshKey}
          onChange={e => setSshKey(e.target.value)}
          placeholder="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ…"
          required={true}
          validator={SSHKeyValidator}
          multiline={true}
          rows={4}
        />

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
            // disabled={!(sshUrlRegex.test(gitUrl.trim()) && sshKeyRegex.test(sshKey.trim()))}
          />
        </div>
      </div>
    </div>
  );
};

export default ModuleImport;
