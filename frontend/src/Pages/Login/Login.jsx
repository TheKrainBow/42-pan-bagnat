// src/Pages/Login.jsx
import React, { useEffect, useMemo, useRef } from "react";
import "./Login.css";
import LoginCard from "./LoginCard";
import { getModulesDomain, parseModuleURL } from "../../utils/modules";
import { exchangeModuleSession } from "../../utils/moduleSession";

const modulesBaseDomain = getModulesDomain().toLowerCase();

const isSafeRedirectTarget = (value) => {
  if (!value) return false;
  if (value.startsWith('/')) return true;
  try {
    const url = new URL(value);
    const host = url.hostname.toLowerCase();
    const currentHost = window.location.hostname.toLowerCase();
    if (host === currentHost) {
      return true;
    }
    if (modulesBaseDomain && (host === modulesBaseDomain || host.endsWith(`.${modulesBaseDomain}`))) {
      return true;
    }
  } catch {
    return false;
  }
  return false;
};

export default function LoginPage() {
  const canvasRef = useRef(null);
  const nextParam = useMemo(() => {
    const params = new URLSearchParams(window.location.search);
    return params.get('next') || '';
  }, []);

  // sim state
  const particles = useRef([]);
  const mouse = useRef({ x: null, y: null, down: false });

  // cached assets
  const logoImgRef = useRef(null);
  const offCanvasRef = useRef(null);
  const offCtxRef = useRef(null);
  const offDataRef = useRef(null); // Uint8ClampedArray of logo colors
  const maskAreaRef = useRef(0);

  // consts
  // Particle counts (auto-sized by logo mask)
  const particlesPer10kPx = 13;
  const minDrawParticles = 120;
  const maxDrawParticles = 1200;
  const nomadParticles = 60;

  // Particle rendering + links
  const maxDist = 70;
  const maxSize = 10;
  const neighborCap = 10;
  const disableLinks = false;
  const skipColorCheckForLinks = false;
  const linkColorThreshold = 55;

  // Mouse interaction
  const mouseRepelRadius = 100;
  const mouseRepelStrength = 0.8;
  const mouseFearMultiplier = 8;

  // Movement / damping
  const returnStrength = 0.05;
  const idleReturnBoost = 1.5;
  const returnBoost = 3;
  const homeFollowRate = 0.08;
  const homeDistanceRamp = 220;
  const homeAccelFloor = 0.25;
  const velocityDamping = 0.96;
  const snapDistance = 0.6;
  const snapVelocity = 0.03;

  // Logo layout + sizing
  const targetLongest = 800;
  const logoVerticalOffset = -0;
  const logoHorizontalOffset = -450;
  const backgroundImagePath = "/icons/panbagnat_transparent.png";

  // Particle colors
  const overwriteColor = ""; // empty or #rgb
  const hasOverwriteColor = overwriteColor.trim().length > 0;

  // canvas dims / layout
  const dims = useRef({
    cssW: 0,
    cssH: 0,
    dw: 0,
    dh: 0,
    dpr: 1,
    offsetX: 0,
    topY: 0,
    bottomY: 0,
    logoWidth: 0,
    logoHeight: 0,
  });

  // grid for neighbor pruning
  const grid = useRef({
    cell: maxDist + maxSize,
    cols: 0,
    rows: 0,
    buckets: [],
  });
  const linkCacheRef = useRef(new Float32Array(0));

  useEffect(() => {
    let raf = 0;
    let resizeRaf = 0;

    const canvas = canvasRef.current;
    const ctx = canvas.getContext("2d", { willReadFrequently: true });

    const logoImg = new Image();
    logoImg.src = backgroundImagePath;
    logoImgRef.current = logoImg;

    const off = document.createElement("canvas");
    const offCtx = off.getContext("2d", { willReadFrequently: true });
    offCanvasRef.current = off;
    offCtxRef.current = offCtx;

    function setCanvasSize(skipRAF = false) {
      const doResize = () => {
        const dpr = Math.min(window.devicePixelRatio || 1, 2);
        const cssW = window.innerWidth;
        const cssH = window.innerHeight;
        canvas.style.width = cssW + "px";
        canvas.style.height = cssH + "px";
        canvas.width = Math.floor(cssW * dpr);
        canvas.height = Math.floor(cssH * dpr);
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

        // logo keeps aspect ratio, canvas fills viewport
        const img = logoImgRef.current;
        const baseW = img?.naturalWidth || img?.width || 1;
        const baseH = img?.naturalHeight || img?.height || 1;
        const longest = Math.max(baseW, baseH) || 1;
        const ratioScale = targetLongest / longest;
        const logoWidth = baseW * ratioScale;
        const logoHeight = baseH * ratioScale;

        const offsetX = (cssW - logoWidth) / 2 + logoHorizontalOffset;
        const topY = (cssH - logoHeight) / 2 + logoVerticalOffset;
        const bottomY = topY + logoHeight;

        off.width = logoWidth;
        off.height = logoHeight;

        dims.current = {
          cssW,
          cssH,
          dw: canvas.width,
          dh: canvas.height,
          dpr,
          offsetX,
          topY,
          bottomY,
          logoWidth,
          logoHeight,
        };

        if (logoImgRef.current?.complete) {
          buildOffscreenAndCaches();
          initParticles(); // will ensure no particle starts in a wall
        }
      };

      if (skipRAF) doResize();
      else {
        if (resizeRaf) cancelAnimationFrame(resizeRaf);
        resizeRaf = requestAnimationFrame(doResize);
      }
    }

    function buildOffscreenAndCaches() {
      const { logoWidth, logoHeight } = dims.current;
      const offCtx = offCtxRef.current;
      if (!logoWidth || !logoHeight) return;

      offCtx.clearRect(0, 0, logoWidth, logoHeight);
      offCtx.drawImage(logoImgRef.current, 0, 0, logoWidth, logoHeight);

      // cache pixels
      const imageData = offCtx.getImageData(0, 0, logoWidth, logoHeight).data;
      offDataRef.current = imageData;
      let maskArea = 0;
      for (let i = 3; i < imageData.length; i += 4) {
        if (imageData[i] > 200) maskArea += 1;
      }
      maskAreaRef.current = maskArea;

    }

    function rand(min, max) {
      return min + Math.random() * (max - min);
    }

    // --- Pixel helpers on the logo mask ---
    function sampleIndex(xCss, yCss) {
      const { offsetX, topY, logoWidth, logoHeight } = dims.current;
      let xi = Math.floor(xCss - offsetX);
      let yi = Math.floor(yCss - topY);
      if (xi < 0 || yi < 0 || xi >= logoWidth || yi >= logoHeight) return -1;
      return ((yi * logoWidth + xi) << 2);
    }

    function isWall(xCss, yCss) {
      const idx = sampleIndex(xCss, yCss);
      if (idx < 0) return true; // treat out-of-bounds as wall
      const data = offDataRef.current;
      const r = data[idx], g = data[idx + 1], b = data[idx + 2], a = data[idx + 3];
      return (a != 255);
    }

    function getLogoColorComponents(xCss, yCss) {
      const idx = sampleIndex(xCss, yCss);
      if (idx < 0) return null;
      const data = offDataRef.current;
      if (!data) return null;
      const a = data[idx + 3];
      if (!a) return null;
      return { r: data[idx], g: data[idx + 1], b: data[idx + 2], a: a / 255 };
    }

    function parseHexColor(value) {
      const match = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(value.trim());
      if (!match) return null;
      let hex = match[1];
      if (hex.length === 3) {
        hex = hex.split("").map((c) => c + c).join("");
      }
      const intVal = parseInt(hex, 16);
      return {
        r: (intVal >> 16) & 255,
        g: (intVal >> 8) & 255,
        b: intVal & 255,
        a: 1,
      };
    }

    const overwriteRgb = hasOverwriteColor ? parseHexColor(overwriteColor) : null;

    function resolveParticleColor(p) {
      if (p.isNomad) return { r: 255, g: 255, b: 255, a: 1 };
      if (overwriteRgb) return overwriteRgb;
      return getLogoColorComponents(p.originX, p.originY) || { r: 255, g: 255, b: 255, a: 1 };
    }

    function linkCrossesWall(ax, ay, bx, by) {
      const aInWall = isWall(ax, ay);
      const bInWall = isWall(bx, by);
      if (aInWall || bInWall) return false;
      const dx = bx - ax;
      const dy = by - ay;
      const distance = Math.hypot(dx, dy);
      const samples = Math.max(6, Math.ceil(distance / 6));
      for (let step = 1; step < samples; step++) {
        const t = step / samples;
        const x = ax + dx * t;
        const y = ay + dy * t;
        if (isWall(x, y)) return true;
      }
      return false;
    }

    // Ensure particles never start inside a wall
    function initParticles() {
      const { offsetX, topY, bottomY, logoWidth, cssW, cssH } = dims.current;
      const logoHeight = bottomY - topY;

      const maskArea = maskAreaRef.current;
      const autoDraw = maskArea > 0
        ? Math.round((maskArea / 10000) * particlesPer10kPx)
        : minDrawParticles;
      const drawCount = Math.max(minDrawParticles, Math.min(maxDrawParticles, autoDraw));
      const nomadCount = Math.max(0, nomadParticles);

      const scaleToTarget = logoWidth / targetLongest;
      const baseCellSize = 28;
      const colTargetSize = Math.max(12, baseCellSize * scaleToTarget);
      const rowTargetSize = Math.max(12, baseCellSize * scaleToTarget);
      const cols = Math.max(1, Math.floor(logoWidth / colTargetSize));
      const rows = Math.max(1, Math.floor(logoHeight / rowTargetSize));
      const colWidth = logoWidth / cols || 1;
      const rowHeight = logoHeight / rows || 1;
      const counts = new Array(cols * rows).fill(0);

      const pickCellIndex = (x, y) => {
        const relX = x - offsetX;
        const relY = y - topY;
        if (relX < 0 || relY < 0 || relX >= logoWidth || relY >= logoHeight) return -1;
        const cx = Math.min(cols - 1, Math.max(0, Math.floor(relX / colWidth)));
        const cy = Math.min(rows - 1, Math.max(0, Math.floor(relY / rowHeight)));
        return cy * cols + cx;
      };

      const cellSeeds = [];
      for (let cy = 0; cy < rows; cy++) {
        for (let cx = 0; cx < cols; cx++) {
          const idx = cy * cols + cx;
          const baseX = offsetX + cx * colWidth;
          const baseY = topY + cy * rowHeight;
          let found = false;
          for (let tries = 0; tries < 80; tries++) {
            const sx = baseX + rand(0, colWidth);
            const sy = baseY + rand(0, rowHeight);
            if (!isWall(sx, sy)) {
              cellSeeds.push({ x: sx, y: sy, idx });
              found = true;
              break;
            }
          }
          if (!found) {
            const centerX = baseX + colWidth / 2;
            const centerY = baseY + rowHeight / 2;
            if (!isWall(centerX, centerY)) {
              cellSeeds.push({ x: centerX, y: centerY, idx });
            }
          }
        }
      }

      const primarySeeds = (() => {
        if (cellSeeds.length <= drawCount) return cellSeeds.slice();
        const seeds = [];
        const step = cellSeeds.length / drawCount;
        for (let k = 0; k < drawCount; k++) {
          const start = Math.floor(k * step);
          const end = Math.max(start, Math.floor((k + 1) * step) - 1);
          const pick = start + Math.floor(Math.random() * (end - start + 1));
          seeds.push(cellSeeds[pick]);
        }
        return seeds;
      })();

      const balancedHome = () => {
        let best = null;
        for (let tries = 0; tries < 400; tries++) {
          const x = offsetX + rand(0, logoWidth);
          const y = rand(topY, bottomY);
          if (isWall(x, y)) continue;
          const idx = pickCellIndex(x, y);
          if (idx < 0) continue;
          const load = counts[idx];
          if (!best || load < best.load || (load === best.load && Math.random() < 0.5)) {
            best = { x, y, idx, load };
            if (load === 0 && tries > 40) break;
          }
        }
        if (!best) {
          const x = offsetX + logoWidth * 0.5;
          const y = (topY + bottomY) * 0.5;
          const idx = pickCellIndex(x, y);
          best = { x, y, idx };
        }
        if (best.idx >= 0) counts[best.idx] += 1;
        return best;
      };

      const totalParticles = drawCount + nomadCount;
      const arr = new Array(totalParticles);
      let seedCursor = 0;
      let cursor = 0;

      const makeParticle = ({ isNomad, originX, originY }) => {
        const spawnX = rand(0, cssW);
        const spawnY = rand(0, cssH);

        const driftAngle = rand(0, Math.PI * 2);
        const drift = rand(0.15, 0.3);
        const baseVX = Math.cos(driftAngle) * drift;
        const baseVY = Math.sin(driftAngle) * drift;
        const vx = isNomad ? baseVX : rand(-0.2, 0.2);
        const vy = isNomad ? baseVY : rand(-0.2, 0.2);
        const speed = isNomad ? 1 : rand(0.8, 1.2);
        const size = rand(0, maxSize);
        return {
          x: spawnX,
          y: spawnY,
          originX,
          originY,
          returnX: originX,
          returnY: originY,
          vx,
          vy,
          speed,
          size,
          baseVX,
          baseVY,
          isNomad,
          feared: false,
          returning: !isNomad,
          distHome: 0,
        };
      };

      for (let k = 0; k < drawCount; k++) {
        const home = seedCursor < primarySeeds.length ? primarySeeds[seedCursor++] : balancedHome();
        const { x, y, idx } = home;
        if (seedCursor <= primarySeeds.length && idx >= 0) counts[idx] += 1;
        arr[cursor++] = makeParticle({ isNomad: false, originX: x, originY: y });
      }

      for (let k = 0; k < nomadCount; k++) {
        const originX = rand(0, cssW);
        const originY = rand(0, cssH);
        arr[cursor++] = makeParticle({ isNomad: true, originX, originY });
      }
      particles.current = arr;
    }

    function rebuildGrid() {
      const { cssW, cssH } = dims.current;
      const cell = grid.current.cell;
      grid.current.cols = Math.ceil(cssW / cell);
      grid.current.rows = Math.ceil(cssH / cell);
      const total = grid.current.cols * grid.current.rows;
      grid.current.buckets = new Array(total);
      for (let i = 0; i < total; i++) grid.current.buckets[i] = [];
    }

    function insertIntoGrid() {
      const buckets = grid.current.buckets;
      for (let i = 0; i < buckets.length; i++) buckets[i].length = 0;
      const cell = grid.current.cell;
      const { cols, rows } = grid.current;

      particles.current.forEach((p, idx) => {
        const cx = Math.max(0, Math.min(cols - 1, Math.floor(p.x / cell)));
        const cy = Math.max(0, Math.min(rows - 1, Math.floor(p.y / cell)));
        const bucketIndex = cy * cols + cx;
        buckets[bucketIndex].push(idx);
      });
    }

    function getNeighbors(idx) {
      const list = [];
      const cell = grid.current.cell;
      const { cols, rows } = grid.current;
      const buckets = grid.current.buckets;
      const p = particles.current[idx];
      const cx = Math.max(0, Math.min(cols - 1, Math.floor(p.x / cell)));
      const cy = Math.max(0, Math.min(rows - 1, Math.floor(p.y / cell)));
      for (let gy = cy - 1; gy <= cy + 1; gy++) {
        if (gy < 0 || gy >= rows) continue;
        for (let gx = cx - 1; gx <= cx + 1; gx++) {
          if (gx < 0 || gx >= cols) continue;
          const bucket = buckets[gy * cols + gx];
          if (bucket.length) list.push(...bucket);
        }
      }
      return list;
    }

    function animate() {
      const { cssW: width, cssH: height, offsetX, topY, bottomY, logoWidth } = dims.current;
      const wrapCoord = (value, limit) => {
        if (limit <= 0) return value;
        return ((value % limit) + limit) % limit;
      };
      const shortestComponent = (delta, limit) => {
        if (limit <= 0) return delta;
        const half = limit / 2;
        if (delta > half) return delta - limit;
        if (delta < -half) return delta + limit;
        return delta;
      };
      const mouseX = mouse.current.x;
      const mouseY = mouse.current.y;
      const hasMouse = mouseX !== null && mouseY !== null;
      const mouseAttract = hasMouse && mouse.current.down;

      // clear with gradient background
      const t = performance.now() * 0.0005;
      const gradient = ctx.createLinearGradient(0, height * Math.sin(t), width, height * Math.cos(t));
      gradient.addColorStop(0, "#173D7A");
      gradient.addColorStop(0.6, "rgba(59, 111, 146, 1)");
      gradient.addColorStop(1, "#00B1B3");
      ctx.fillStyle = gradient;
      ctx.fillRect(0, 0, width, height);

      // update + bounce
      for (const p of particles.current) {
        if (p.isNomad && !p.feared) {
          p.vx = p.vx * 0.7 + p.baseVX * 0.3;
          p.vy = p.vy * 0.7 + p.baseVY * 0.3;
        }

        const stepX = p.vx * p.speed;
        const stepY = p.vy * p.speed;
        const distMousePre = hasMouse ? Math.hypot(p.x - mouseX, p.y - mouseY) : Infinity;
        const escaping = hasMouse && distMousePre < mouseRepelRadius && !mouseAttract;
        const wasInWall = isWall(p.x, p.y);
        const allowThroughWall = mouseAttract || escaping || wasInWall || p.returning || p.isNomad;

        // predict next pos
        let nx = p.x + stepX;
        let ny = p.y + stepY;

        // bounce on black: test axis separately to reflect the right component
        const hitDiag = isWall(nx, ny);
        if (hitDiag && !allowThroughWall) {
          const hitX = isWall(p.x + stepX, p.y);
          const hitY = isWall(p.x, p.y + stepY);

          if (hitX) p.vx = -p.vx;
          if (hitY) p.vy = -p.vy;

          // recompute after reflection and nudge a bit to avoid sticking
          nx = p.x + p.vx * p.speed;
          ny = p.y + p.vy * p.speed;

        }

        nx = wrapCoord(nx, width);
        ny = wrapCoord(ny, height);

        if (escaping && hitDiag) {
          // allow slight overshoot by nudging further away from mouse direction
          const dirX = mouseX !== null ? nx - mouseX : 0;
          const dirY = mouseY !== null ? ny - mouseY : 0;
          const mag = Math.hypot(dirX, dirY) || 1;
          nx += (dirX / mag) * 2;
          ny += (dirY / mag) * 2;
          nx = wrapCoord(nx, width);
          ny = wrapCoord(ny, height);
        }

        p.x = nx;
        p.y = ny;

        const distMouse = hasMouse ? Math.hypot(p.x - mouseX, p.y - mouseY) : Infinity;
        const mouseNear = distMouse < mouseRepelRadius;

        if (mouseNear) {
          if (!p.feared) {
            p.feared = true;
            if (!p.isNomad) {
              p.returning = false;
              p.returnX = p.originX;
              p.returnY = p.originY;
            }
          }
          const safeDist = Math.max(distMouse, 0.0001);
          let influence = mouseRepelStrength * mouseFearMultiplier * (1 - safeDist / mouseRepelRadius);
          if (mouseAttract) {
            const slowFactor = Math.max(0.15, safeDist / mouseRepelRadius);
            influence *= slowFactor;
            p.vx += ((mouseX - p.x) / safeDist) * influence;
            p.vy += ((mouseY - p.y) / safeDist) * influence;
          } else {
            p.vx += ((p.x - mouseX) / safeDist) * influence;
            p.vy += ((p.y - mouseY) / safeDist) * influence;
          }
        } else if (p.feared) {
          p.feared = false;
          if (!p.isNomad) p.returning = true;
        }

        let distHome = 0;
        if (!p.isNomad && !mouseAttract) {
          const targetX = p.returning ? p.returnX : p.originX;
          const targetY = p.returning ? p.returnY : p.originY;
          const homeVecX = shortestComponent(targetX - p.x, width);
          const homeVecY = shortestComponent(targetY - p.y, height);
          distHome = Math.hypot(homeVecX, homeVecY);
          const homeDirX = distHome ? homeVecX / distHome : 0;
          const homeDirY = distHome ? homeVecY / distHome : 0;
          const ramp = Math.max(homeAccelFloor, Math.min(1.5, distHome / homeDistanceRamp));
          const homeAccelBase = (p.returning ? returnStrength * returnBoost : returnStrength * idleReturnBoost) * ramp;
          if (!p.feared && homeAccelBase > 0) {
            p.vx += homeDirX * homeAccelBase;
            p.vy += homeDirY * homeAccelBase;
          }

          if (!p.feared && !p.returning) {
            p.originX += (p.x - p.originX) * homeFollowRate;
            p.originY += (p.y - p.originY) * homeFollowRate;
          }

          if (p.returning && distHome < snapDistance) {
            p.returning = false;
            p.originX = targetX;
            p.originY = targetY;
            p.vx = 0;
            p.vy = 0;
            p.x = targetX;
            p.y = targetY;
          }
        }

        p.vx *= velocityDamping;
        p.vy *= velocityDamping;
        if (p.isNomad && !p.feared) {
          p.vx = p.vx * 0.7 + p.baseVX * 0.3;
          p.vy = p.vy * 0.7 + p.baseVY * 0.3;
        }
        p.distHome = distHome;
      }

      // rebuild spatial grid and insert all
      insertIntoGrid();

      // clamp velocity and snap if close to origin
      for (const p of particles.current) {
        const velMag = Math.hypot(p.vx, p.vy);
        // if (velMag > maxVelocity) {
        //   const scale = maxVelocity / velMag;
        //   p.vx *= scale;
        //   p.vy *= scale;
        // }
        if (!hasMouse && !p.returning && !p.feared && p.distHome < snapDistance && velMag < snapVelocity) {
          p.x = p.originX;
          p.y = p.originY;
          p.vx = 0;
          p.vy = 0;
        }
        p.color = resolveParticleColor(p);
      }

      // draw links (neighbor pruning, squared distances + gradient)
      if (!disableLinks) {
        if (linkCacheRef.current.length !== particles.current.length) {
          linkCacheRef.current = new Float32Array(particles.current.length);
        }
        const maxLinksSqCache = linkCacheRef.current;
        for (let i = 0; i < particles.current.length; i++) {
          const p = particles.current[i];
          maxLinksSqCache[i] = (maxDist + p.size) * (maxDist + p.size);
        }

        const colorThresholdSq = linkColorThreshold * linkColorThreshold;
        for (let i = 0; i < particles.current.length; i++) {
          const a = particles.current[i];
          const neighbors = getNeighbors(i);

          let drawn = 0;
          const aMax = maxLinksSqCache[i];

          for (const j of neighbors) {
            if (j <= i) continue;
            const b = particles.current[j];

            const bMax = maxLinksSqCache[j];
            const localMaxSq = Math.min(
              (maxDist + a.size + b.size) * (maxDist + a.size + b.size),
              aMax + bMax
            );

            const dx = b.x - a.x;
            const dy = b.y - a.y;
            const distSq = dx * dx + dy * dy;
            if (distSq >= localMaxSq) continue;
            const colorA = a.color || resolveParticleColor(a);
            const colorB = b.color || resolveParticleColor(b);
            const dr = colorA.r - colorB.r;
            const dg = colorA.g - colorB.g;
            const db = colorA.b - colorB.b;
            if (!skipColorCheckForLinks) {
              const colorDistSq = dr * dr + dg * dg + db * db;
              if (colorDistSq > colorThresholdSq) continue;
            }
            const ratio = distSq / localMaxSq; // squared ratio
            const alpha = (1 - ratio) / 2 + 0.35;

            if (linkCrossesWall(a.x, a.y, b.x, b.y)) continue;

            const mixR = Math.round((colorA.r + colorB.r) / 2);
            const mixG = Math.round((colorA.g + colorB.g) / 2);
            const mixB = Math.round((colorA.b + colorB.b) / 2);
            ctx.strokeStyle = `rgba(${mixR}, ${mixG}, ${mixB}, ${alpha})`;
            ctx.lineWidth = alpha;
            ctx.beginPath();
            ctx.moveTo(a.x, a.y);
            ctx.lineTo(b.x, b.y);
            ctx.stroke();

            drawn++;
            if (drawn >= neighborCap) break;
          }
        }
      }

      for (const p of particles.current) {
        const color = p.color || resolveParticleColor(p);
        ctx.fillStyle = `rgba(${color.r}, ${color.g}, ${color.b}, ${color.a})`;
        const radius = 1.1 + (p.size / maxSize) * 0.9;
        ctx.beginPath();
        ctx.arc(p.x, p.y, radius, 0, Math.PI * 2);
        ctx.fill();
      }

      raf = requestAnimationFrame(animate);
    }

    // init on image load
    logoImg.onload = () => {
      setCanvasSize(true);
      rebuildGrid();
      buildOffscreenAndCaches();
      initParticles();
      insertIntoGrid();
      animate();
    };

    // listeners
    setCanvasSize();
    rebuildGrid();

    const onResize = () => {
      setCanvasSize();
      rebuildGrid();
    };
    window.addEventListener("resize", onResize, { passive: true });
    const onMove = (e) => {
      mouse.current.x = e.clientX;
      mouse.current.y = e.clientY;
    };
    window.addEventListener("mousemove", onMove, { passive: true });
    const onDown = (e) => {
      if (e.button === 0) mouse.current.down = true;
    };
    const onUp = (e) => {
      if (e.button === 0) mouse.current.down = false;
    };
    window.addEventListener("mousedown", onDown, { passive: true });
    window.addEventListener("mouseup", onUp, { passive: true });
    const onLeave = () => {
      mouse.current.x = null;
      mouse.current.y = null;
      mouse.current.down = false;
    };
    window.addEventListener("mouseleave", onLeave, { passive: true });

    return () => {
      cancelAnimationFrame(raf);
      cancelAnimationFrame(resizeRaf);
      window.removeEventListener("resize", onResize);
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mousedown", onDown);
      window.removeEventListener("mouseup", onUp);
      window.removeEventListener("mouseleave", onLeave);
    };
  }, []);

  useEffect(() => {
    (async () => {
      const res = await fetch("/api/v1/users/me");
      if (res.status === 200) {
        await maybeWarmModuleSession(nextParam, modulesBaseDomain);
        const target = nextParam && isSafeRedirectTarget(nextParam) ? nextParam : "/me";
        window.location.href = target;
      }
    })();
  }, [nextParam]);

  const handleLogin = () => {
    const target = nextParam
      ? `/auth/42/login?next=${encodeURIComponent(nextParam)}`
      : `/auth/42/login?next=${encodeURIComponent("/me")}`;
    window.location.href = target;
  };

  const handleMagicLink = async (email) => {
    try {
      const res = await fetch("/api/v1/auth/magic-link", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      });
      if (res.ok) {
        alert("Magic link sent to " + email);
      } else {
        throw new Error("Failed to send");
      }
    } catch (err) {
      alert("Error sending magic link: " + err.message);
    }
  };

  return (
    <div className="login-page">
      <canvas ref={canvasRef} className="bg-canvas" />
      <div className="login-layout">
        <div className="logo-stack">
        </div>
        <LoginCard onLogin={handleLogin} />
      </div>
    </div>
  );
}

async function maybeWarmModuleSession(nextParam, modulesDomain) {
  if (!nextParam || !isSafeRedirectTarget(nextParam)) {
    return;
  }
  const moduleInfo = parseModuleURL(nextParam, modulesDomain);
  if (!moduleInfo || !moduleInfo.slug) {
    return;
  }
  try {
    const resp = await fetch(`/api/v1/modules/pages/${moduleInfo.slug}/session`, {
      method: 'POST',
      credentials: 'include',
    });
    if (!resp.ok) return;
    const body = await resp.json();
    if (!body?.token) return;
    await exchangeModuleSession(moduleInfo.origin, body.token);
  } catch (err) {
    console.error('Failed to warm module session', err);
  }
}
