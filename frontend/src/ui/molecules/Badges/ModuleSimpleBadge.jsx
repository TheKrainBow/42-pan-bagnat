import React from 'react';
import './ModuleSimpleBadge.css';

export default function ModuleSimpleBadge({ module, onClick, onDelete }) {
  return (
    <div className="module-simple-badge" onClick={onClick} role="button" tabIndex={0}>
      <span className="msb-name">{module.name}</span>
      {onDelete && (
        <button className="msb-delete" onClick={(e) => { e.stopPropagation(); onDelete(); }} aria-label="Remove module">×</button>
      )}
    </div>
  );
}
