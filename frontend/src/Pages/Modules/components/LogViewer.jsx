import React, {
  useState,
  useEffect,
  useRef,
  useLayoutEffect,
  useImperativeHandle,
  forwardRef,
} from 'react';
import { socketService } from 'Global/SocketService';
import './LogViewer.css';

const LogViewer = forwardRef(({ logType = 'module', moduleId = "", containerName = "" }, ref) => {
  const [logs, setLogs]           = useState([]);
  const [nextToken, setNextToken] = useState(null);
  const containerRef              = useRef(null);
  const isLoadingRef              = useRef(false);
  const prevScrollHeightRef       = useRef(0);
  const wasAtBottomBeforeRef      = useRef(false);
  const firstLoadRef = useRef(true);
  let source = null;
  if (logType === 'module') {
    source = `/api/v1/modules/${moduleId}/logs`;
  } else if (logType === 'container') {
    source = `/api/v1/modules/${moduleId}/containers/${containerName}/logs`;
  }

  useEffect(() => {
    if (!firstLoadRef.current || logs.length === 0) return;
    const container = containerRef.current;
    if (container) {
      container.scrollTop = container.scrollHeight;
      firstLoadRef.current = false;
    }
  }, [logs]);

  useEffect(() => {
  if (logType !== 'module' || !moduleId) return;
    socketService.subscribeModule(moduleId);
    const unsubscribe = socketService.subscribe(msg => {
      if (msg.eventType === 'log' && msg.module_id === moduleId) {
        appendLog({
          ...msg.payload,
          created_at: msg.timestamp,
        });
      }
    });

    return () => {
      socketService.unsubscribeModule(moduleId);
      unsubscribe();
    };
  }, [logType, moduleId]);

  const fetchLogs = async (token = null) => {
    if (isLoadingRef.current) return;
    isLoadingRef.current = true;

    const container = containerRef.current;
    if (token && container) {
      prevScrollHeightRef.current = container.scrollHeight;
    }

    try {
      const params = new URLSearchParams();
      if (logType === 'module') {
        params.set('order', '-created_at');
        if (token) params.set('next_page_token', token);
        else params.set('limit', 50);
      }

      const url = logType === 'module' ? `${source}?${params}` : source;
      const res = await fetch(url);
      if (!res.ok) throw new Error(res.statusText);
      const data = await res.json();

      if (logType === 'module') {
        setLogs(prev => token ? [...prev, ...data.logs] : data.logs);
        setNextToken(data.next_page_token);
      } else {
        setLogs(data);
      }
    } catch (err) {
      console.error('Failed to fetch logs:', err);
      setLogs([`Failed to fetch logs: ${err.message}`]);
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
  }), [source, logType]);

  useLayoutEffect(() => {
    const c = containerRef.current;
    if (!c) return;
    if (wasAtBottomBeforeRef.current) {
      c.scrollTop = c.scrollHeight;
    }
    wasAtBottomBeforeRef.current = false;
  }, [logs]);

  useEffect(() => {
    setLogs([]);
    setNextToken(null);
    fetchLogs();
    firstLoadRef.current = true;
  }, [source, logType]);

  const onScroll = () => {
    const el = containerRef.current;
    if (!el) return;
    if (el.scrollTop === 0 && nextToken) {
      fetchLogs(nextToken);
    }
  };

  const displayLogs = logType === 'module' ? logs.slice().reverse() : logs;

  const renderLog = (log, i) => {
    if (logType === 'container') {
      return (
        <div key={i} className="log-entry level-info">
          {log}
        </div>
      );
    }

    const level = (log.level || log.Level || 'info').toLowerCase();
    const cls = `log-entry level-${level}`;
    const tsTag = log.created_at || log.timestamp;
    const ts = new Date(tsTag).toLocaleString();

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
              {isError ? val : `${key}: ${val}`}
            </div>
          );
        })}
      </div>
    );
  };

  return (
    <div className="log-box" ref={containerRef} onScroll={onScroll}>
      {displayLogs.length === 0 ? (
        <div className="log-placeholder">No logs available</div>
      ) : (
        displayLogs.map(renderLog)
      )}
    </div>
  );
});

export default LogViewer;
