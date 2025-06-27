// components/ModuleLogs.jsx
import React, { useState, useEffect, useRef } from 'react';
import './ModuleLogs.css';

const ModuleLogs = ({ moduleId }) => {
  const [logs, setLogs] = useState([]);
  const [nextToken, setNextToken] = useState(null);
  const containerRef = useRef(null);
  const isLoadingRef = useRef(false);
  const prevScrollHeightRef = useRef(0);

  // Fetch logs: if token==null, initial newest; otherwise older page
  const fetchLogs = async (token = null) => {
    if (isLoadingRef.current) return;
    isLoadingRef.current = true;

    const container = containerRef.current;
    if (token && container) {
      // remember the full height before fetching older logs
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
        token
          ? [...prev, ...data.logs] // append older logs at end of our array
          : data.logs               // initial load
      );
      setNextToken(data.next_page_token);
    } catch (err) {
      console.error('Failed to fetch module logs:', err);
    } finally {
      isLoadingRef.current = false;
      // after React has painted with new logs, adjust scroll
      requestAnimationFrame(() => {
        if (!container) return;
        if (token) {
          // restore scroll so content doesn't jump
          const newHeight = container.scrollHeight;
          container.scrollTop = newHeight - prevScrollHeightRef.current;
        } else {
          // initial load: scroll all the way to bottom
          container.scrollTop = container.scrollHeight;
        }
      });
    }
  };

  // initial load & reset when moduleId changes
  useEffect(() => {
    setLogs([]);
    setNextToken(null);
    fetchLogs();
  }, [moduleId]);

  // if user scrolls to the top, load older logs
  const onScroll = () => {
    const el = containerRef.current;
    if (!el) return;
    if (el.scrollTop === 0 && nextToken) {
      fetchLogs(nextToken);
    }
  };

  return (
    <div
      className="log-box"
      ref={containerRef}
      onScroll={onScroll}
    >
      {logs.length === 0 ? (
        <div className="log-placeholder">No logs yet</div>
      ) : (
        // logs are stored newest→oldest, so reverse to show oldest at top
        logs
          .slice()
          .reverse()
          .map((log, i) => {
            const cls = `log-entry level-${log.level.toLowerCase()}`;
            const ts = new Date(log.created_at).toLocaleString();
            return (
              <div key={i} className={cls}>
                {`${ts} [${log.level}] ${log.message}`}
              </div>
            );
          })
      )}
    </div>
  );
};

export default ModuleLogs;
