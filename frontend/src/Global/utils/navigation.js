export function openInNewTab(url) {
  const target = String(url || '').trim();
  if (!target) return;
  window.open(target, '_blank', 'noopener,noreferrer');
}

export function isMiddleClick(event) {
  return event?.button === 1;
}
