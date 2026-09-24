// Fixed design-time resolution for the dashboard canvas. Every viewer sees
// the exact same layout (same Fabric.js JSON) regardless of their window
// size — the canvas element itself is scaled via CSS transform to fit
// whatever width is available, both in the editor and the read-only view.
export const DASHBOARD_WIDTH = 1600;
export const DASHBOARD_HEIGHT = 800;

// Fabric can't resolve CSS custom properties (var(--background)) itself, so
// read the computed value once at canvas-creation time.
export function resolveCssColor(varName, fallback) {
  if (typeof window === 'undefined') return fallback;
  const value = getComputedStyle(document.documentElement).getPropertyValue(varName).trim();
  return value || fallback;
}

export function resolveBaseFontFamily() {
  if (typeof window === 'undefined') return 'sans-serif';
  return getComputedStyle(document.body).fontFamily || 'sans-serif';
}

// <canvas> only ever paints a single static frame of a GIF — drawImage
// doesn't animate — so any GIF drawn onto the Fabric canvas looks frozen.
// Real <img> elements do animate, so both the editor and the read-only view
// overlay one per image object, positioned/sized/rotated to match it exactly,
// instead of relying on the canvas's own (static) rendering of that object.
let overlayKeyCounter = 0;
export function computeImageOverlays(canvas) {
  return canvas.getObjects()
    .filter(o => o.type === 'image' && typeof o.getSrc === 'function' && o.getSrc())
    .map((o) => {
      if (o.__pbOverlayKey == null) o.__pbOverlayKey = ++overlayKeyCounter;
      const center = o.getCenterPoint();
      const width = o.getScaledWidth();
      const height = o.getScaledHeight();
      return {
        key: o.__pbOverlayKey,
        src: o.getSrc(),
        left: center.x - width / 2,
        top: center.y - height / 2,
        width,
        height,
        angle: o.angle || 0,
      };
    });
}
