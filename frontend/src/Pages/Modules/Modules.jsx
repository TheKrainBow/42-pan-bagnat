// Modules.jsx
import React, { useState, useEffect, useRef, useCallback } from 'react';
import './Modules.css';
import AppIcon from 'Global/AppIcon/AppIcon';
import Header from 'Global/Header/Header';
import ModuleImport from 'Pages/Modules/Components/ModuleImport/ModuleImport';
import ModuleStatusBadge from 'Pages/Modules/Components/ModuleStatusBadge/ModuleStatusBadge';
import ModuleBadge from 'Global/ModuleBadge/ModuleBadge';
import { Link } from 'react-router-dom';
import { setModuleStatusUpdater } from 'Global/SocketService/SocketService';
import { fetchWithAuth } from 'Global/utils/Auth';

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


  const handleSubmit = async ({ gitUrl, sshKey }) => {
    try {
      const response = await fetchWithAuth('/api/v1/admin/modules', {
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
            <ModuleBadge key={mod.id} mod={mod} />
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
