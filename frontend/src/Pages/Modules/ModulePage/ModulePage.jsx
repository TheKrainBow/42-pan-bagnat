import React, { useEffect, useMemo, useState } from 'react';
import { useParams, Navigate } from 'react-router-dom';
import './ModulePage.css';
import Button from 'Global/Button/Button';
import ModuleStatusCard, { WrenchIcon, LockIcon, ExternalLinkIcon, LoadingIcon, AlertIcon } from 'Pages/Modules/Components/ModuleStatusCard/ModuleStatusCard';
import { getModulesDomain, getModulesProtocol } from '../../../utils/modules';
import { exchangeModuleSession } from '../../../utils/moduleSession';
import { getModulePageMode } from '../../../utils/modulePageMode';

export default function ModulePage({ pages }) {
  const { slug } = useParams();
  const [status, setStatus] = useState('loading');
  const [retryKey, setRetryKey] = useState(0);
  const [authReady, setAuthReady] = useState(false);
  const [redirectCountdown, setRedirectCountdown] = useState(3);

  const page = pages.find((p) => p.slug === slug);
  const isRedirection = page?.kind === 'redirection';
  const pageMode = getModulePageMode(page);
  const modulesDomain = useMemo(() => getModulesDomain(), []);
  const modulesProtocol = useMemo(() => getModulesProtocol(modulesDomain), [modulesDomain]);
  const moduleOrigin = page && !isRedirection && pageMode !== 'page_only' ? `${modulesProtocol}://${page.slug}.${modulesDomain}` : '';
  const iframeSrc = moduleOrigin ? `${moduleOrigin}/` : '';
  const externalUrl = page && !isRedirection ? `${modulesProtocol}://${page.slug}.${modulesDomain}` : '';

  // Redirections aren't iframed at all: the sidebar normally sends the
  // browser straight to target_url and never routes here, but a direct visit
  // to /modules/:slug (bookmark, back button, typed URL) still can — show a
  // short countdown (with a manual skip) instead of bouncing out silently.
  useEffect(() => {
    setRedirectCountdown(3);
  }, [isRedirection, page?.slug]);

  useEffect(() => {
    if (!isRedirection || !page?.target_url) return;
    if (redirectCountdown <= 0) {
      window.location.replace(page.target_url);
      return;
    }
    const timer = setTimeout(() => setRedirectCountdown((s) => s - 1), 1000);
    return () => clearTimeout(timer);
  }, [isRedirection, page, redirectCountdown]);

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
    if (!page || isRedirection || pageMode === 'page_only' || page.module_disabled) return;
    setStatus('loading');
  }, [page, isRedirection, pageMode, retryKey]);

  useEffect(() => {
    if (!page || isRedirection || pageMode === 'page_only' || page.module_disabled) {
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
  }, [page, isRedirection, pageMode, moduleOrigin, retryKey]);

  useEffect(() => {
    if (!page || isRedirection || pageMode === 'page_only' || page.module_disabled || !authReady) return;
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
  }, [page, isRedirection, pageMode, retryKey, authReady]);

  if (!slug) {
    // Land on the dashboard instead of silently auto-opening the first
    // sidebar module: that used to load the module's iframe (and register a
    // usage activity for it) before the user ever chose to open anything.
    return <Navigate to="/dashboard" replace />;
  }

  if (!page) {
    return (
      <div className="module-page-container">
        <ModuleStatusCard
          accent="red"
          icon={<LockIcon />}
          badge="Access restricted"
          title="Module not found or access denied."
          description="You may not have the required role for this page, or it no longer exists."
        />
      </div>
    );
  }

  if (isRedirection) {
    return (
      <div className="module-page-container">
        <ModuleStatusCard
          accent="blue"
          icon={<ExternalLinkIcon />}
          badge="Redirecting"
          title={`Redirecting in ${redirectCountdown}s`}
          description={
            <>
              <span className="status-card-url" title={page.target_url}>{page.target_url}</span>
              This link leaves Pan Bagnat.
            </>
          }
          action={
            <Button
              label="Redirect me now"
              color="blue"
              href={page.target_url}
              onClick={() => window.location.assign(page.target_url)}
            />
          }
        />
      </div>
    );
  }

  if (page.module_disabled) {
    return (
      <div className="module-page-container">
        <ModuleStatusCard
          accent="blue"
          icon={<WrenchIcon />}
          badge="Module status: disabled"
          title="This module is currently disabled by an administrator."
          description="If you were expecting to use it, let them know and they may turn it back on."
        />
      </div>
    );
  }

  if (pageMode === 'page_only') {
    return (
      <div className="module-page-container">
        <ModuleStatusCard
          accent="blue"
          icon={<ExternalLinkIcon />}
          badge="Opens in a new tab"
          title="Ce module n'est pas disponible en iframe."
          action={
            <Button
              label="Acceder au site"
              color="blue"
              href={externalUrl}
              onClick={() => window.location.assign(externalUrl)}
            />
          }
        />
      </div>
    );
  }

  return (
    <div className="module-page-container">
      {status === 'loading' && (
        <ModuleStatusCard
          accent="blue"
          icon={<LoadingIcon />}
          badge="Module status: loading"
          title="This module is currently loading."
          description="Hang tight while we connect to it — this should only take a moment."
        />
      )}
      {status === 'error' && (
        <ModuleStatusCard
          accent="red"
          icon={<AlertIcon />}
          badge="Module status: error"
          title="This module is not accessible right now."
          description="It may be temporarily down, or the connection timed out."
          action={
            <Button label="Retry" color="blue" onClick={() => setRetryKey((k) => k + 1)} />
          }
        />
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
