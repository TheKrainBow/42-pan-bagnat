import React, { useEffect, useMemo, useState } from 'react';
import { useParams, Navigate } from 'react-router-dom';
import './ModulePage.css';
import Button from 'Global/Button/Button';
import { getModulesDomain, getModulesProtocol } from '../../../utils/modules';
import { exchangeModuleSession } from '../../../utils/moduleSession';

export default function ModulePage({ pages }) {
  const { slug } = useParams();
  const [status, setStatus] = useState('loading');
  const [retryKey, setRetryKey] = useState(0);
  const [authReady, setAuthReady] = useState(false);

  const page = pages.find((p) => p.slug === slug);
  const modulesDomain = useMemo(() => getModulesDomain(), []);
  const modulesProtocol = useMemo(() => getModulesProtocol(modulesDomain), [modulesDomain]);
  const moduleOrigin = page ? `${modulesProtocol}://${page.slug}.${modulesDomain}` : '';
  const iframeSrc = moduleOrigin ? `${moduleOrigin}/` : '';

  useEffect(() => {
    if (pages.length === 0) return;
    const newPage = pages.find((p) => p.slug === slug);
    if (newPage) {
      setRetryKey((k) => k + 1);
    }
  }, [slug, pages]);

  useEffect(() => {
    setRetryKey(0);
  }, [slug]);

  useEffect(() => {
    if (!page) return;
    setStatus('loading');
  }, [page, retryKey]);

  useEffect(() => {
    if (!page) {
      setAuthReady(false);
      return;
    }
    if (!page.need_auth) {
      setAuthReady(true);
      return;
    }

    let canceled = false;
    setAuthReady(false);

    const run = async () => {
      try {
        const resp = await fetch(`/api/v1/modules/pages/${page.slug}/session`, {
          method: 'POST',
          credentials: 'include',
        });
        if (!resp.ok) {
          throw new Error(`token request failed with ${resp.status}`);
        }
        const body = await resp.json();
        if (!body?.token) {
          throw new Error('token payload missing');
        }
        if (!moduleOrigin) {
          throw new Error('module origin missing');
        }
        await exchangeModuleSession(moduleOrigin, body.token);
        if (!canceled) {
          setAuthReady(true);
        }
      } catch (err) {
        console.error('Failed to prepare module session', err);
        if (!canceled) {
          setAuthReady(false);
          setStatus('error');
        }
      }
    };

    run();
    return () => {
      canceled = true;
    };
  }, [page, moduleOrigin, retryKey]);

  useEffect(() => {
    if (!page || !authReady) return;
    const iframe = document.getElementById('moduleIframe');
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
  }, [page, retryKey, authReady]);

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
        src={authReady ? iframeSrc : 'about:blank'}
        title={page.slug}
        frameBorder="0"
        className="module-iframe"
        referrerPolicy="strict-origin-when-cross-origin"
        style={{ display: status === 'ready' ? 'block' : 'none' }}
      />
    </div>
  );
}
