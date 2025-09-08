import React from 'react';
import './ModuleBadge.css';
import AppIcon from '../../molecules/AppIcon/AppIcon';

export default function ModuleBadge({ mod, onClick }) {
  return (
    <div className="module-badge" onClick={onClick} role="button" tabIndex={0}>
      <AppIcon app={{ icon_url: mod.icon_url, name: mod.name }} fallback="/icons/modules.png" />
      <div className="module-info">
        <div className="module-name">{mod.name}</div>
        <div className={`module-status ${mod.status}`}>{mod.status}</div>
      </div>
    </div>
  );
}
