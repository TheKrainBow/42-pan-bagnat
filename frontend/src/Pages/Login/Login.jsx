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
  const mouse = useRef({ x: null, y: null });

  // cached assets
  const logoImgRef = useRef(null);
  const offCanvasRef = useRef(null);
  const offCtxRef = useRef(null);
  const offDataRef = useRef(null); // Uint8ClampedArray of logo colors
  const blurredLayersRef = useRef([]); // pre-blurred logo canvases

  // anim/reveal
  const revealAnimation = useRef(null);

  // consts
  const revealRadius = 100;
  const numParticles = 170;
  const maxDist = 70;
  const maxSize = 30;
  const neighborCap = 12;
  const steps = 8; // flashlight layers

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
    squareSize: 0,
  });

  // grid for neighbor pruning
  const grid = useRef({
    cell: maxDist + maxSize,
    cols: 0,
    rows: 0,
    buckets: new Map(),
  });

  useEffect(() => {
    let raf = 0;
    let resizeRaf = 0;

    const canvas = canvasRef.current;
    const ctx = canvas.getContext("2d", { willReadFrequently: true });

    const logoImg = new Image();
    logoImg.src = "/icons/panbagnat.png";
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

        // logo square is full height and centered horizontally
        const squareSize = cssH;
        const offsetX = (cssW - squareSize) / 2;
        const topY = 0;
        const bottomY = cssH - 0;

        off.width = squareSize;
        off.height = squareSize;

        dims.current = {
          cssW,
          cssH,
          dw: canvas.width,
          dh: canvas.height,
          dpr,
          offsetX,
          topY,
          bottomY,
          squareSize,
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
      const { squareSize } = dims.current;
      const offCtx = offCtxRef.current;
      if (!squareSize) return;

      offCtx.clearRect(0, 0, squareSize, squareSize);
      offCtx.drawImage(logoImgRef.current, 0, 0, squareSize, squareSize);

      // cache pixels
      offDataRef.current = offCtx.getImageData(0, 0, squareSize, squareSize).data;

      // blurred flashlight layers
      const layers = [];
      for (let i = 0; i < steps; i++) {
        const t = i / (steps - 1);
        const blurPx = 8 * (1 - t);
        const grayPct = (blurPx / 16) * 100;

        const c = document.createElement("canvas");
        c.width = squareSize;
        c.height = squareSize;
        const cctx = c.getContext("2d");
        cctx.filter = `blur(${blurPx}px) grayscale(${grayPct}%)`;
        cctx.drawImage(logoImgRef.current, 0, 0, squareSize, squareSize);
        cctx.filter = "none";
        layers.push(c);
      }
      blurredLayersRef.current = layers;
    }

    function rand(min, max) {
      return min + Math.random() * (max - min);
    }

    // --- Pixel helpers on the logo mask ---
    function sampleIndex(xCss, yCss) {
      const { offsetX, squareSize } = dims.current;
      let xi = (xCss - offsetX) | 0;
      let yi = yCss | 0;
      if (xi < 0 || yi < 0 || xi >= squareSize || yi >= squareSize) return -1;
      return ((yi * squareSize + xi) << 2);
    }

    function isWall(xCss, yCss) {
      const idx = sampleIndex(xCss, yCss);
      if (idx < 0) return true; // treat out-of-bounds as wall
      const data = offDataRef.current;
      const r = data[idx], g = data[idx + 1], b = data[idx + 2], a = data[idx + 3];
      return ((r <= 20 && g <= 20 && b <= 20) || a <= 200);
    }

    function colorAtLogo(xCss, yCss) {
      const idx = sampleIndex(xCss, yCss);
      if (idx < 0) return [72, 60, 60, 255];
      const d = offDataRef.current;
      const r = d[idx], g = d[idx + 1], b = d[idx + 2], a = d[idx + 3];
      // if black (wall), we return fallback color; links can still cross walls
      if (r === 0 && g === 0 && b === 0) return [72, 60, 60, a];
      return [r, g, b, a];
    }

    // Ensure particles never start inside a wall
    function initParticles() {
      const { offsetX, topY, bottomY, squareSize } = dims.current;
      const arr = new Array(numParticles);
      for (let k = 0; k < numParticles; k++) {
        let tries = 0, x, y;
        do {
          x = offsetX + rand(0, squareSize);
          y = rand(topY, bottomY);
          tries++;
        } while (isWall(x, y) && tries < 500);
        // If stubborn, drop it in the center line
        if (isWall(x, y)) {
          x = offsetX + squareSize * 0.5;
          y = (topY + bottomY) * 0.5;
        }
        const vx = rand(-0.5, 0.5);
        const vy = rand(-0.5, 0.5);
        const speed = rand(0.5, 1);
        const size = rand(0, maxSize);
        arr[k] = { x, y, vx, vy, speed, size };
      }
      particles.current = arr;
    }

    function rebuildGrid() {
      const { cssW, cssH } = dims.current;
      const cell = grid.current.cell;
      grid.current.cols = Math.ceil(cssW / cell);
      grid.current.rows = Math.ceil(cssH / cell);
    }

    function insertIntoGrid() {
      grid.current.buckets.clear();
      const cell = grid.current.cell;
      const { cols, rows } = grid.current;

      particles.current.forEach((p, idx) => {
        const cx = Math.max(0, Math.min(cols - 1, Math.floor(p.x / cell)));
        const cy = Math.max(0, Math.min(rows - 1, Math.floor(p.y / cell)));
        const key = `${cx},${cy}`;
        let bucket = grid.current.buckets.get(key);
        if (!bucket) {
          bucket = [];
          grid.current.buckets.set(key, bucket);
        }
        bucket.push(idx);
      });
    }

    function getNeighbors(idx) {
      const list = [];
      const cell = grid.current.cell;
      const { cols, rows } = grid.current;
      const p = particles.current[idx];
      const cx = Math.max(0, Math.min(cols - 1, Math.floor(p.x / cell)));
      const cy = Math.max(0, Math.min(rows - 1, Math.floor(p.y / cell)));
      for (let gy = cy - 1; gy <= cy + 1; gy++) {
        if (gy < 0 || gy >= rows) continue;
        for (let gx = cx - 1; gx <= cx + 1; gx++) {
          if (gx < 0 || gx >= cols) continue;
          const bucket = grid.current.buckets.get(`${gx},${gy}`);
          if (bucket) list.push(...bucket);
        }
      }
      return list;
    }

    function drawFlashlight(ctx) {
      const { cssW: width, cssH: height, offsetX, squareSize } = dims.current;

      let centerX = mouse.current.x ?? width / 2;
      let centerY = mouse.current.y ?? height / 2;

      if (mouse.current.x !== null || revealAnimation.current !== null) {
        const maxRadius = Math.hypot(width, height);
        const currentRadius = revealAnimation.current ?? revealRadius;

        if (revealAnimation.current !== null) {
          revealAnimation.current += 10;
          centerX = width / 2;
          centerY = height / 2;
          if (revealAnimation.current >= maxRadius) {
            revealAnimation.current = null;
          }
        }

        const layers = blurredLayersRef.current;
        for (let i = 0; i < steps; i++) {
          const t = i / (steps - 1);
          const r = currentRadius * (1 - 0.2 * t);

          ctx.save();
          ctx.beginPath();
          ctx.arc(centerX, centerY, r, 0, Math.PI * 2);
          ctx.clip();
          ctx.drawImage(layers[i], offsetX, 0, squareSize, squareSize);
          ctx.restore();
        }
      }
    }

    function animate() {
      const { cssW: width, cssH: height, offsetX, topY, bottomY, squareSize } = dims.current;

      // clear
      ctx.fillStyle = "#000";
      ctx.fillRect(0, 0, width, height);

      // update + bounce
      for (const p of particles.current) {
        const stepX = p.vx * p.speed;
        const stepY = p.vy * p.speed;

        // predict next pos
        let nx = p.x + stepX;
        let ny = p.y + stepY;

        // bounce on black: test axis separately to reflect the right component
        const hitDiag = isWall(nx, ny);
        if (hitDiag) {
          const hitX = isWall(p.x + stepX, p.y);
          const hitY = isWall(p.x, p.y + stepY);

          if (hitX) p.vx = -p.vx;
          if (hitY) p.vy = -p.vy;

          // recompute after reflection and nudge a bit to avoid sticking
          nx = p.x + p.vx * p.speed;
          ny = p.y + p.vy * p.speed;

          if (nx < offsetX) nx = offsetX;
          if (nx > offsetX + squareSize) nx = offsetX + squareSize;
          if (ny < topY) ny = topY;
          if (ny > bottomY) ny = bottomY;
        }

        p.x = nx;
        p.y = ny;
      }

      // rebuild spatial grid and insert all
      insertIntoGrid();

      // draw links (neighbor pruning, squared distances + gradient)
      const maxLinksSqCache = new Float32Array(particles.current.length);
      for (let i = 0; i < particles.current.length; i++) {
        const p = particles.current[i];
        maxLinksSqCache[i] = (maxDist + p.size) * (maxDist + p.size);
      }

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
          const ratio = distSq / localMaxSq; // squared ratio
          const alpha = (1 - ratio) / 2 + 0.35;

          const col1 = colorAtLogo(a.x, a.y);
          const col2 = colorAtLogo(b.x, b.y);

          const grad = ctx.createLinearGradient(a.x, a.y, b.x, b.y);
          grad.addColorStop(0, `rgba(${col1[0]}, ${col1[1]}, ${col1[2]}, ${alpha})`);
          grad.addColorStop(1, `rgba(${col2[0]}, ${col2[1]}, ${col2[2]}, ${alpha})`);

          ctx.strokeStyle = grad;
          ctx.lineWidth = alpha;
          ctx.beginPath();
          ctx.moveTo(a.x, a.y);
          ctx.lineTo(b.x, b.y);
          ctx.stroke();

          drawn++;
          if (drawn >= neighborCap) break;
        }
      }

      // flashlight reveal
      drawFlashlight(ctx);

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
    const onLeave = () => {
      mouse.current.x = null;
      mouse.current.y = null;
    };
    window.addEventListener("mouseleave", onLeave, { passive: true });

    return () => {
      cancelAnimationFrame(raf);
      cancelAnimationFrame(resizeRaf);
      window.removeEventListener("resize", onResize);
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseleave", onLeave);
    };
  }, []);

  useEffect(() => {
    (async () => {
      const res = await fetch("/api/v1/users/me");
      if (res.status === 200) {
        await maybeWarmModuleSession(nextParam, modulesBaseDomain);
        const target = nextParam && isSafeRedirectTarget(nextParam) ? nextParam : "/modules";
        window.location.href = target;
      }
    })();
  }, [nextParam]);

  const handleLogin = () => {
    revealAnimation.current = revealRadius;
    const target = nextParam ? `/auth/42/login?next=${encodeURIComponent(nextParam)}` : "/auth/42/login";
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
      <LoginCard onLogin={handleLogin} onMagicLink={handleMagicLink} />
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
