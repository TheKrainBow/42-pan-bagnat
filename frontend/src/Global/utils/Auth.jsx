import { toast } from 'react-toastify';

function parentPath(pathname) {
  const i = pathname.lastIndexOf("/");
  return i > 0 ? pathname.slice(0, i) : "/";
}

export async function fetchWithAuth(url, options = {}) {
  const res = await fetch(url, {
    ...options,
    credentials: "include",
  });

  if (res.status === 401) {
    window.location.href = "/login";
    return null;
  }

  if (res.status === 403) {
    window.location.href = "/login";
    toast.error("Unauthorized")
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

    toastId = toast.error("Page Not found", {
      autoClose: toastMs,
      onClose: handleRedirect,
      onClick: () => {
        toast.dismiss(toastId); // close immediately
        handleRedirect();
      },
    });
    return null;
  }
  return res;
}