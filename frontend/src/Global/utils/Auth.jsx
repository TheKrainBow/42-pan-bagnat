export async function fetchWithAuth(url, options = {}) {
  const res = await fetch(url, {
    ...options,
    credentials: "include", // 👈 ensures cookies are sent
  });

  if (res.status === 401) {
    window.location.href = "/login";
    return null;
  }

  if (res.status === 403 || res.status === 500) {
    window.location.href = "/login";
    toast.error("Unauthorized")
    return null;
  }

  return res;
}