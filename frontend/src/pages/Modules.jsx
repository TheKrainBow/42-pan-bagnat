// Modules.jsx
import React, { useState, useEffect, useRef, useCallback } from 'react';
import './Modules.css';
import { AppIcon } from '../components/AppIcon';
import { Header } from '../components/Header';
import { Link } from 'react-router-dom';

const Modules = () => {
  const [modules, setModules] = useState([]);
  const [filterQuery, setFilterQuery] = useState('');
  const [debouncedFilter, setDebouncedFilter] = useState('');
  const [nextPage, setNextPage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const loadingRef = useRef(false);
  const scrollContainerRef = useRef(null);
  const isFirst = useRef(true);

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

      const res = await fetch(`http://localhost:8080/api/v1/modules?${params.toString()}`);
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

  return (
    <div className="p-4">
      <Header
        title="Modules"
        value={filterQuery}
        onChange={(e) => setFilterQuery(e.target.value)}
      />
      <div className="modules-container" ref={scrollContainerRef}>
        <div className="modules-grid">
          {modules.map((mod) => (
            <Link key={mod.id} to={`/modules/${mod.id}`} className={`module-card ${mod.status === 'enabled' ? 'active' : 'disabled'}`}>
              <div className="module-icon">
                <AppIcon app={{ icon_url: mod.icon_url, name: mod.name }} fallback="/icons/modules.png" />
              </div>
              <div className="module-content">
                <div className="module-title-row">
                  <strong>{mod.name}</strong>
                  <span className="module-status">{mod.status.toUpperCase()}</span>
                </div>
                <p className="module-description">v{mod.version} • {mod.late_commits} late commits</p>
                <p className="module-updated">Last update: {new Date(mod.last_update).toLocaleDateString()}</p>
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
    </div>
  );
};

export default Modules;
