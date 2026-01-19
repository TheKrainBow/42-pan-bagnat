import React, { useEffect, useMemo, useState } from 'react';
import { useParams, Navigate } from 'react-router-dom';
import './ModulePage.css';
import Button from 'Global/Button/Button';

const getModulesDomain = () => {
  const envValue = (import.meta.env.VITE_MODULES_BASE_DOMAIN || '').trim();
  if (envValue) return envValue;
  return 'modules.127.0.0.1.nip.io';
};

const getModulesProtocol = (domain) => {
  const override = (import.meta.env.VITE_MODULES_PROTOCOL || '').trim().toLowerCase();
  if (override === 'http' || override === 'https') {
    return override;
  }
  if (domain.endsWith('.127.0.0.1.nip.io') || domain.endsWith('.nip.io')) {
    return 'http';
  }
  return window.location.protocol === 'https:' ? 'https' : 'http';
};

export default function ModulePage({ pages }) {
  const { slug } = useParams();
  const [status, setStatus] = useState('loading');
  const [retryKey, setRetryKey] = useState(0);

  const page = pages.find((p) => p.slug === slug);
  const modulesDomain = useMemo(() => getModulesDomain(), []);
  const modulesProtocol = useMemo(() => getModulesProtocol(modulesDomain), [modulesDomain]);
  const iframeSrc = page ? `${modulesProtocol}://${page.slug}.${modulesDomain}/` : '';

  useEffect(() => {
    if (pages.length === 0) return;
    const newPage = pages.find((p) => p.slug === slug);
    if (newPage) {
      setStatus('loading');
      setRetryKey((k) => k + 1);
    }
  }, [slug, pages]);

  useEffect(() => {
    setRetryKey(0);
  }, [slug]);

  useEffect(() => {
    if (!page) return;
    const iframe = document.getElementById('moduleIframe');
    setStatus('loading');
    if (!iframe) return;

    const timeout = setTimeout(() => setStatus('error'), 8000);
    iframe.onload = () => {
      clearTimeout(timeout);
      setStatus('ready');
    };
    iframe.onerror = () => {
      clearTimeout(timeout);
      setStatus('error');
    };
    return () => clearTimeout(timeout);
  }, [page, retryKey]);

  if (!slug) {
    if (pages.length > 0) {
      return <Navigate to={`/modules/${pages[0].slug}`} replace />;
    }
    return <div className="module-page-placeholder">No accessible modules.</div>;
  }

  if (!page && pages.length > 0) {
    return <Navigate to={`/modules/${pages[0].slug}`} replace />;
  }

  if (!page) {
    return <div className="module-page-placeholder">Module not found or access denied.</div>;
  }

  return (
    <div className="module-page-container">
      {status === 'loading' && (
        <div className="module-page-status">
          <p>🔄 Loading module...</p>
        </div>
      )}
      {status === 'error' && (
        <div className="module-page-status error">
          <p>❌ This module is not accessible right now.</p>
          <Button label="Retry" onClick={() => setRetryKey((k) => k + 1)}></Button>
        </div>
      )}
      <iframe
        id="moduleIframe"
        key={`${page.slug}-${retryKey}`}
        src={iframeSrc}
        title={page.slug}
        frameBorder="0"
        className="module-iframe"
        style={{ display: status === 'ready' ? 'block' : 'none' }}
      />
    </div>
  );
}
