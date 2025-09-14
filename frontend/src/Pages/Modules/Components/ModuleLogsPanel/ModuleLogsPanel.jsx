import React, { useEffect, useMemo, useRef, useState } from 'react';
import './ModuleLogsPanel.css';
import { socketService } from 'Global/SocketService/SocketService';
import { fetchWithAuth } from 'Global/utils/Auth';

// Normalized log entry
// { sourceType: 'module'|'container', source: string, timestamp: string, message: string, level?: string, meta?: object }

function loadSelection(moduleId) {
  try {
    const raw = localStorage.getItem(`logs.selection.${moduleId}`);
    if (!raw) return { module: true, containers: {} };
    const parsed = JSON.parse(raw);
    // basic shape guard
    if (!parsed || typeof parsed !== 'object') return { module: true, containers: {} };
    if (typeof parsed.module !== 'boolean' || typeof parsed.containers !== 'object') {
      return { module: true, containers: {} };
    }
    return parsed;
  } catch { return { module: true, containers: {} }; }
}

function saveSelection(moduleId, sel) {
  try { localStorage.setItem(`logs.selection.${moduleId}`, JSON.stringify(sel)); } catch {}
}

export default function ModuleLogsPanel({ moduleId }) {
  const [containers, setContainers] = useState([]);
  const [selected, setSelected] = useState(() => loadSelection(moduleId));
  const [logs, setLogs] = useState([]);
  const [modNextToken, setModNextToken] = useState(null);
  const [modLoading, setModLoading] = useState(false);
  const boxRef = useRef(null);
  const backfillLockRef = useRef(false);
  const pinOnNextRenderRef = useRef(false);
  const stickToBottomRef = useRef(true);
  const modLastTokenRef = useRef(null);
  const seqRef = useRef(0);
  const containerLoadedRef = useRef({});
  const containerLastTsRef = useRef({});
  const containerSubscribedRef = useRef({});

  // Fetch containers list
  useEffect(() => {
    fetchWithAuth(`/api/v1/admin/modules/${moduleId}/docker/ls`)
      .then(res => res.json())
      .then(data => setContainers(data || []))
      .catch(() => setContainers([]));
  }, [moduleId]);

  // Subscribe to containers updates over WS to keep selector in sync
  useEffect(() => {
    const topic = `containers:${moduleId}`;
    socketService.subscribeTopic(topic);
    const unsub = socketService.subscribe(msg => {
      if (msg.eventType === 'containers_updated' && msg.module_id === moduleId && Array.isArray(msg.payload)) {
        setContainers(msg.payload.map(c => ({ name: c.name, status: c.status, since: c.since, reason: c.reason })));
      }
    });
    return () => { socketService.unsubscribeTopic(topic); unsub(); };
  }, [moduleId]);

  // Auto-subscribe and backfill for containers pre-selected on load
  useEffect(() => {
    (containers || []).forEach(c => {
      const name = c.name || c.Name;
      if (!name) return;
      const want = !!selected.containers[name];
      const isSub = !!containerSubscribedRef.current[name];
      if (want && !isSub) {
        // subscribe to WS
        socketService.subscribeTopic(`container:${moduleId}:${name}`);
        containerSubscribedRef.current[name] = true;
        // fetch initial tail or since last seen
        const since = containerLastTsRef.current[name] || null;
        if (!containerLoadedRef.current[name] || since) {
          fetchContainerLogs(moduleId, name, since).then(lines => {
            const entries = lines.map(line => {
              const { ts, msg } = parseDockerTimestamptedLine(line);
              return { sourceType: 'container', source: name, timestamp: ts, message: msg };
            });
            setLogs(prev => mergeAndDedupe(prev, entries));
            containerLoadedRef.current[name] = true;
            if (entries.length > 0) {
              containerLastTsRef.current[name] = entries[entries.length - 1].timestamp;
            }
          });
        }
      } else if (!want && isSub) {
        socketService.unsubscribeTopic(`container:${moduleId}:${name}`);
        containerSubscribedRef.current[name] = false;
      }
    });
  }, [moduleId, containers, selected]);

  // Socket listener
  useEffect(() => {
    const unsub = socketService.subscribe(msg => {
      if (msg.eventType === 'log' && selected.module && msg.module_id === moduleId) {
        const p = msg.payload || {};
        appendLog({
          sourceType: 'module',
          source: 'Module',
          timestamp: msg.timestamp,
          t: Date.parse(msg.timestamp),
          message: p.message || '',
          level: p.level || 'INFO',
          meta: p.meta || null,
          id: p.id,
          seq: seqRef.current++,
        });
      }
      if (msg.eventType === 'container_log' && msg.module_id === moduleId) {
        const p = msg.payload || {};
        const name = p.container;
        if (!name) return;
        if (!selected.containers[name]) return;
        const entry = {
          sourceType: 'container',
          source: name,
          timestamp: msg.timestamp,
          t: Date.parse(msg.timestamp),
          message: p.message || '',
          seq: seqRef.current++,
        };
        containerLastTsRef.current[name] = entry.timestamp;
        appendLog(entry);
        maybeBackfillFor(entry.timestamp);
      }
    });
    return () => unsub();
  }, [moduleId, selected]);

  // Manage module/container topic subscriptions with precise actions instead of bulk
  useEffect(() => {
    // on mount or moduleId change: subscribe module if selected, and selected containers
    if (selected.module) socketService.subscribeTopic(`module:${moduleId}`);
    for (const name of Object.keys(selected.containers)) {
      if (selected.containers[name]) socketService.subscribeTopic(`container:${moduleId}:${name}`);
    }
    return () => {
      // cleanup: unsubscribe everything we might have subscribed for this moduleId
      if (selected.module) socketService.unsubscribeTopic(`module:${moduleId}`);
      for (const name of Object.keys(selected.containers)) {
        if (selected.containers[name]) socketService.unsubscribeTopic(`container:${moduleId}:${name}`);
      }
    };
  }, [moduleId]);

  // initial fetches when toggling sources on
  useEffect(() => {
    // Reset view when module changes
    setLogs([]);
    modLastTokenRef.current = null;
    containerLoadedRef.current = {};
    // Reload persisted selection for this module
    setSelected(loadSelection(moduleId));
    // initial module history
    if (selected.module) {
      setModLoading(true);
      fetchModuleLogs(moduleId).then(({ entries, nextToken }) => {
        setLogs(prev => mergeAndDedupe(prev, entries));
        setModNextToken(nextToken || null);
        // On first load, pin to bottom
        maintainScrollAfterAppend(true, null);
        stickToBottomRef.current = true;
      }).finally(() => setModLoading(false));
    }
  }, [moduleId]);

  // build backfill function bound to current setters and selection
  const backfillLastTokenRef = useRef(null);
  const maybeBackfillFor = React.useMemo(
    () => maybeBackfillForFactory(
      moduleId,
      selected,
      () => modNextToken,
      setModNextToken,
      setLogs,
      backfillLockRef,
      backfillLastTokenRef,
      () => getEarliestModuleTimestamp(logs)
    ),
    [moduleId, selected, modNextToken, logs]
  );

  const isAtBottom = () => {
    const el = boxRef.current;
    if (!el) return false;
    return Math.abs((el.scrollHeight - el.scrollTop) - el.clientHeight) <= 5;
  };

  const maintainScrollAfterAppend = (wasPinned, prevScrollHeight) => {
    const el = boxRef.current;
    if (!el) return;
    requestAnimationFrame(() => {
      if (wasPinned) {
        el.scrollTop = el.scrollHeight;
      } else if (prevScrollHeight != null) {
        const newHeight = el.scrollHeight;
        el.scrollTop = newHeight - prevScrollHeight;
      }
    });
  };

  const appendLog = (entry) => {
    // Track stickiness and append; avoid expensive full dedupe by checking recent items
    stickToBottomRef.current = isAtBottom();
    setLogs(prev => {
      const k = keyFor(entry);
      // check last 10 items for duplicate
      for (let i = prev.length - 1, c = 0; i >= 0 && c < 10; i--, c++) {
        if (keyFor(prev[i]) === k) return prev;
      }
      return [...prev, entry];
    });
  };

  const toggleModule = () => {
    const pinned = isAtBottom();
    pinOnNextRenderRef.current = pinned;
    stickToBottomRef.current = pinned;
    const nextVal = !selected.module;
    setSelected(prev => {
      const n = { ...prev, module: nextVal };
      saveSelection(moduleId, n);
      return n;
    });
    if (nextVal) {
      // turning on → fetch last 50 module logs
      const pinned = isAtBottom();
      modLastTokenRef.current = null;
      setModLoading(true);
      fetchModuleLogs(moduleId).then(({ entries, nextToken }) => {
        setLogs(prev => mergeAndDedupe(prev, entries));
        setModNextToken(nextToken || null);
        maintainScrollAfterAppend(pinned, null);
      }).finally(() => setModLoading(false));
      socketService.subscribeTopic(`module:${moduleId}`);
    } else {
      // turning off → unsubscribe and keep scroll if pinned
      socketService.unsubscribeTopic(`module:${moduleId}`);
      maintainScrollAfterAppend(pinned, null);
    }
  };

  const toggleContainer = (name) => {
    const pinned = isAtBottom();
    pinOnNextRenderRef.current = pinned;
    stickToBottomRef.current = pinned;
    const nextVal = !selected.containers[name];
    setSelected(prev => {
      const n = { ...prev, containers: { ...prev.containers, [name]: nextVal } };
      saveSelection(moduleId, n);
      return n;
    });
    if (nextVal) {
      // turning on → fetch latest container logs (tail)
      const since = containerLastTsRef.current[name] || null;
      if (!containerLoadedRef.current[name] || since) {
        fetchContainerLogs(moduleId, name, since).then(lines => {
          const entries = lines.map(line => {
            const { ts, msg } = parseDockerTimestamptedLine(line);
            return { sourceType: 'container', source: name, timestamp: ts, t: Date.parse(ts), message: msg };
          });
          setLogs(prev => mergeAndDedupe(prev, entries));
          containerLoadedRef.current[name] = true;
          if (entries.length > 0) {
            containerLastTsRef.current[name] = entries[entries.length - 1].timestamp;
          }
          maintainScrollAfterAppend(pinned, null);
        });
      } else {
        maintainScrollAfterAppend(pinned, null);
      }
      socketService.subscribeTopic(`container:${moduleId}:${name}`);
    } else {
      // turning off → unsubscribe and maintain bottom if pinned
      socketService.unsubscribeTopic(`container:${moduleId}:${name}`);
      maintainScrollAfterAppend(pinned, null);
    }
  };

  // After render, keep pinned if sticky or explicitly flagged
  React.useLayoutEffect(() => {
    const el = boxRef.current;
    if (!el) return;
    if (stickToBottomRef.current) {
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          el.scrollTop = el.scrollHeight;
        });
      });
    } else if (pinOnNextRenderRef.current) {
      el.scrollTop = el.scrollHeight;
      pinOnNextRenderRef.current = false;
    }
  }, [logs, selected]);

  // Assign deterministic colors to containers (by name order), 8-color palette cycling
  const moduleColor = '#8ab4f8';
  const containerColors = useMemo(() => {
    // Palette excludes module blue; shifted by one to avoid first container matching module color family
    const palette = [
      '#ff7b72', // red-ish
      '#7bffb2', // mint
      '#ffb27b', // orange
      '#d07bff', // purple
      '#ffd27b', // yellow
      '#7bffd2', // aquamarine
      '#ff7bd2', // pink
      '#9ccc65', // green
    ];
    const shift = 1;
    const map = {};
    const names = (containers || []).map(c => c.name || c.Name).filter(Boolean).sort();
    names.forEach((n, i) => { map[n] = palette[(i + shift) % palette.length]; });
    return map;
  }, [containers]);

  const renderedLogs = useMemo(() => {
    const filtered = logs.filter(e =>
      (e.sourceType === 'module' && selected.module) ||
      (e.sourceType === 'container' && selected.containers[e.source])
    );
    return filtered.sort((a, b) => {
      const ta = a.t ?? Date.parse(a.timestamp);
      const tb = b.t ?? Date.parse(b.timestamp);
      if (ta !== tb) return ta - tb;
      // Stable tie-breakers: module log id if present, otherwise seq
      if (a.sourceType === 'module' && b.sourceType === 'module' && a.id != null && b.id != null) {
        return a.id - b.id;
      }
      const sa = a.seq ?? 0, sb = b.seq ?? 0;
      return sa - sb;
    });
  }, [logs, selected]);

  // Infinite scroll (older module logs) when reaching top
  const onScroll = () => {
    const el = boxRef.current;
    if (!el) return;
    // Update stickiness when user scrolls
    stickToBottomRef.current = Math.abs((el.scrollHeight - el.scrollTop) - el.clientHeight) <= 5;
    if (el.scrollTop === 0 && selected.module && modNextToken && !modLoading) {
      const tokenToUse = modNextToken;
      // prevent refetching same page repeatedly
      if (modLastTokenRef.current === tokenToUse) return;
      modLastTokenRef.current = tokenToUse;

      const prevHeight = el.scrollHeight;
      setModLoading(true);
      fetchModuleLogs(moduleId, tokenToUse)
        .then(({ entries, nextToken }) => {
          setLogs(prev => mergeAndDedupe(prev, entries));
          // advance token; if backend misbehaves and returns same token, null it to avoid loops
          setModNextToken(nextToken && nextToken !== tokenToUse ? nextToken : (nextToken === tokenToUse ? null : nextToken));
          setModLoading(false);
          maintainScrollAfterAppend(false, prevHeight);
        })
        .catch(() => {
          setModLoading(false);
        });
    }
  };

  return (
    <div className="module-logs-layout">
      <div className="log-selector">
        <div className="selector-title">Sources</div>
        <label className="selector-item">
          <input type="checkbox" className="selector-checkbox" style={{ "--cb-color": '#8ab4f8' }} checked={!!selected.module} onChange={toggleModule} />
          <span className="selector-label">Module Logs</span>
        </label>
        <div className="selector-subtitle">Containers</div>
        {(!containers || containers.length === 0) && (
          <div className="selector-empty">No containers</div>
        )}
        {containers && containers.map(c => (
          <label key={c.name} className="selector-item">
            <input type="checkbox" className="selector-checkbox" style={{ "--cb-color": containerColors[c.name] || '#e8eaed' }} checked={!!selected.containers[c.name]} onChange={() => toggleContainer(c.name)} />
            <span className="selector-label">{c.name}</span>
          </label>
        ))}
      </div>
      <div className="log-panel" ref={boxRef} onScroll={onScroll}>
        {renderedLogs.length === 0 ? (
          (selected.module || Object.values(selected.containers).some(Boolean)) ? (
            <div className="log-placeholder">{modLoading ? 'Loading logs…' : 'No logs available'}</div>
          ) : (
            <div className="log-placeholder">No logs selected</div>
          )
        ) : (
          renderedLogs.map((e, idx) => (
            <LogLine key={idx} entry={e} containerColors={containerColors} />
          ))
        )}
      </div>
    </div>
  );
}

