import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { toast } from 'react-toastify';
import Button from 'Global/Button/Button';
import { fetchWithAuth } from 'Global/utils/Auth';
import { socketService } from 'Global/SocketService/SocketService';
import ClaudeAttentionDot from './ClaudeAttention';
import { claudeNeedsAttention } from './claudeAttentionStatus';
import './ModuleClaudePanel.css';

const ACTIVE = ['starting', 'idle', 'working', 'waiting_input'];

const STATUS_LABELS = {
  starting: 'Starting',
  idle: 'Idle',
  working: 'Working',
  waiting_input: 'Waiting for you',
  stopped: 'Stopped',
  aborted: 'Aborted',
};

const formatDate = (iso) => {
  if (!iso) return '';
  const d = new Date(iso);
  const sameDay = d.toDateString() === new Date().toDateString();
  return sameDay
    ? d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    : d.toLocaleString([], { dateStyle: 'short', timeStyle: 'short' });
};

export const ClaudeStatusBadge = ({ status }) => (
  <span className={`claude-status claude-status-${status}`}>{STATUS_LABELS[status] || status}</span>
);

// Terminal attached to the session's tmux. Unmounting (tab closed, other session
// opened) only detaches: claude keeps running in its container.
const ClaudeTerminal = ({ moduleId, sessionId, onClosed }) => {
  const containerRef = useRef(null);
  const [connected, setConnected] = useState(false);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'monospace',
      theme: { background: '#000000' },
      scrollback: 0, // tmux keeps the history
    });
    // Ctrl+Shift+C copies the selection instead of opening the browser inspector.
    term.attachCustomKeyEventHandler((ev) => {
      if (ev.type === 'keydown' && ev.ctrlKey && ev.shiftKey && ev.code === 'KeyC') {
        ev.preventDefault();
        const selection = term.getSelection();
        if (selection) navigator.clipboard?.writeText(selection).catch(() => {});
        return false;
      }
      return true;
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(containerRef.current);
    fit.fit();

    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const url = `${protocol}://${window.location.host}/api/v1/admin/modules/${encodeURIComponent(moduleId)}/claude/sessions/${encodeURIComponent(sessionId)}/terminal`;
    const ws = new WebSocket(url);
    ws.binaryType = 'arraybuffer';
    const encoder = new TextEncoder();
    let closedByUs = false;
    let retryTimer = null;

    const sendResize = () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
      }
    };

    ws.onopen = () => { setConnected(true); sendResize(); term.focus(); };
    ws.onmessage = (ev) => {
      if (typeof ev.data === 'string') {
        try {
          const msg = JSON.parse(ev.data);
          if (msg.type === 'error') term.write(`\r\n\x1b[31m${msg.message}\x1b[0m\r\n`);
        } catch { /* ignore */ }
        return;
      }
      term.write(new Uint8Array(ev.data));
    };
    ws.onclose = () => {
      setConnected(false);
      if (closedByUs) return;
      // Network blip or backend restart: claude is still running, reattach.
      // The parent unmounts this terminal if the session actually ended.
      onClosed?.();
      retryTimer = setTimeout(() => setAttempt((a) => a + 1), 3000);
    };

    const dataSub = term.onData((d) => {
      if (ws.readyState === WebSocket.OPEN) ws.send(encoder.encode(d));
    });
    const resizeSub = term.onResize(sendResize);
    const observer = new ResizeObserver(() => fit.fit());
    observer.observe(containerRef.current);

    return () => {
      closedByUs = true;
      clearTimeout(retryTimer);
      observer.disconnect();
      dataSub.dispose();
      resizeSub.dispose();
      ws.close();
      term.dispose();
    };
  }, [moduleId, sessionId, attempt, onClosed]);

  return (
    <div className="claude-terminal-wrapper">
      {!connected && (
        <div className="claude-terminal-overlay">
          <span className="claude-spinner" /> Connecting…
          <Button label="Retry" color="gray" onClick={() => setAttempt((a) => a + 1)} />
        </div>
      )}
      <div className="claude-terminal" ref={containerRef} />
    </div>
  );
};

