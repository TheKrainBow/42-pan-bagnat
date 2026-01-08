import React, { useEffect, useRef, useState } from 'react';
import './ModuleImport.css';
import Button from 'Global/Button/Button';
import Field from 'Global/Field/Field';
import { useTour, TourAnchor, dataAnchorProps } from 'Global/Tour/TourProvider';
import { useNavigate } from 'react-router-dom';
import { fetchWithAuth } from 'Global/utils/Auth';

const ModuleImport = ({ onClose }) => {
  const gitInputRef = useRef();
  const gitBranchInputRef = useRef();
  const nameInputRef = useRef();
  const importBtnRef = useRef();
  const navigate = useNavigate();

  const [moduleName, setModuleName] = useState('Hello World');
  const [gitUrl, setGitUrl] = useState('https://github.com/pan-bagnat/hello-world.git');
  const [gitBranch, setGitBranch] = useState('');
  const [sshKeys, setSshKeys] = useState([]);
  const [selectedSSHKey, setSelectedSSHKey] = useState('');
  const [keysError, setKeysError] = useState('');
  const [keysLoading, setKeysLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const sshUrlRegex = /^git@[^\s]+:[^\s]+\.git$/;
  const httpUrlRegex = /^https?:\/\/[^\s/]+\/[^\s/]+\/[^\s/]+\.git$/;

  const validate = () => {
    const isGitValid = gitInputRef.current.isValid(true);
    const isNameValid = nameInputRef.current.isValid(true);
    if (!isGitValid) gitInputRef.current.triggerShake();
    if (!isNameValid) nameInputRef.current.triggerShake();
    return isGitValid && isNameValid;
  };

  const loadSSHKeys = async () => {
    setKeysLoading(true);
    setKeysError('');
    try {
      const res = await fetchWithAuth('/api/v1/admin/ssh-keys');
      if (!res.ok) throw new Error('Failed to fetch SSH keys');
      const data = await res.json();
      setSshKeys(Array.isArray(data?.ssh_keys) ? data.ssh_keys : []);
    } catch (err) {
      setKeysError(err.message || 'Unable to load SSH keys');
    } finally {
      setKeysLoading(false);
    }
  };

  useEffect(() => {
    loadSSHKeys();
  }, []);

  const handleSubmit = async () => {
    if (!validate()) return;

    setIsSubmitting(true);
    try {
      const payload = { name: moduleName, git_url: gitUrl };
      const trimmed = gitBranch.trim();
      if (trimmed !== '') payload.git_branch = trimmed;
      if (selectedSSHKey) payload.ssh_key_id = selectedSSHKey;
      const res = await fetchWithAuth('/api/v1/admin/modules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      if (!res.ok) throw new Error('Failed to import module');
      const data = await res.json();
      navigate(`${data.id}`);
    } catch (err) {
      alert(err.message || 'An error occurred.');
    } finally {
      setIsSubmitting(false);
    }
  };

  // ---------------- Tutorial logic ----------------
  const tour = useTour();

  const startTutorial = () => tour?.start('module-import');

  return (
    <div className="mi-overlay">
      <div className="mi-content">
        <div className="mi-header-row">
          <h3>Import Module</h3>
          <Button color="gray" label="I need help!" onClick={startTutorial} />
        </div>

        <TourAnchor id="moduleImport.name"><div {...dataAnchorProps('moduleImport.name')}>
        <Field
          label="Module Name"
          ref={nameInputRef}
          value={moduleName}
          onChange={e => setModuleName(e.target.value)}
          placeholder="my-awesome-module"
          required={true}
          validator={value => !value ? 'Module name is required.' : null}
        />
        </div></TourAnchor>

        <TourAnchor id="moduleImport.git"><div {...dataAnchorProps('moduleImport.git')}>
        <Field
          label="Git Repository URL (SSH)"
          ref={gitInputRef}
          value={gitUrl}
          onChange={e => setGitUrl(e.target.value)}
          placeholder="git@github.com:org/repo.git"
          required={true}
          validator={value => (sshUrlRegex.test(value) || httpUrlRegex.test(value)) ? null : 'Must be a valid URL.'}
        />
        </div></TourAnchor>

        <TourAnchor id="moduleImport.branch"><div {...dataAnchorProps('moduleImport.branch')}>
        <Field
          label="Git Branch Name"
          ref={gitBranchInputRef}
          value={gitBranch}
          onChange={e => setGitBranch(e.target.value)}
          placeholder="main"
          required={false}
        />
        </div></TourAnchor>
        <div className="mi-field">
          <label>SSH Key</label>
          <select
            className="mi-select"
            value={selectedSSHKey}
            onChange={e => setSelectedSSHKey(e.target.value)}
            disabled={keysLoading}
          >
            <option value="">
              Generate a new key
            </option>
            {sshKeys.map(key => (
              <option key={key.id} value={key.id}>
                {key.name}
              </option>
            ))}
          </select>
          <p className="mi-select-hint">
            {keysError
              ? <span className="mi-error-text">{keysError}</span>
              : 'Pick an existing key or let Pan Bagnat generate a new one for you.'}
          </p>
        </div>

        <div className="mi-actions">
          <Button label="Cancel" color="gray" onClick={onClose} />
          <TourAnchor id="moduleImport.submit"><div {...dataAnchorProps('moduleImport.submit')}>
          <Button
            label={isSubmitting ? 'Importing...' : 'Import Module'}
            color="blue"
            onClick={handleSubmit}
            disabled={isSubmitting}
            ref={importBtnRef}
          />
          </div></TourAnchor>
        </div>
      </div>

      {/* App-level TourOverlay is rendered by TourProvider */}
    </div>
  );
};

export default ModuleImport;
