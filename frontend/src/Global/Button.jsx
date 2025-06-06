import React from 'react';
import './Button.css';

const Button = ({ label, icon, color = 'gray', onClick, ...props }) => {
  const isIcon = Boolean(icon);

  return (
    <button
      className={`custom-btn ${color} ${isIcon ? 'icon-btn' : ''}`}
      onClick={onClick}
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