const ModuleClaudePanel = ({ module }) => {
  const moduleId = module.id;
  const [sessions, setSessions] = useState([]);
  const [occupant, setOccupant] = useState(null);
  const [me, setMe] = useState('');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  // Session whose terminal is shown; defaults to the caller's running session.
  const [openSessionId, setOpenSessionId] = useState(null);
  const [terminalKey, setTerminalKey] = useState(0);

  const load = useCallback(async () => {
    try {
      const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/claude/sessions`);
      if (!res?.ok) throw new Error(res?.statusText || 'request failed');
      const data = await res.json();
      setSessions(Array.isArray(data.sessions) ? data.sessions : []);
      setOccupant(data.occupant || null);
      setMe(data.me || '');
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [moduleId]);

  useEffect(() => { load(); }, [load]);

  // Live status updates (hooks reported by the container).
  useEffect(() => {
    socketService.subscribeTopic(`module:${moduleId}`);
    const unsub = socketService.subscribe((evt) => {
      if (evt?.eventType === 'claude_status' && evt?.module_id === moduleId) load();
    });
    return () => { socketService.unsubscribeTopic(`module:${moduleId}`); unsub(); };
  }, [moduleId, load]);

  // The tab is on screen: the results of finished tasks are now seen.
  const hasUnseen = sessions.some((s) => s.unseen_result);
  useEffect(() => {
    const markSeen = () => {
      if (document.visibilityState !== 'visible' || !hasUnseen) return;
      fetchWithAuth(`/api/v1/admin/modules/${moduleId}/claude/seen`, { method: 'POST' }).catch(() => {});
    };
    markSeen();
    document.addEventListener('visibilitychange', markSeen);
    return () => document.removeEventListener('visibilitychange', markSeen);
  }, [moduleId, hasUnseen]);

  const mine = occupant && occupant.user_id === me;
  const takenByOther = occupant && !mine;
  const runningId = sessions.find((s) => ACTIVE.includes(s.status) && s.id === occupant?.session_id)?.id || null;

  // The caller's running session (if any) is the one shown in the terminal.
  useEffect(() => {
    setOpenSessionId(runningId);
  }, [runningId]);

  const askNotifications = () => {
    // Asked on a click: browsers ignore permission requests without a user gesture.
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission().catch(() => {});
    }
  };

  const start = async (resume) => {
    askNotifications();
    setBusy(true);
    try {
      const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/claude/sessions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(resume ? { resume } : {}),
      });
      if (res?.status === 409) {
        const data = await res.json().catch(() => ({}));
        toast.error(`Claude is already used on this module by ${data?.occupant?.user_login || 'someone else'}.`);
      } else if (!res?.ok) {
        const data = await res?.json?.().catch(() => ({}));
        toast.error(data?.message || 'Unable to start Claude.');
      } else {
        const sess = await res.json();
        setOpenSessionId(sess.id);
        setTerminalKey((k) => k + 1);
      }
    } catch (err) {
      console.error(err);
      toast.error('Unable to start Claude.');
    } finally {
      setBusy(false);
      load();
    }
  };

  // Stop lets claude exit cleanly; kill destroys the container right away, for a
  // session that hangs. Both keep the conversation, which can be resumed.
  const stop = async (sess, kill = false) => {
    const working = sess.status === 'working' || sess.status === 'waiting_input';
    const question = kill
      ? 'Kill this session? Its container is destroyed immediately and the current task is lost. The conversation can be resumed.'
      : 'Claude is working on a task. Stop it anyway?';
    if ((kill || working) && !window.confirm(question)) return;
    const query = kill ? '?kill=true' : (working ? '?force=true' : '');
    setBusy(true);
    try {
      const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/claude/sessions/${sess.id}/stop${query}`, { method: 'POST' });
      if (!res?.ok) {
        const data = await res?.json?.().catch(() => ({}));
        toast.error(data?.message || 'Unable to stop the session.');
      }
    } finally {
      setBusy(false);
      load();
    }
  };

  const remove = async (sess) => {
    const active = ACTIVE.includes(sess.status) && sess.id === occupant?.session_id;
    const working = sess.status === 'working' || sess.status === 'waiting_input';
    const what = sess.title ? `"${sess.title}"` : 'this session';
    if (!window.confirm(`Delete ${what} and its conversation history?${active ? ' It will be stopped.' : ''}`)) return;
    setBusy(true);
    try {
      const res = await fetchWithAuth(
        `/api/v1/admin/modules/${moduleId}/claude/sessions/${sess.id}${working ? '?force=true' : ''}`,
        { method: 'DELETE' },
      );
      if (!res?.ok) toast.error('Unable to delete the session.');
    } finally {
      setBusy(false);
      load();
    }
  };

  const openSession = sessions.find((s) => s.id === openSessionId);

  return (
    <div className="claude-panel">
      <div className="claude-prod-banner">
        <strong>PRODUCTION</strong> — Claude works directly on the live <code>{module.slug}</code> module
        (repository and containers). There is no staging.
      </div>

      {takenByOther && (
        <div className="claude-occupied">
          Claude is in use on this module by <strong>{occupant.user_login}</strong> since {formatDate(occupant.since)}.
          Only one Claude session can run per module.
        </div>
      )}

      <div className="claude-toolbar">
        <Button
          label="New session"
          color="blue"
          onClick={() => start(null)}
          disabled={busy || loading || !!occupant}
          disabledMessage={takenByOther ? `Claude is in use by ${occupant.user_login}` : 'A session is already running, stop it first'}
        />
        <span className="claude-help">
          Not logged in yet? Type <code>/login</code> in the terminal and follow the instructions, with your own Claude account.
        </span>
      </div>

      {openSession && ACTIVE.includes(openSession.status) && (
        <div className="claude-current">
          <div className="claude-current-header">
            <ClaudeStatusBadge status={openSession.status} />
            <span className="claude-current-title">{openSession.title || 'New conversation'}</span>
            <span className="claude-current-hint">Closing this tab does not stop Claude.</span>
            <Button label="Stop" color="red" onClick={() => stop(openSession)} disabled={busy} title="Let Claude exit cleanly" />
          </div>
          <ClaudeTerminal
            key={`${openSession.id}-${terminalKey}`}
            moduleId={moduleId}
            sessionId={openSession.id}
            onClosed={load}
          />
        </div>
      )}

      <div className="claude-sessions">
        <h4>Your sessions on this module</h4>
        {loading && <div className="claude-empty">Loading…</div>}
        {!loading && sessions.length === 0 && <div className="claude-empty">No session yet.</div>}
        {sessions.map((s) => {
          const isRunning = ACTIVE.includes(s.status) && s.id === occupant?.session_id;
          return (
            <div key={s.id} className={`claude-session-row${s.id === openSessionId ? ' open' : ''}`}>
              {claudeNeedsAttention({ ...s, active: isRunning })
                ? <ClaudeAttentionDot session={s} />
                : <span className="claude-attention-placeholder" />}
              <ClaudeStatusBadge status={isRunning ? s.status : (s.status === 'aborted' ? 'aborted' : 'stopped')} />
              <span className="claude-session-title">{s.title || 'Untitled conversation'}</span>
              <span className="claude-session-date">{formatDate(s.last_activity_at)}</span>
              <span className="claude-session-actions">
                {isRunning ? (
                  <>
                    {s.id !== openSessionId && <Button label="Open" color="gray" onClick={() => setOpenSessionId(s.id)} />}
                    <Button label="Kill" color="red" onClick={() => stop(s, true)} title="Destroy the container now (session stuck or frozen)" />
                  </>
                ) : (
                  <Button
                    label="Resume"
                    color="gray"
                    onClick={() => start(s.id)}
                    disabled={busy || !!occupant || !s.claude_session_id}
                    disabledMessage={!s.claude_session_id ? 'This conversation was never started' : 'Another session is running'}
                  />
                )}
                <Button label="Delete" color="red" onClick={() => remove(s)} disabled={busy} />
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default ModuleClaudePanel;
