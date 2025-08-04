import React from 'react';
import { Link } from 'react-router-dom';
import AppIcon from 'Global/AppIcon/AppIcon';
import ModuleStatusBadge from 'Pages/Modules/Components/ModuleStatusBadge/ModuleStatusBadge';
import './ModuleBadge.css';

export default function ModuleBadge({ mod }) {
  return (
    <Link to={`/admin/modules/${mod.id}?tab=settings`} className={`module-card ${mod.status}`}>
      <div className="module-icon">
        <AppIcon app={{ icon_url: mod.icon_url, name: mod.name }} fallback="/icons/modules.png" />
      </div>
      <div className="module-content">
        <div className="module-title-row">
          <strong>{mod.name}</strong>
          <ModuleStatusBadge status={mod.status} />
        </div>
        {mod.last_update && new Date(mod.last_update).getFullYear() > 2000 ? (
          <>
            <p className="module-description">
              v{mod.version} • {mod.late_commits} late commits
            </p>
            <p className="module-updated">
              Last update: {new Date(mod.last_update).toLocaleDateString()}
            </p>
          </>
        ) : (
          <p className="module-waiting">Action required</p>
        )}
      </div>
    </Link>
  );
}
