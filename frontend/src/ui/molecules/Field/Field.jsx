import React, { useImperativeHandle, useState, forwardRef } from 'react';
import './Field.css';

const Field = forwardRef(({ label, value, onChange, required=false, placeholder='', validator }, ref) => {
  const [err, setErr] = useState('');
  useImperativeHandle(ref, () => ({
    isValid(force=false) {
      const e = validator ? validator(value) : null;
      if (force) setErr(e || '');
      return !e;
    },
    triggerShake() {
      // styling already supports :invalid/err; no-op here
    }
  }));
  return (
    <label className={`field ${err ? 'with-error' : ''}`}>
      <span className="field-label">{label}{required && ' *'}</span>
      <input value={value} onChange={onChange} placeholder={placeholder} className="field-input" />
      {err && <span className="field-error">{err}</span>}
    </label>
  );
});

export default Field;
