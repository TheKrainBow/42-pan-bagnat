import React from 'react';
import './ClaudeRepoNotice.css';

const formatTime = (iso) => (iso ? new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '');

// Shown above the IDE and Shell tabs while a Claude session is open on the
// module: Claude edits the same working tree from its own container.
const ClaudeRepoNotice = ({ occupant, isMine, onOpen }) => {
  const working = occupant.status === 'working' || occupant.status === 'waiting_input';
  const who = isMine ? 'Your Claude session' : `Claude (${occupant.user_login})`;
  const what = working ? 'is modifying this repository right now' : 'is open on this repository';
  return (
    <div className={`claude-repo-notice${working ? ' working' : ''}`} role="status">
      <span>
        <strong>{who}</strong> {what} (since {formatTime(occupant.since)}).
        {' '}Edits made here may conflict with Claude&apos;s.
      </span>
      {isMine && (
        <button type="button" className="claude-repo-notice-link" onClick={onOpen}>Open Claude</button>
      )}
    </div>
  );
};

export default ClaudeRepoNotice;
