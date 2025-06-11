import React, { useState, useEffect, useImperativeHandle, forwardRef } from 'react';
import './Field.css';

const Field = forwardRef(({
  label,
  value,
  onChange,
  placeholder = '',
  required = false,
  validator,
  type = 'text',
  multiline = false,
  rows = 3,
  alwaysShowError = false,
}, ref) => {
  const [errors, setErrors] = useState([]);
  const [isTouched, setIsTouched] = useState(alwaysShowError);
  const [shake, setShake] = useState(false);

  // Normalize validator output to an array of strings
  const runValidation = (final = false) => {
    if (!isTouched && !final) return [];

    if (required && value.trim() === '') {
      return ['This field is required.'];
    }

    const result = validator?.(value);
    if (!result) return [];
    return Array.isArray(result) ? result : [result];
  };

  useEffect(() => {
    setErrors(runValidation());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value, isTouched]);

  // Shake on demand
  const triggerShake = () => {
    setShake(true);
    setTimeout(() => setShake(false), 300);
  };

  const handleBlur = () => {
    setIsTouched(true);
  };

  // Parent can call .triggerShake() via ref
  useImperativeHandle(ref, () => ({
    triggerShake,
    isValid: (final) => {
        const errs = runValidation(final);
        setErrors(errs);
        setIsTouched(true);
        return errs.length === 0;
    },
    }));

  const classNames = [
    'field-input',
    errors.length > 0 ? 'invalid' : '',
    shake ? 'shake' : '',
  ].join(' ').trim();

  return (
    <div className="field-wrapper">
      <label className="field-label">
        {label}
        {required && <span className="field-asterisk">*</span>}
      </label>

      {multiline ? (
        <textarea
          className={classNames}
          rows={rows}
          placeholder={placeholder}
          value={value}
          onChange={onChange}
          onBlur={handleBlur}
        />
      ) : (
        <input
          className={classNames}
          type={type}
          placeholder={placeholder}
          value={value}
          onChange={onChange}
          onBlur={handleBlur}
        />
      )}

      {errors.length === 1 ? (
        <p className="field-error">{errors[0]}</p>
        ) : errors.length > 1 ? (
        <ul className="field-error-list">
            {errors.map((msg, i) => (
            <li key={i} className="field-error">{msg}</li>
            ))}
        </ul>
        ) : null}
    </div>
  );
});

export default Field;
