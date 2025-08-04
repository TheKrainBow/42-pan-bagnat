// RoleBadge.jsx
import React from 'react';
import { getReadableStyles } from 'Global/utils/ColorUtils';
import './RoleBadge.css';

const RoleBadge = ({ role, children, onClick, onDelete }) => {
  const styles = getReadableStyles(role.color);
  const withHover = onClick != null;

  const handleDeleteClick = e => {
    e.stopPropagation();
    onDelete?.();
  };

  return (
    <span
      className={`role-badge${withHover ? ' with-hover' : ''}`}
      style={styles}
      onClick={onClick}
      role="button"
      tabIndex={0}
    >
      {children}
      {onDelete && (
        <button
          className="role-badge-delete"
          onClick={handleDeleteClick}
          aria-label="Remove role"
        >
          ✕
        </button>
      )}
    </span>
  );
};

export default RoleBadge;
