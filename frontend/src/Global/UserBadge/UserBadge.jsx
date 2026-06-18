// UserBadge.jsx
import React from 'react';
import './UserBadge.css';

export default function UserBadge({ user, onClick, onDelete, href }) {
  const withHover = onClick != null || href != null;
  const Tag = href ? 'a' : 'div';
  return (
    <Tag
      className={`user-badge${withHover ? ' with-hover' : ''}`}
      onClick={onClick}
      href={href}
      role="button"
      tabIndex={0}
    >
      <img
        src={user.ft_photo}
        alt={user.ft_login}
        className="user-badge-avatar"
      />
      <span className="user-badge-name">{user.ft_login}</span>
      {onDelete && (
        <button
          className="user-badge-delete"
          onClick={e => {
            e.stopPropagation();
            onDelete();
          }}
          aria-label="Remove user"
        >
          ×
        </button>
      )}
    </Tag>
  );
}
