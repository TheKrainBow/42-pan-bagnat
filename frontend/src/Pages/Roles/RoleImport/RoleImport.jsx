// src/components/RoleImport.jsx
import React, { useState, useEffect, useRef } from 'react';
import { Wheel } from '@uiw/react-color';
import AppIcon from 'ui/molecules/AppIcon/AppIcon';
import Field from 'ui/molecules/Field/Field';
import './RoleImport.css';
import { fetchWithAuth } from 'Global/utils/Auth';

export default function RoleImport({ show, onClose, onCreateSuccess }) {
  // --- form state ---
  const [name, setName] = useState('');
  const [color, setColor] = useState('#7c3aed');
  const [isDefault, setIsDefault] = useState(false);
  const [modulesList, setModulesList] = useState([]);
  const [selectedModules, setSelectedModules] = useState([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // --- refs for Field validation & shake ---
  const nameRef = useRef();
  const colorRef = useRef();

  // reset all fields
  const resetForm = () => {
    setName('');
    setColor('#7c3aed');
    setIsDefault(false);
    setSelectedModules([]);
    setError('');
    setLoading(false);
  };
  const handleCloseModal = () => {
    resetForm();
    onClose();
  };

  // fetch modules when modal opens
  useEffect(() => {
    if (!show) return;
    fetchWithAuth('/api/v1/admin/modules?limit=1000')
      .then(res => res.json())
      .then(body => setModulesList(Array.isArray(body.modules) ? body.modules : []))
      .catch(err => console.error('Failed to load modules', err));
  }, [show]);

  const toggleModule = id => {
    setSelectedModules(prev =>
      prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]
    );
  };

  if (!show) return null;

  const handleCreate = async () => {
    setError('');
    // run final validation on fields
    const validName = nameRef.current.isValid(true);
    const validColor = colorRef.current.isValid(true);
    if (!validName || !validColor) {
      // shake invalid fields
      if (!validName) nameRef.current.triggerShake();
      if (!validColor) colorRef.current.triggerShake();
      return;
    }

    setLoading(true);
    try {
      const res = await fetchWithAuth('/api/v1/admin/roles', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name.trim(),
          color,
          is_default: isDefault,
          modules: selectedModules,
        }),
      });
      if (!res.ok) throw new Error(await res.text());
      onCreateSuccess();
      handleCloseModal();
    } catch (err) {
      setError(err.message || 'Failed to create role');
      setLoading(false);
    }
  };

  return (
    <div
      className="modal-backdrop"
      onClick={e => e.target === e.currentTarget && handleCloseModal()}
    >
      <div className="modal">
        <h3 className="modal-title">Create a new role</h3>

        <Field
          ref={nameRef}
          label="Group Name"
          value={name}
          onChange={e => setName(e.target.value)}
          placeholder="e.g. IT"
          required
        />

        <div className="form-row checkbox-row">
          <label className="checkbox">
            <input
              type="checkbox"
              checked={isDefault}
              onChange={e => setIsDefault(e.target.checked)}
            />
            is set by default*
          </label>
        </div>

        <div className="form-row">
          <Field
            ref={colorRef}
            label="Color"
            value={color}
            onChange={e => setColor(e.target.value)}
            placeholder="#7c3aed"
            required
            validator={val =>
              /^#[0-9A-Fa-f]{6}$/.test(val)
                ? null
                : 'Invalid hex color (e.g. #1f2937).'
            }
          />
          <div className="color-picker-row">
            <Wheel
              color={color}
              onChange={c => setColor(c.hex?.toLowerCase?.() || color)}
              width={180}
              height={180}
            />
            <div className="color-side">
              <div
                className="color-preview"
                style={{ background: color }}
              />
            </div>
          </div>
        </div>

        <div className="form-row">
          <label>Applications*</label>
          <div className="modules-picker">
            {modulesList.map(mod => (
              <div
                key={mod.id}
                className={
                  'module-item' +
                  (selectedModules.includes(mod.id) ? ' selected' : '')
                }
                title={mod.name}
                onClick={() => toggleModule(mod.id)}
              >
                <AppIcon app={mod} fallback="/icons/modules.png" />
              </div>
            ))}
          </div>
        </div>

        {error && <div className="form-error">{error}</div>}

        <div className="modal-actions">
          <button
            className="btn"
            onClick={handleCloseModal}
            disabled={loading}
          >
            Cancel
          </button>
          <button
            className="btn btn-primary"
            onClick={handleCreate}
            disabled={loading}
          >
            {loading ? 'Creating…' : 'Create'}
          </button>
        </div>
      </div>
    </div>
  );
}
