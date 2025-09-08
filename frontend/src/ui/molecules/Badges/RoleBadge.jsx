import React from 'react';
import './RoleBadge.css';

export default function RoleBadge({ role, children, onClick, onDelete }) {
  return (
    <span
      className="role-badge"
      style={{ backgroundColor: role?.color || '#888' }}
      onClick={onClick}
    >
      {children}
      {onDelete && (
        <button className="role-badge-delete" onClick={(e) => { e.stopPropagation(); onDelete(); }} aria-label="Remove role">×</button>
      )}
    </span>
  );
}
