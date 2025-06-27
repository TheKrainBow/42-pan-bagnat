import React, {
  useState,
  useEffect,
  useRef,
  forwardRef,
  useImperativeHandle
} from 'react';
import './ModuleLogs.css';

const ModuleLogs = forwardRef(({ moduleId }, ref) => {
  const [logs, setLogs]           = useState([]);
  const [nextToken, setNextToken] = useState(null);
  const containerRef              = useRef(null);
  const isLoadingRef              = useRef(false);
  const prevScrollHeightRef       = useRef(0);

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
      if (token) {
        params.set('next_page_token', token);
      } else {
        params.set('limit', 20);
      }
      const res = await fetch(
        `http://localhost:8080/api/v1/modules/${moduleId}/logs?${params}`
      );
      if (!res.ok) throw new Error(res.statusText);
      const data = await res.json();

      setLogs(prev =>
        token ? [...prev, ...data.logs] : data.logs
      );
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

  // expose `refresh()` to parent via ref
  useImperativeHandle(ref, () => ({
    refresh: () => fetchLogs()
  }), [moduleId]);

  // initial load & when moduleId changes
  useEffect(() => {
    setLogs([]);
    setNextToken(null);
    fetchLogs();
  }, [moduleId]);

  const onScroll = () => {
    const el = containerRef.current;
    if (!el) return;
    if (el.scrollTop === 0 && nextToken) {
      fetchLogs(nextToken);
    }
  };

  const displayLogs = logs.slice().reverse();

  return (
    <div
      className="log-box"
      ref={containerRef}
      onScroll={onScroll}
    >
      {displayLogs.length === 0
        ? <div className="log-placeholder">No logs registered yet</div>
        : displayLogs.map((log,i)=> {
            const cls = `log-entry level-${log.level.toLowerCase()}`;
            const ts  = new Date(log.created_at).toLocaleString();
            return (
              <div key={i} className={cls}>
                <div className="log-main">
                  {`${ts} [${log.level}] ${log.message}`}
                </div>
                {log.meta?.error && (
                  <div className="log-error-detail">
                    {log.meta.error}
                  </div>
                )}
              </div>
            );
          })
      }
    </div>
  );
});

export default ModuleLogs;
