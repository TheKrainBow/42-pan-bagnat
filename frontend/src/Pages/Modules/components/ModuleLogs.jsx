import React, {
  useState,
  useEffect,
  useRef,
  useLayoutEffect,
  useImperativeHandle,
  forwardRef,
} from 'react';
import './ModuleLogs.css';

// ModuleLogs displays a scrollable log box for a given module and listens for real-time log events over WebSocket.
const ModuleLogs = forwardRef(({ moduleId }, ref) => {
  const [logs, setLogs]           = useState([]);
  const [nextToken, setNextToken] = useState(null);
  const containerRef              = useRef(null);
  const isLoadingRef              = useRef(false);
  const prevScrollHeightRef       = useRef(0);
  const wasAtBottomBeforeRef      = useRef(false);

  // Fetch logs via REST API
  const fetchLogs = async (token = null) => {
    if (isLoadingRef.current) return;
    isLoadingRef.current = true;

    const container = containerRef.current;
    if (token && container) {
      prevScrollHeightRef.current = container.scrollHeight;
    }

    try {
      const params = new URLSearchParams();
      params.set('order', '-created_at');
      if (token) params.set('next_page_token', token);
      else params.set('limit', 20);

      const res = await fetch(
        `http://localhost:8080/api/v1/modules/${moduleId}/logs?${params}`
      );
      if (!res.ok) throw new Error(res.statusText);
      const data = await res.json();

      setLogs(prev => token ? [...prev, ...data.logs] : data.logs);
      setNextToken(data.next_page_token);
    } catch (err) {
      console.error('Failed to fetch module logs:', err);
    } finally {
      isLoadingRef.current = false;
      requestAnimationFrame(() => {
        if (!container) return;
        if (token) {
          const newHeight = container.scrollHeight;
          container.scrollTop = newHeight - prevScrollHeightRef.current;
        } else {
          container.scrollTop = container.scrollHeight;
        }
      });
    }
  };

  useImperativeHandle(ref, () => ({
    refresh: () => {
      setLogs([]);
      setNextToken(null);
      fetchLogs();
    },
    appendLog: log => {
      const c = containerRef.current;
      if (c) {
        wasAtBottomBeforeRef.current =
          Math.abs((c.scrollHeight - c.scrollTop) - c.clientHeight) <= 5;
      }
      setLogs(prev => [log, ...prev]);
    }
  }), [moduleId]);

  useLayoutEffect(() => {
    const c = containerRef.current;
    if (!c) return;
    if (wasAtBottomBeforeRef.current) {
      c.scrollTop = c.scrollHeight;
    }
    // reset for next batch
    wasAtBottomBeforeRef.current = false;
  }, [logs]);

  // Load initial logs when moduleId changes
  useEffect(() => {
    setLogs([]);
    setNextToken(null);
    fetchLogs();
  }, [moduleId]);

  // Infinite scroll: load more when scrolled to top
  const onScroll = () => {
    const el = containerRef.current;
    if (!el) return;
    if (el.scrollTop === 0 && nextToken) {
      fetchLogs(nextToken);
    }
  };

  const displayLogs = logs.slice().reverse();

  return (
    <div className="log-box" ref={containerRef} onScroll={onScroll}>
      {displayLogs.length === 0 ? (
        <div className="log-placeholder">No logs registered yet</div>
      ) : (
        displayLogs.map((log, i) => {
          const level = (log.level || log.Level).toLowerCase();
          const cls = `log-entry level-${level}`;
          const tsTag = log.created_at || log.timestamp;
          const ts  = new Date(tsTag).toLocaleString();

          return (
            <div key={i} className={cls}>
              <div className="log-main">
                {`${ts} [${log.level}] ${log.message}`}
              </div>
              {log.meta && Object.entries(log.meta).map(([key, val]) => {
                const isError = key === 'error';
                return (
                  <div
                    key={key}
                    className={isError ? 'log-error-detail' : 'log-meta-detail'}
                  >
                    {isError
                      ? val               // show just the error message
                      : `${key}: ${val}` // prefix others with their key
                    }
                  </div>
                )
              })}
            </div>
          );
        })
      )}
    </div>
  );
});

export default ModuleLogs;
