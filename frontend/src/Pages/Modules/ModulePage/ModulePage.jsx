import React, { useEffect, useMemo, useState } from 'react';
import { useParams, Navigate } from 'react-router-dom';
import './ModulePage.css';
import Button from 'Global/Button/Button';
import { getModulesDomain, getModulesProtocol } from '../../../utils/modules';
import { exchangeModuleSession } from '../../../utils/moduleSession';
import { loadSidebarPrefs, getVisibleSidebarPages } from '../../../utils/sidebarPrefs';
import { getModulePageMode } from '../../../utils/modulePageMode';

export default function ModulePage({ pages, user }) {
  const { slug } = useParams();
  const [status, setStatus] = useState('loading');
  const [retryKey, setRetryKey] = useState(0);
  const [authReady, setAuthReady] = useState(false);
  const [prefs, setPrefs] = useState(() => loadSidebarPrefs(user?.ft_login));

  const visiblePages = useMemo(() => getVisibleSidebarPages(pages, prefs), [pages, prefs]);
  const page = pages.find((p) => p.slug === slug);
  const pageMode = getModulePageMode(page);
  const modulesDomain = useMemo(() => getModulesDomain(), []);
  const modulesProtocol = useMemo(() => getModulesProtocol(modulesDomain), [modulesDomain]);
  const moduleOrigin = page && pageMode !== 'page_only' ? `${modulesProtocol}://${page.slug}.${modulesDomain}` : '';
  const iframeSrc = moduleOrigin ? `${moduleOrigin}/` : '';
  const externalUrl = page ? `${modulesProtocol}://${page.slug}.${modulesDomain}` : '';

  useEffect(() => {
    if (user?.ft_login) {
      setPrefs(loadSidebarPrefs(user.ft_login));
    }
  }, [user?.ft_login]);

  useEffect(() => {
    function onPrefsChanged(e) {
      if (!user?.ft_login) return;
      if (!e?.detail?.login || e.detail.login === user.ft_login) {
        setPrefs(loadSidebarPrefs(user.ft_login));
      }
    }

    window.addEventListener('pb:prefs:sidebarChanged', onPrefsChanged);
    return () => window.removeEventListener('pb:prefs:sidebarChanged', onPrefsChanged);
  }, [user?.ft_login]);

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
    if (!page || pageMode === 'page_only') return;
    setStatus('loading');
  }, [page, pageMode, retryKey]);

  useEffect(() => {
    if (!page || pageMode === 'page_only') {
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
  }, [page, pageMode, moduleOrigin, retryKey]);

  useEffect(() => {
    if (!page || pageMode === 'page_only' || !authReady) return;
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
  }, [page, pageMode, retryKey, authReady]);

  if (!slug) {
    if (visiblePages.length > 0) {
      return <Navigate to={`/modules/${visiblePages[0].slug}`} replace />;
    }
    return <div className="module-page-placeholder">No accessible modules.</div>;
  }

  if (!page) {
    return <div className="module-page-placeholder">Module not found or access denied.</div>;
  }

  if (pageMode === 'page_only') {
    return (
      <div className="module-page-container">
        <div className="module-page-status module-page-status-page-only">
          <p>Ce module n&apos;est pas disponible en iframe.</p>
          <Button
            label="Acceder au site"
            color="blue"
            href={externalUrl}
            onClick={() => window.location.assign(externalUrl)}
          />
        </div>
      </div>
    );
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
        src={authReady ? iframeSrc : "about:blank"}
        title={page.slug}
        frameBorder="0"
        className="module-iframe"
        referrerPolicy="strict-origin-when-cross-origin"
        style={{ display: status === "ready" ? "block" : "none" }}
      />
    </div>
  );
}
