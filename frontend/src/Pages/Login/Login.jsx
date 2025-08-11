// src/Pages/Login.jsx
import React, { useEffect, useRef } from "react";
import "./Login.css";
import LoginCard from "./LoginCard";

export default function LoginPage() {
  const canvasRef = useRef(null);

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
  const numParticles = 200;
  const maxDist = 50;
  const maxSize = 70; // your random size cap
  const neighborCap = 12; // optional cap per particle
  const steps = 8; // flashlight “ring” steps (cached layers)

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
    squareSize: 0, // equals cssH (logo square side in CSS pixels)
  });

  // grid for neighbor pruning
  const grid = useRef({
    cell: maxDist + maxSize, // safe cell size
    cols: 0,
    rows: 0,
    buckets: new Map(), // key "cx,cy" -> array of indices
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
        const squareSize = cssH;
        const offsetX = (cssW - squareSize) / 2;
        const topY = 220;
        const bottomY = cssH - 220;
        off.width = squareSize;
        off.height = squareSize;
        dims.current = { cssW, cssH, dw: canvas.width, dh: canvas.height, dpr, offsetX, topY, bottomY, squareSize };
        if (logoImgRef.current?.complete) {
          buildOffscreenAndCaches();
          initParticles();
        }
      };
      if (skipRAF) {
        doResize();
      } else {
        if (resizeRaf) cancelAnimationFrame(resizeRaf);
        resizeRaf = requestAnimationFrame(doResize);
      }
    }

    function buildOffscreenAndCaches() {
      const { squareSize } = dims.current;
      const offCtx = offCtxRef.current;

      if (!squareSize) return;
      // draw the logo once into offscreen
      offCtx.clearRect(0, 0, squareSize, squareSize);
      offCtx.drawImage(logoImgRef.current, 0, 0, squareSize, squareSize);

      // cache raw pixel array (no per-frame getImageData)
      const offData = offCtx.getImageData(0, 0, squareSize, squareSize).data;
      offDataRef.current = offData;

      // prebuild blurred layers
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

    function initParticles() {
      const { offsetX, topY, bottomY, squareSize } = dims.current;
      particles.current = Array.from({ length: numParticles }, () => {
        const x = offsetX + rand(0, squareSize);
        const y = rand(topY, bottomY);
        const vx = rand(-0.5, 0.5);
        const vy = rand(-0.5, 0.5);
        const speed = rand(0.5, 1);
        const size = rand(0, maxSize);
        return { x, y, vx, vy, speed, size };
      });
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

    function colorAtLogo(xCss, yCss) {
      // xCss, yCss are CSS-space coordinates
      const { offsetX, squareSize } = dims.current;
      const xi = Math.max(0, Math.min(squareSize - 1, (xCss - offsetX) | 0));
      const yi = Math.max(0, Math.min(squareSize - 1, yCss | 0));
      const idx = ((yi * squareSize + xi) << 2);
      const data = offDataRef.current;
      const r = data[idx], g = data[idx + 1], b = data[idx + 2], a = data[idx + 3];
      // replace black with fallback color (72,60,60)
      if (r === 0 && g === 0 && b === 0) return [72, 60, 60, a];
      return [r, g, b, a];
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

        // reuse cached blurred layers
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
      const { cssW: width, cssH: height, offsetX, topY, bottomY } = dims.current;

      // clear
      ctx.fillStyle = "#000";
      ctx.fillRect(0, 0, width, height);

      // update particles
      for (const p of particles.current) {
        p.x += p.vx * p.speed;
        p.y += p.vy * p.speed;

        if (p.x < offsetX) p.x = offsetX + dims.current.squareSize;
        if (p.x > offsetX + dims.current.squareSize) p.x = offsetX;
        if (p.y < topY) p.y = bottomY;
        if (p.y > bottomY) p.y = topY;
      }

      // rebuild spatial grid and insert all
      insertIntoGrid();

      // draw links with neighbor pruning and squared distances
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
          if (j <= i) continue; // avoid double draw
          const b = particles.current[j];

          // quick reject using each particle’s individual radius
          const bMax = maxLinksSqCache[j];
          const localMaxSq = Math.min(
            (maxDist + a.size + b.size) * (maxDist + a.size + b.size),
            aMax + bMax
          );

          const dx = b.x - a.x;
          const dy = b.y - a.y;
          const distSq = dx * dx + dy * dy;
          if (distSq >= localMaxSq) continue;

          const dist = Math.sqrt(distSq);
          const alpha = (1 - dist / Math.sqrt(localMaxSq)) / 2 + 0.35;

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
          if (drawn >= neighborCap) break; // soft cap per particle
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
        window.location.href = "/modules";
      }
    })();
  }, []);

  const handleLogin = () => {
    revealAnimation.current = revealRadius;
    window.location.href = "/auth/42/login";
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
