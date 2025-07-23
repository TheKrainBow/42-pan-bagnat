import React, { useEffect } from 'react';
import './ModulePage.css';

export default function ModulePage({ page }) {
  useEffect(() => {
    const iframe = document.getElementById('moduleIframe');
    
    if (iframe) {
      iframe.onload = function () {
        console.log('Iframe loaded successfully.');
        const iframeDoc = iframe.contentWindow.document;
        const basePath = iframe.src.split('/module-page')[0] + `/module-page/${page.slug}/`;
        const baseTag = document.createElement('base');
        baseTag.setAttribute('href', basePath);
        iframeDoc.head.appendChild(baseTag);
        const links = iframeDoc.querySelectorAll('a');
        links.forEach((link, index) => {
          const href = link.getAttribute('href');
          if (href && !href.startsWith('http')) {
            link.setAttribute('href', basePath + href);
          }
        });
      };
    } else {
      console.log('Iframe not found!');
    }
  }, [page]);  // Re-run when page.slug changes

  if (!page || !page.slug) {
    return (
      <div className="module-page-placeholder">
        <p>Select a module from the sidebar to get started.</p>
      </div>
    );
  }

  return (
    <div className="module-page-container">
      <iframe
        id="moduleIframe"
        src={`http://localhost/module-page/${page.slug}`}  // Path to the inner website
        title={page.slug}
        frameBorder="0"
        className="module-iframe"
      />
    </div>
  );
}
