// src/colorUtils.js

const LUMINANCE_THRESHOLD = 4.5;

function lin(c) {
  const v = c / 255;
  return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
}

function getLuminance(r, g, b) {
  return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
}

function hexToRgb(hex) {
  const bigint = parseInt(hex, 16);
  return [(bigint >> 16) & 255, (bigint >> 8) & 255, bigint & 255];
}

function contrastRatio(L1, L2) {
  const lighter = Math.max(L1, L2);
  const darker = Math.min(L1, L2);
  return (lighter + 0.05) / (darker + 0.05);
}

function adjustHex(hex, amt) {
  return hex
    .match(/.{2}/g)
    .map(pair => {
      const v = Math.max(0, Math.min(255, parseInt(pair, 16) + amt));
      const s = v.toString(16);
      return s.length === 1 ? '0' + s : s;
    })
    .join('');
}

export function getReadableStyles(rawHex) {
  let hex = rawHex.replace(/^#|0x/, '');
  let [r, g, b] = hexToRgb(hex);
  let bgLum = getLuminance(r, g, b);

  let whiteRatio = contrastRatio(1.0, bgLum);
  let blackRatio = contrastRatio(bgLum, 0.0);
  let textColor = whiteRatio > blackRatio ? '#ffffff' : '#000000';
  let bestRatio = Math.max(whiteRatio, blackRatio);
  let tries = 0;

  while (bestRatio < LUMINANCE_THRESHOLD && tries++ < 5) {
    hex = adjustHex(hex, whiteRatio > blackRatio ? -20 : +20);
    [r, g, b] = hexToRgb(hex);
    bgLum = getLuminance(r, g, b);
    whiteRatio = contrastRatio(1.0, bgLum);
    blackRatio = contrastRatio(bgLum, 0.0);
    if (whiteRatio > blackRatio) {
      textColor = '#ffffff';
      bestRatio = whiteRatio;
    } else {
      textColor = '#000000';
      bestRatio = blackRatio;
    }
  }

  return { backgroundColor: `#${hex}`, color: textColor };
}
