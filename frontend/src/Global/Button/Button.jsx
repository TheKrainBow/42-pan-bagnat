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
  const isIcon = Boolean(icon) && !label;
  const isIconAndText = Boolean(icon) && label;
  const isSquare = isSingleEmoji(label);

  function isSingleEmoji(str) {
    if (typeof str !== 'string') return false;
    const trimmed = str.trim();
    const segmenter = new Intl.Segmenter(undefined, { granularity: 'grapheme' });
    const graphemes = [...segmenter.segment(trimmed)];
    return graphemes.length === 1 && /\p{Extended_Pictographic}/u.test(graphemes[0].segment);
  }

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
      className={`custom-btn ${color} ${isIconAndText ? 'icon-txt-btn' : ''} ${isIcon ? 'icon-btn' : ''} ${isSquare ? 'square' : ''}${disabled ? 'disabled' : ''} ${shake ? 'shake' : ''} ${highlight ? 'highlighted' : ''} ${attention ? 'attention' : ''} ${ripple ? 'ripple' : ''}`}
      onClick={handleClick}
      aria-disabled={disabled}
      tabIndex={disabled ? -1 : 0}
      {...props}
    >
      {icon && <img src={icon} alt={label || ''} className="btn-icon-image" />}
      {label && <span className="btn-label">{label}</span>}
    </button>
  </div>
);
});

export default Button;
