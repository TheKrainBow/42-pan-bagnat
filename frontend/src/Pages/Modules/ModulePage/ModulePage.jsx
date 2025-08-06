import React, { useEffect, useState } from 'react';
import { useParams, Navigate } from 'react-router-dom';
import './ModulePage.css';
import Button from 'Global/Button/Button';

export default function ModulePage({ pages }) {
  const { slug } = useParams();
  const [status, setStatus] = useState('loading');
  const [retryKey, setRetryKey] = useState(0); // to force iframe reload

  const page = pages.find((p) => p.slug === slug);

  useEffect(() => {
    if (pages.length === 0) return;

    const newPage = pages.find((p) => p.slug === slug);
    if (newPage) {
      setStatus('loading');
      setRetryKey(k => k + 1); // force iframe reload
    }
  }, [slug, pages]);

  useEffect(() => {
    setRetryKey(0);
  }, [slug]);
  useEffect(() => {
    if (!page) return;

    const iframe = document.getElementById('moduleIframe');
    // if (iframe) {
    //   iframe.src = 'about:blank'; // wipe iframe instantly
    // }
    setStatus('loading');

    if (!iframe) return;

    const timeout = setTimeout(() => {
      setStatus('error');
    }, 8000); // fail after 8s

    iframe.onload = () => {
      clearTimeout(timeout);

      try {
        // Avoid security errors on cross-origin iframes
        const iframeDoc = iframe.contentWindow.document;
        if (!iframeDoc || iframeDoc.body.innerHTML.trim() === '') {
          setStatus('error');
          return;
        }

        const basePath = iframe.src.split('/module-page')[0] + `/module-page/${page.slug}/`;
        const baseTag = document.createElement('base');
        baseTag.setAttribute('href', basePath);
        iframeDoc.head.appendChild(baseTag);

        const links = iframeDoc.querySelectorAll('a');
        links.forEach((link) => {
          const href = link.getAttribute('href');
          if (href && !href.startsWith('http')) {
            link.setAttribute('href', basePath + href);
          }
        });

        setStatus('ready');
      } catch (e) {
        setStatus('error');
      }
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
        src={`${window.location.origin}/module-page/${page.slug}/index.html`}
        title={page.slug}
        frameBorder="0"
        className="module-iframe"
        style={{ display: status === 'ready' ? 'block' : 'none' }}
      />
    </div>
  );
}
