// RoleBadge.jsx
import React from 'react';
import { getReadableStyles } from 'Global/utils/ColorUtils';
import './RoleBadge.css';

const RoleBadge = ({ role, children, onClick, onDelete, href }) => {
  const styles = getReadableStyles(role.color);
  const withHover = onClick != null;

  const handleDeleteClick = e => {
    e.stopPropagation();
    onDelete?.();
  };

  const Tag = href ? 'a' : 'span';

  return (
    <Tag
      className={`role-badge${withHover || href ? ' with-hover' : ''}`}
      style={styles}
      onClick={onClick}
      href={href}
      role={href || onClick ? 'button' : undefined}
      tabIndex={href || onClick ? 0 : undefined}
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
    </Tag>
  );
};

export default RoleBadge;
