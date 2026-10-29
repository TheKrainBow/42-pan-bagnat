// Modules.jsx
import React, { useState, useEffect, useRef, useCallback } from 'react';
import './Modules.css';
import AppIcon from 'Global/AppIcon/AppIcon';
import Header from 'Global/Header/Header';
import ModuleImport from 'Pages/Modules/Components/ModuleImport/ModuleImport';
import ModuleStatusBadge from 'Pages/Modules/Components/ModuleStatusBadge/ModuleStatusBadge';
import ModuleBadge from 'Global/ModuleBadge/ModuleBadge';
import { Link } from 'react-router-dom';
import { setModuleStatusUpdater, socketService } from 'Global/SocketService/SocketService';
import { fetchWithAuth } from 'Global/utils/Auth';
import { claudeNeedsAttention } from 'Pages/Modules/Components/ModuleClaudePanel/claudeAttentionStatus';

const Modules = () => {
  const [modules, setModules] = useState([]);
  const [filterQuery, setFilterQuery] = useState('');
  const [debouncedFilter, setDebouncedFilter] = useState('');
  const [nextPage, setNextPage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const loadingRef = useRef(false);
  const scrollContainerRef = useRef(null);
  const isFirst = useRef(true);
  const [showModal, setShowModal] = useState(false);
  // Claude sessions shown on the cards: running ones (anyone's) and the
  // caller's ended ones with an unseen result. session id -> session.
  const [claudeSessions, setClaudeSessions] = useState({});
  const [me, setMe] = useState('');

  useEffect(() => {
    let cancelled = false;
    fetchWithAuth('/api/v1/admin/claude/active')
      .then((res) => (res?.ok ? res.json() : {}))
      .then((data) => {
        if (cancelled || !Array.isArray(data?.sessions)) return;
        setMe(data.me || '');
        setClaudeSessions(Object.fromEntries(data.sessions.map((s) => [s.id, s])));
      })
      .catch(() => {});
    socketService.subscribeTopic('claude:status');
    const unsub = socketService.subscribe((evt) => {
      if (evt?.eventType !== 'claude_status' || !evt.payload) return;
      const s = evt.payload;
      setClaudeSessions((prev) => {
        const next = { ...prev };
        if (s.active || s.unseen_result) next[s.id] = s;
        else delete next[s.id];
        return next;
      });
    });
    return () => { cancelled = true; socketService.unsubscribeTopic('claude:status'); unsub(); };
  }, []);

  const claudeFor = (moduleId) => {
    const list = Object.values(claudeSessions).filter((s) => s.module_id === moduleId);
    return {
      active: list.find((s) => s.active) || null,
      // Only the owner is asked to come back: others just see the status line.
      attention: list.find((s) => s.user_id === me && claudeNeedsAttention(s)) || null,
    };
  };

  useEffect(() => {
    // Register live update handler
    setModuleStatusUpdater((id, newStatus) => {
      setModules(prev =>
        prev.map(mod =>
          mod.id === id ? { ...mod, status: newStatus } : mod
        )
      );
    });

    // Cleanup on unmount
    return () => {
      setModuleStatusUpdater(null);
    };
  }, []);

  const fetchModules = useCallback(async (append = false, token = '') => {
    if (loadingRef.current) return;
    loadingRef.current = true;
    setIsLoading(true);

    try {
      const params = new URLSearchParams();
      if (debouncedFilter) params.set('filter', debouncedFilter);
      if (token) {
        params.set('next_page_token', token);
      } else {
        params.set('limit', 20);
      }

      const res = await fetchWithAuth(`/api/v1/admin/modules?${params.toString()}`);
      const data = await res.json();

      setModules(prev =>
        append ? [...prev, ...data.modules] : Array.isArray(data.modules) ? data.modules : []
      );
      setNextPage(data.next_page_token);
    } catch (err) {
      console.error(err);
      if (!append) setModules([]);
    } finally {
      loadingRef.current = false;
      setIsLoading(false);
    }
  }, [debouncedFilter]);

  const handleScroll = useCallback(() => {
    const el = scrollContainerRef.current;
    if (!el || isLoading || !nextPage) return;
    if (el.scrollTop + el.clientHeight >= el.scrollHeight - 10) {
      fetchModules(true, nextPage);
    }
  }, [nextPage, isLoading, fetchModules]);

  useEffect(() => {
    const t = setTimeout(() => setDebouncedFilter(filterQuery), 300);
    return () => clearTimeout(t);
  }, [filterQuery]);

  useEffect(() => {
    const el = scrollContainerRef.current;
    el?.addEventListener('scroll', handleScroll);
    return () => el?.removeEventListener('scroll', handleScroll);
  }, [handleScroll]);

  useEffect(() => {
    fetchModules();
  }, [fetchModules]);

  useEffect(() => {
    if (isFirst.current) {
      isFirst.current = false;
      fetchModules();
    } else {
      fetchModules(false, '');
    }
  }, [debouncedFilter, fetchModules]);


  const handleImport = () => {
    setShowModal(true);
  };

  const handleClose = () => {
    setShowModal(false);
  };


  // Module import handled inside ModuleImport component

  return (
    <div className="p-4">
      <Header
        title="Modules"
        value={filterQuery}
        onChange={(e) => setFilterQuery(e.target.value)}
        actionButtonLabel="Import Module"
        onActionButtonClick={handleImport}
      />
      <div className="modules-container" ref={scrollContainerRef}>
        <div className="modules-grid">
          {modules.map((mod) => {
            const claude = claudeFor(mod.id);
            return <ModuleBadge key={mod.id} mod={mod} claude={claude.active} claudeAttention={claude.attention} />;
          })}
        </div>
        {nextPage && (
          <div className="load-more-wrapper">
            <div className="loader" style={{ width: 24, height: 24 }} />
          </div>
        )}
      </div>

      {showModal && (
        <ModuleImport onClose={handleClose} />
      )}
    </div>
  );
};

export default Modules;
