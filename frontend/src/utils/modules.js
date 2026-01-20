const DEFAULT_MODULES_DOMAIN = 'modules.127.0.0.1.nip.io';

export function getModulesDomain() {
  const envValue = (import.meta.env.VITE_MODULES_BASE_DOMAIN || '').trim();
  return envValue || DEFAULT_MODULES_DOMAIN;
}

export function getModulesProtocol(domain) {
  const override = (import.meta.env.VITE_MODULES_PROTOCOL || '').trim().toLowerCase();
  if (override === 'http' || override === 'https') {
    return override;
  }
  const lowerDomain = domain.toLowerCase();
  if (
    lowerDomain.endsWith('.127.0.0.1.nip.io') ||
    lowerDomain.endsWith('.nip.io') ||
    lowerDomain === 'localhost' ||
    lowerDomain.endsWith('.localhost')
  ) {
    return 'http';
  }
  return window.location.protocol === 'https:' ? 'https' : 'http';
}

export function extractModuleSlugFromHost(host, modulesDomain) {
  if (!host || !modulesDomain) return '';
  const lowerHost = host.toLowerCase();
  const suffix = `.${modulesDomain.toLowerCase()}`;
  if (!lowerHost.endsWith(suffix)) return '';
  const slug = lowerHost.slice(0, lowerHost.length - suffix.length);
  return slug || '';
}

export function parseModuleURL(value, modulesDomain = getModulesDomain()) {
  if (!value) return null;
  try {
    const url = new URL(value);
    const slug = extractModuleSlugFromHost(url.hostname, modulesDomain);
    if (!slug) {
      return null;
    }
    return {
      slug,
      origin: `${url.protocol}//${url.hostname}`,
      url,
    };
  } catch {
    return null;
  }
}
