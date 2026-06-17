export const DEFAULT_SIDEBAR_PREFS = { order: [], hidden: {} };

export function loadSidebarPrefs(login) {
  if (!login) return DEFAULT_SIDEBAR_PREFS;
  try {
    const raw = localStorage.getItem(`pb:sidebar:${login}`);
    return raw ? JSON.parse(raw) : DEFAULT_SIDEBAR_PREFS;
  } catch {
    return DEFAULT_SIDEBAR_PREFS;
  }
}

export function getOrderedSidebarPages(pages, prefs) {
  const order = Array.isArray(prefs?.order) ? prefs.order : [];
  const hidden = prefs?.hidden || {};
  const bySlug = new Map(pages.map(p => [p.slug, p]));
  const ordered = [];

  for (const slug of order) {
    if (bySlug.has(slug)) {
      ordered.push(bySlug.get(slug));
      bySlug.delete(slug);
    }
  }

  for (const page of pages) {
    if (!ordered.find(x => x.slug === page.slug)) {
      ordered.push(page);
    }
  }

  return ordered.map(page => ({ ...page, _hidden: !!hidden[page.slug] }));
}

export function getVisibleSidebarPages(pages, prefs) {
  return getOrderedSidebarPages(pages, prefs).filter(page => !page._hidden);
}
