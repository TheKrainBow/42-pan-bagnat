import React from 'react';
import './Button.css';

const Button = ({ label, color = 'gray', onClick, ...props }) => {
  return (
    <button
      className={`custom-btn ${color}`}
      onClick={onClick}
      {...props}
    >
      {label}
    </button>
  );
};

export default Button;
