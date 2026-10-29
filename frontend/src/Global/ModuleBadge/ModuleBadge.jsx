import React from 'react';
import { Link } from 'react-router-dom';
import AppIcon from 'Global/AppIcon/AppIcon';
import ModuleStatusBadge from 'Pages/Modules/Components/ModuleStatusBadge/ModuleStatusBadge';
import ClaudeAttentionDot from 'Pages/Modules/Components/ModuleClaudePanel/ClaudeAttention';
import './ModuleBadge.css';

const CLAUDE_LABELS = {
  starting: 'Claude starting',
  idle: 'Claude idle',
  working: 'Claude working',
  waiting_input: 'Claude waiting',
};

export default function ModuleBadge({ mod, claude, claudeAttention }) {
  return (
    <Link
      to={`/admin/modules/${mod.id}?tab=${claudeAttention ? 'claude' : 'settings'}`}
      className={`module-card ${mod.status}`}
    >
      {claudeAttention && <ClaudeAttentionDot session={claudeAttention} className="module-card-claude-dot" />}
      <div className="module-icon">
        <AppIcon app={{ icon_url: mod.icon_url, name: mod.name }} fallback="/icons/modules.png" />
        {mod.status === 'disabled' && (
          <span className="module-disabled-overlay">Disabled</span>
        )}
      </div>
      <div className="module-content">
        <div className="module-title-row">
          <strong>{mod.name}</strong>
          <ModuleStatusBadge status={mod.status} />
        </div>
        {claude && (
          <p className={`module-claude module-claude-${claude.status}`} title={`Claude session of ${claude.user_login}`}>
            <span className="module-claude-dot" /> {CLAUDE_LABELS[claude.status] || 'Claude'} · {claude.user_login}
          </p>
        )}
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
