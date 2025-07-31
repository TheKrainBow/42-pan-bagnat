import React, { useState, forwardRef, useImperativeHandle } from 'react';
import { toast } from 'react-toastify';

import './Button.css';

const Button = forwardRef(({
    label,
    icon,
    color = 'gray',
    onClick,
    disabled = false,
    onClickDisabled,
    disabledMessage,
    ...props
  }, ref) => {
  const [shake, setShake] = useState(false);
  const [highlight, setHighlight] = useState(false);
  const [attention, setAttention] = useState(false);
  const [ripple, setRipple] = useState(false);
  const isIcon = Boolean(icon);

  useImperativeHandle(ref, () => ({
    animateHighlight() {
      setAttention(true);
      setRipple(false);
      setTimeout(() => {
        setAttention(false);
        setRipple(true);
        setTimeout(() => setRipple(false), 700); // cleanup ripple
      }, 500); // after grow+fall is done
    }
  }));

  const handleClick = async (e) => {
  if (disabled) {
    setShake(true);
    setTimeout(() => setShake(false), 300);
    if (onClickDisabled) {
      onClickDisabled(e);
    } else if (disabledMessage) {
      toast.error(disabledMessage);
    }
    return;
  }

  try {
    await onClick?.(e);
  } catch (err) {
    console.error(err);
  }
};

return (
  <div className="button-wrapper">
    <button
      type="button"
      className={`custom-btn ${color} ${isIcon ? 'icon-btn' : ''} ${disabled ? 'disabled' : ''} ${shake ? 'shake' : ''} ${attention ? 'attention' : ''} ${ripple ? 'ripple' : ''}`}
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
  </div>
);
});

export default Button;
