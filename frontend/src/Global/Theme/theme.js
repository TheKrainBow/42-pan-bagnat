// Simple theme manager: persists to localStorage and toggles body class

const STORAGE_KEY = 'theme'; // 'dark' | 'light'

export function getStoredTheme() {
  try {
    const v = localStorage.getItem(STORAGE_KEY);
    return v === 'light' ? 'light' : 'dark';
  } catch {
    return 'dark';
  }
}

export function applyTheme(theme) {
  const t = theme === 'light' ? 'light' : 'dark';
  const targets = [document.body, document.documentElement].filter(Boolean);
  targets.forEach(el => {
    const cls = el.classList;
    cls.remove('theme-dark');
    cls.remove('theme-light');
    cls.add(t === 'light' ? 'theme-light' : 'theme-dark');
  });
  try {
    const meta = document.querySelector('meta[name="theme-color"]');
    if (meta) meta.setAttribute('content', t === 'light' ? '#ffffff' : '#000000');
  } catch {}
}

export function setTheme(theme) {
  const t = theme === 'light' ? 'light' : 'dark';
  try { localStorage.setItem(STORAGE_KEY, t); } catch {}
  applyTheme(t);
  // notify listeners
  try {
    window.dispatchEvent(new CustomEvent('pb:themeChanged', { detail: { theme: t } }));
  } catch {}
}

export function toggleTheme() {
  const cur = getStoredTheme();
  setTheme(cur === 'light' ? 'dark' : 'light');
}

export function initTheme() {
  applyTheme(getStoredTheme());
}
