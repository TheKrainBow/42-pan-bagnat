const MODULE_SESSION_PATH = '/_pb/session';

export const exchangeModuleSession = (origin, token) =>
  new Promise((resolve, reject) => {
    if (!origin || !token) {
      reject(new Error('missing origin or token'));
      return;
    }
    const iframe = document.createElement('iframe');
    iframe.style.position = 'absolute';
    iframe.style.width = '0';
    iframe.style.height = '0';
    iframe.style.border = '0';
    iframe.style.visibility = 'hidden';
    iframe.setAttribute('aria-hidden', 'true');
    iframe.src = `${origin}${MODULE_SESSION_PATH}?token=${encodeURIComponent(token)}`;

    const cleanup = () => {
      window.clearTimeout(timeout);
      iframe.remove();
    };

    const timeout = window.setTimeout(() => {
      cleanup();
      reject(new Error('module session exchange timeout'));
    }, 5000);

    iframe.onload = () => {
      cleanup();
      resolve();
    };
    iframe.onerror = () => {
      cleanup();
      reject(new Error('module session exchange failed'));
    };

    const target = document.body || document.documentElement;
    if (!target) {
      reject(new Error('document not ready'));
      return;
    }
    target.appendChild(iframe);
  });
