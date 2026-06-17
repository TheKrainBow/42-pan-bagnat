export function getModulePageMode(page) {
  if (page?.pageOnly || page?.page_only) return 'page_only';
  if (page?.iframeOnly || page?.iframe_only) return 'iframe_only';
  return 'both';
}

export function pageModeToFlags(mode) {
  switch (mode) {
    case 'iframe_only':
      return { iframeOnly: true, pageOnly: false };
    case 'page_only':
      return { iframeOnly: false, pageOnly: true };
    default:
      return { iframeOnly: false, pageOnly: false };
  }
}
