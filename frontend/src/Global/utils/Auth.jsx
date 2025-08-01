export async function fetchWithAuth(url, options = {}) {
  const res = await fetch(url, {
    ...options,
    credentials: "include", // 👈 ensures cookies are sent
  });

  if (res.status === 401) {
    window.location.href = "/login";
    return null;
  }

  return res;
}