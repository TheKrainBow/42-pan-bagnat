import React, { useState } from 'react';
import './Button.css';

const Button = ({ label, icon, color = 'gray', onClick, disabled = false, ...props }) => {
  const [shake, setShake] = useState(false);
  const isIcon = Boolean(icon);

  const handleClick = async (e) => {
    if (disabled) {
      setShake(true);
      setTimeout(() => setShake(false), 300);
      return;
    }

    try {
      await onClick?.(e);
    } catch (err) {
      console.error(err);
    }
  };

  return (
    <button
      type="button"
      className={`custom-btn ${color} ${isIcon ? 'icon-btn' : ''} ${disabled ? 'disabled' : ''} ${shake ? 'shake' : ''}`}
      onClick={handleClick}
      aria-disabled={disabled}
      tabIndex={disabled ? -1 : 0}
      {...props}
    >
      {isIcon ? (
        <img src={icon} alt={label || ''} className="btn-icon-image" />
      ) : (
        label
      )}
    </button>
  );
};

export default Button;
