// Modules.jsx
import React, { useState, useEffect, useRef, useCallback } from 'react';
import './Modules.css';
import AppIcon from 'Global/AppIcon';
import Header from 'Global/Header';
import ModuleImport from 'Modules/components/ModuleImport';
import ModuleStatusBadge from 'Modules/components/ModuleStatusBadge';
import { Link } from 'react-router-dom';
import { setModuleStatusUpdater } from 'Global/SocketService';

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

      const res = await fetch(`/api/v1/modules?${params.toString()}`);
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


  const handleSubmit = async ({ gitUrl, sshKey }) => {
    try {
      const response = await fetch('/api/v1/modules', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          git_url: gitUrl,
          ssh_key: sshKey,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        console.error('Error from backend:', error);
        return;
      }

      const result = await response.json();
      console.log('Module created:', result);

      // Optionally refresh module list or trigger UI update
      fetchModules(false, '');
      setShowModal(false);
    } catch (err) {
      console.error('Request failed:', err);
    }
  };

  return (
    <div className="p-4">
      <Header
        title="Modules"
        value={filterQuery}
        onChange={(e) => setFilterQuery(e.target.value)}
        actionButtonLabel="Import Module"
        onActionButtonClick={handleImport}
        onFilterClick={handleImport}
      />
      <div className="modules-container" ref={scrollContainerRef}>
        <div className="modules-grid">
          {modules.map((mod) => (
            <Link key={mod.id} to={`/admin/modules/${mod.id}`} className={`module-card ${mod.status}`}>
              <div className="module-icon">
                <AppIcon app={{ icon_url: mod.icon_url, name: mod.name }} fallback="/icons/modules.png" />
              </div>
              <div className="module-content">
                <div className="module-title-row">
                  <strong>{mod.name}</strong>
                  <ModuleStatusBadge status={mod.status} />
                </div>
                  {mod.last_update && new Date(mod.last_update).getFullYear() > 2000 ? (
                  <>
                    <p className="module-description">v{mod.version} • {mod.late_commits} late commits</p>
                    <p className="module-updated">Last update: {new Date(mod.last_update).toLocaleDateString()}</p>
                  </>
                ) : (
                  <p className="module-waiting">Action required</p>
                )}
              </div>
            </Link>
          ))}
        </div>
        {nextPage && (
          <div className="load-more-wrapper">
            <div className="loader" style={{ width: 24, height: 24 }} />
          </div>
        )}
      </div>

      {showModal && (
        <ModuleImport onClose={handleClose} onSubmit={handleSubmit} />
      )}
    </div>
  );
};

export default Modules;
