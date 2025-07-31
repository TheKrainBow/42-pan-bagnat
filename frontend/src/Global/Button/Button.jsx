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
    callToAction() {
      setRipple(false);
      setAttention(false);
      setHighlight(false);
      setAttention(true);
      setTimeout(() => {
        setAttention(false);
        setRipple(true);
        setTimeout(() => {
          setRipple(false);
          setHighlight(true);
          setTimeout(() => {
            setHighlight(false);
          }, 1500)
        }, 700); // cleanup ripple
      }, 500); // after grow+fall is done
    },
    triggerShake() {
      setShake(true);
      setTimeout(() => setShake(false), 300);
    },
  }));

  const handleClick = async (e) => {
  if (disabled) {
    if (onClickDisabled) {
      onClickDisabled(e);
    } else {
      setShake(true);
      setTimeout(() => setShake(false), 300);
    }
    if (disabledMessage) {
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
      className={`custom-btn ${color} ${isIcon ? 'icon-btn' : ''} ${disabled ? 'disabled' : ''} ${shake ? 'shake' : ''} ${highlight ? 'highlighted' : ''} ${attention ? 'attention' : ''} ${ripple ? 'ripple' : ''}`}
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
