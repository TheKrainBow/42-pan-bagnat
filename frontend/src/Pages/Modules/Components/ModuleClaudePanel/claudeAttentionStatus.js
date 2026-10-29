// A session needs its owner when Claude asks something, or when a task ended
// and the owner has not opened the Claude tab since.
export const claudeNeedsAttention = (s) =>
  !!s && ((s.active !== false && s.status === 'waiting_input') || !!s.unseen_result);

export const claudeAttentionReason = (s) =>
  s?.status === 'waiting_input' ? 'Claude is waiting for your answer' : 'Claude finished a task you have not seen';
