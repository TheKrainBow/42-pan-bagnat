import React from 'react';
import AppIcon from 'Global/AppIcon/AppIcon';
import './ModuleSimpleBadge.css';

export default function ModuleSimpleBadge({ module, onClick, onDelete, href }) {
  const withHover = onClick != null || href != null;
  const Tag = href ? 'a' : 'div';
  return (
    <Tag
      className={`module-simple-badge${withHover ? ' with-hover' : ''}`}
      onClick={onClick}
      href={href}
      tabIndex={0}
      role="button"
    >
      <AppIcon
        app={{ icon_url: module.icon_url, name: module.name }}
        fallback="/icons/modules.png"
        className="module-simple-icon"
      />
      <span className="module-simple-name">{module.name}</span>
      {onDelete && (
        <button
          className="module-simple-badge-delete"
          onClick={e => {
            e.stopPropagation();
            onDelete();
          }}
          aria-label={`Remove ${module.name}`}
        >
          ×
        </button>
      )}
    </Tag>
  );
}