function LogLine({ entry, containerColors }) {
  const ts = new Date(entry.timestamp);
  const dateStr = ts.toLocaleDateString();
  const timeStr = ts.toLocaleTimeString([], { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
  const level = (entry.level || '').toLowerCase();
  const isModule = entry.sourceType === 'module';
  const cls = isModule ? `log-row level-${level || 'info'}` : 'log-row';
  const srcColor = isModule ? '#8ab4f8' : (containerColors?.[entry.source] || '#e8eaed');
  return (
    <div className={cls}>
      <span className="log-src" style={{ color: srcColor }}>[{entry.source}]</span>
      <span className="log-date">[{dateStr}]</span>
      <span className="log-time">[{timeStr}]</span>
      <span className="log-msg">
        {isModule && entry.level ? `[${entry.level}] ` : ''}{entry.message}
      </span>
      {isModule && entry.meta && Object.entries(entry.meta).map(([k, v]) => (
        <div key={k} className={k === 'error' ? 'log-meta-error' : 'log-meta'}>
          {k === 'error' ? v : `${k}: ${v}`}
        </div>
      ))}
    </div>
  );
}

async function fetchModuleLogs(moduleId, nextToken = null) {
  try {
    const params = new URLSearchParams();
    params.set('order', '-created_at');
    if (nextToken) params.set('next_page_token', nextToken);
    else params.set('limit', 50);
    const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/logs?` + params.toString());
    if (!res.ok) return [];
    const data = await res.json();
    const entries = (data.logs || []).map(l => ({
      sourceType: 'module',
      source: 'Module',
      timestamp: l.created_at || l.timestamp,
      t: Date.parse(l.created_at || l.timestamp),
      message: l.message,
      level: l.level,
      meta: l.meta || null,
      id: l.id,
    }));
    return { entries, nextToken: data.next_page_token };
  } catch {
    return { entries: [], nextToken: null };
  }
}

async function fetchContainerLogs(moduleId, name, since = null) {
  try {
    const url = new URL(`/api/v1/admin/modules/${moduleId}/docker/${name}/logs`, window.location.origin);
    if (since) url.searchParams.set('since', since);
    const res = await fetchWithAuth(url.pathname + url.search);
    if (!res.ok) return [];
    return await res.json();
  } catch {
    return [];
  }
}

function parseDockerTimestamptedLine(line) {
  const idx = line.indexOf(' ');
  if (idx > 0) {
    const ts = line.slice(0, idx).trim();
    const msg = line.slice(idx + 1).trim();
    return { ts, msg };
  }
  return { ts: new Date().toISOString(), msg: line };
}

function mergeAndDedupe(existing, incoming) {
  const key = e => (e.sourceType === 'module' && e.id != null)
    ? `m:${e.id}`
    : `${e.sourceType}:${e.source}|${e.timestamp}|${e.level || ''}|${e.message}`;
  const seen = new Set(existing.map(key));
  const add = [];
  for (const e of incoming) {
    const k = key(e);
    if (!seen.has(k)) {
      seen.add(k);
      add.push(e);
    }
  }
  return [...existing, ...add];
}

function keyFor(e) {
  return (e.sourceType === 'module' && e.id != null)
    ? `m:${e.id}`
    : `${e.sourceType}:${e.source}|${e.timestamp}|${e.level || ''}|${e.message}`;
}

function getEarliestModuleTimestamp(logs) {
  let minTs = null;
  for (const e of logs) {
    if (e.sourceType !== 'module') continue;
    const t = new Date(e.timestamp).getTime();
    if (!isFinite(t)) continue;
    if (minTs == null || t < minTs) minTs = t;
  }
  return minTs;
}

async function backfillUntil(moduleId, targetTs, getNextToken, setNextToken, setLogs, loadingRef, lastTokenRef) {
  // Guard against concurrent backfills
  if (loadingRef.current) return;
  loadingRef.current = true;
  try {
    let nextToken = getNextToken();
    if (!nextToken) return;
    if (lastTokenRef.current === nextToken) return; // avoid reusing same token repeatedly
    lastTokenRef.current = nextToken;
    // Fetch up to 3 pages to avoid long loops
    for (let i = 0; i < 3 && nextToken; i++) {
      const { entries, nextToken: nxt } = await fetchModuleLogs(moduleId, nextToken);
      if (!entries || entries.length === 0) { setNextToken(null); lastTokenRef.current = null; break; }
      setLogs(prev => mergeAndDedupe(prev, entries));
      if (!nxt || nxt === nextToken) {
        // exhausted or backend sent identical token → stop
        setNextToken(null);
        lastTokenRef.current = null;
        break;
      }
      setNextToken(nxt);
      lastTokenRef.current = nxt;
      const earliest = getEarliestModuleTimestamp(entries);
      if (earliest != null && earliest <= targetTs) break;
      nextToken = nxt;
    }
  } finally {
    loadingRef.current = false;
  }
}

function maybeBackfillForFactory(moduleId, selected, getNextToken, setNextToken, setLogs, lockRef, lastTokenRef, getEarliestTs) {
  return (isoTs) => {
    if (!selected.module) return;
    const targetTs = new Date(isoTs).getTime();
    if (!isFinite(targetTs)) return;
    if (!getNextToken()) return;
    const earliest = getEarliestTs && getEarliestTs();
    if (earliest == null || !(targetTs < earliest)) return; // only backfill if container log is older than oldest module log displayed
    if (lockRef.current) return;
    // Trigger async backfill loop
    backfillUntil(moduleId, targetTs, getNextToken, setNextToken, setLogs, lockRef, lastTokenRef);
  };
}
