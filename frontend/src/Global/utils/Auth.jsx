import { toast } from 'react-toastify';

function stashToastAndRedirect(kind, message, path = '/login') {
  try {
    sessionStorage.setItem('pb:pendingToast', JSON.stringify({
      kind,
      message,
      at: Date.now()
    }));
  } catch {}
  // Optional: show an immediate local toast too (in case the page lingers)
  if (toast && typeof toast[kind] === 'function') {
    toast[kind](message);
  }
  window.location.assign(path); // hard redirect → always works
}

export async function fetchWithAuth(url, options = {}) {
  const res = await fetch(url, { ...options, credentials: 'include' });

  if (res.status === 401) {
    let msg = "Please sign in again";
    try {
      const j = await res.json();
      msg = j?.message || msg;
    } catch {}
    stashToastAndRedirect('error', msg);
    return null;
  }
  if (res.status === 403) {
    let msg = "Forbidden";
    try {
      const j = await res.json();
      msg = j?.message || msg;
    } catch {}
    stashToastAndRedirect('error', msg);
    return null;
  }
  if (res.status === 409 || res.status === 422) {
    const j = await res.json().catch(() => ({}));
    toast.error(j?.message || "Operation not allowed");
    return null;
  }

  if (res.status === 404) {
    const startPath = window.location.pathname;
    const toastMs = 2000;
    let toastId;
    const handleRedirect = () => {
      if (window.location.pathname !== startPath) return;
      window.history.back();
    };
    toastId = toast.error('Page Not found', {
      autoClose: toastMs,
      onClose: handleRedirect,
      onClick: () => {
        toast.dismiss(toastId);
        handleRedirect();
      },
    });
    return null;
  }

  return res;
}