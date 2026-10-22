import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import Button from 'Global/Button/Button';
import './ModuleShellPanel.css';

const formatRemaining = (secs) => {
  const h = Math.floor(secs / 3600);
  const m = String(Math.floor((secs % 3600) / 60)).padStart(2, '0');
  const s = String(secs % 60).padStart(2, '0');
  return h > 0 ? `${h}:${m}:${s}` : `${m}:${s}`;
};

// Root shell in a sandbox container where only the module repo is mounted
// (/workspace). Access is enforced server-side (admin only, WebSocket).
// The container is only started on explicit user action, never on tab open.
const ModuleShellPanel = ({ moduleId }) => {
  const containerRef = useRef(null);
  const wsRef = useRef(null);
  // 'idle' -> 'starting' (container booting) -> 'connected' -> 'closed'
  const [phase, setPhase] = useState('idle');
  // Incremented on each start click; drives the session effect.
  const [session, setSession] = useState(0);
  // Remaining session time in seconds (server-enforced hard limit).
  const [remaining, setRemaining] = useState(null);
  const expiresAtRef = useRef(null);

  useEffect(() => {
    if (session === 0) return undefined;

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'monospace',
      theme: { background: '#000000' },
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
    const url = `${protocol}://${window.location.host}/api/v1/admin/modules/${encodeURIComponent(moduleId)}/shell`;
    const ws = new WebSocket(url);
    wsRef.current = ws;
    ws.binaryType = 'arraybuffer';
    const encoder = new TextEncoder();
    let gotOutput = false;

    const sendResize = () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
      }
    };

    ws.onopen = () => sendResize();
    ws.onmessage = (ev) => {
      if (typeof ev.data === 'string') {
        try {
          const msg = JSON.parse(ev.data);
          if (msg.type === 'session') {
            expiresAtRef.current = Date.now() + msg.expires_in * 1000;
            setRemaining(msg.expires_in);
          } else if (msg.type === 'error') {
            term.write(`\r\n\x1b[31m${msg.message}\x1b[0m\r\n`);
          }
        } catch { /* ignore */ }
        return;
      }
      if (!gotOutput) {
        // First shell output means the container is up and bash is running.
        gotOutput = true;
        setPhase('connected');
        term.focus();
      }
      term.write(new Uint8Array(ev.data));
    };
    ws.onclose = () => {
      setPhase('closed');
      term.write('\r\n\x1b[33m[session closed]\x1b[0m\r\n');
    };
    ws.onerror = () => setPhase('closed');

    const dataSub = term.onData((d) => {
      if (ws.readyState === WebSocket.OPEN) ws.send(encoder.encode(d));
    });
    const resizeSub = term.onResize(sendResize);
    const observer = new ResizeObserver(() => fit.fit());
    observer.observe(containerRef.current);

    const countdown = setInterval(() => {
      if (expiresAtRef.current) {
        setRemaining(Math.max(0, Math.round((expiresAtRef.current - Date.now()) / 1000)));
      }
    }, 1000);

    return () => {
      clearInterval(countdown);
      expiresAtRef.current = null;
      observer.disconnect();
      dataSub.dispose();
      resizeSub.dispose();
      ws.onclose = null;
      ws.close();
      wsRef.current = null;
      term.dispose();
    };
  }, [moduleId, session]);

  const start = () => {
    setRemaining(null);
    setPhase('starting');
    setSession((s) => s + 1);
  };

  const stop = () => {
    wsRef.current?.close();
  };

  return (
    <div className="module-shell">
      {phase === 'idle' && (
        <div className="module-shell-start">
          <p>Opens a root shell on the module repository (/workspace) in a temporary container.</p>
          <Button label="Start shell" color="blue" onClick={start} />
        </div>
      )}
      {phase === 'starting' && (
        <div className="module-shell-status">
          <span className="module-shell-spinner" /> Starting container…
        </div>
      )}
      {phase === 'connected' && (
        <div className="module-shell-status">
          Connected — root shell on the module repository (/workspace)
          {remaining !== null && (
            <span className={`module-shell-timer${remaining <= 300 ? ' warning' : ''}`}>
              Time remaining: {formatRemaining(remaining)}
            </span>
          )}
          <Button label="Stop" color="red" onClick={stop} />
        </div>
      )}
      {phase === 'closed' && (
        <div className="module-shell-status">
          Disconnected
          <Button label="Start new session" color="blue" onClick={start} />
        </div>
      )}
      <div
        className="module-shell-term"
        ref={containerRef}
        style={{ display: phase === 'idle' ? 'none' : undefined }}
      />
    </div>
  );
};

export default ModuleShellPanel;
