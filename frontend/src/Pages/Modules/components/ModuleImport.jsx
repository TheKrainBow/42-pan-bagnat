import React, { useState, useRef } from 'react';
import './ModuleImport.css';
import Button from 'Global/Button';
import Field from 'Global/Field';
import { useNavigate } from 'react-router-dom';

const ModuleImport = ({ onClose }) => {
  const gitInputRef = useRef();
  const gitBranchInputRef = useRef();
  const nameInputRef = useRef();
  const navigate = useNavigate();

  const [moduleName, setModuleName] = useState('Hello World');
  const [gitUrl, setGitUrl] = useState('git@github.com:pan-bagnat/hello-world.git');
  const [gitBranch, setGitBranch] = useState('');
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

  const handleSubmit = async () => {
    if (!validate()) return;

    setIsSubmitting(true);
    const finalBranch = gitBranch.trim() === '' ? 'main' : gitBranch;
    try {
      const res = await fetch('http://localhost:8080/api/v1/modules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: moduleName, git_url: gitUrl, git_branch: finalBranch }),
      });

      if (!res.ok) throw new Error('Failed to import module');
      const data = await res.json();
      navigate(`/modules/${data.id}`);
    } catch (err) {
      alert(err.message || 'An error occurred.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="mi-overlay">
      <div className="mi-content">
        <h3>Import Module</h3>

        <Field
          label="Module Name"
          ref={nameInputRef}
          value={moduleName}
          onChange={e => setModuleName(e.target.value)}
          placeholder="my-awesome-module"
          required={true}
          validator={value => !value ? 'Module name is required.' : null}
        />

        <Field
          label="Git Repository URL (SSH)"
          ref={gitInputRef}
          value={gitUrl}
          onChange={e => setGitUrl(e.target.value)}
          placeholder="git@github.com:org/repo.git"
          required={true}
          validator={value => (sshUrlRegex.test(value) || httpUrlRegex.test(value)) ? null : 'Must be a valid URL.'}
        />

        <Field
          label="Git Branch Name"
          ref={gitInputRef}
          value={gitBranch}
          onChange={e => setGitBranch(e.target.value)}
          placeholder="main"
          required={false}
        />

        <div className="mi-actions">
          <Button label="Cancel" color="gray" onClick={onClose} />
          <Button
            label={isSubmitting ? 'Importing...' : 'Import Module'}
            color="blue"
            onClick={handleSubmit}
            disabled={isSubmitting}
          />
        </div>
      </div>
    </div>
  );
};

export default ModuleImport;
