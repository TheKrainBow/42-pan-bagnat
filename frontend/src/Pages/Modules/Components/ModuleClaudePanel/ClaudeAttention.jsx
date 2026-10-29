import React from 'react';
import { claudeAttentionReason } from './claudeAttentionStatus';
import './ClaudeAttention.css';

const ClaudeAttentionDot = ({ session, title, className = '' }) => (
  <span
    className={`claude-attention-dot${session?.status === 'waiting_input' ? ' waiting' : ''} ${className}`}
    title={title || claudeAttentionReason(session)}
    aria-label={title || claudeAttentionReason(session)}
  />
);

export default ClaudeAttentionDot;
