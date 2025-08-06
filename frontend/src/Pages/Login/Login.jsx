// src/Pages/Login.jsx
import React, { useEffect, useRef } from "react";
import "./Login.css";
import LoginCard from "./LoginCard";

export default function LoginPage() {
  const canvasRef = useRef(null);
  const particles = useRef([]);
  const mouse = useRef({ x: null, y: null });

  useEffect(() => {
    const canvas = canvasRef.current;
    const ctx = canvas.getContext("2d");
    let width, height;
    const numParticles = 500;
    const maxDist = 120;
    const revealRadius = 200;
    const restoreFactor = 0.05;
    let offsetX;

    function randomBetween(min, max) {
      return min + Math.random() * (max - min);
    }

    // prepare offscreen logo
    const logoImg = new Image();
    logoImg.src = "/icons/panbagnat.png";
    const offCanvas = document.createElement("canvas");
    const offCtx = offCanvas.getContext("2d");

    function setup() {
      width = canvas.width = window.innerWidth;
      height = canvas.height = window.innerHeight;
      offCanvas.width = height;
      offCanvas.height = height;
      offCtx.clearRect(0, 0, height, height);
      offCtx.drawImage(logoImg, 0, 0, height, height);
      offsetX = (width - height) / 2;
      // init particles in logo square
      particles.current = Array.from({ length: numParticles }).map(() => {
        const x = offsetX + randomBetween(0, height);
        const y = randomBetween(0, height);
        const vx = randomBetween(-0.5, 0.5);
        const vy = randomBetween(-0.5, 0.5);
        return { x, y, vx, vy, initVx: vx, initVy: vy };
      });
    }

    logoImg.onload = () => {
      setup();
      animate();
    };
    window.addEventListener("resize", setup);
    window.addEventListener("mousemove", e => { mouse.current.x = e.clientX; mouse.current.y = e.clientY; });
    window.addEventListener("mouseleave", () => { mouse.current.x = null; mouse.current.y = null; });

    function animate() {
      // 1. clear to black wall
      ctx.fillStyle = "#000";
      ctx.fillRect(0, 0, width, height);
      
      // 2. update particles within square
      particles.current.forEach(p => {
        p.x += p.vx;
        p.y += p.vy;
        // wrap X in square
        if (p.x < offsetX) p.x = offsetX + height;
        if (p.x > offsetX + height) p.x = offsetX;
        // wrap Y
        if (p.y < 0) p.y = height;
        if (p.y > height) p.y = 0;
        // restore speed
        p.vx += (p.initVx - p.vx) * restoreFactor;
        p.vy += (p.initVy - p.vy) * restoreFactor;
      });

      // 3. draw links
      particles.current.forEach((a, i) => {
        for (let j = i + 1; j < particles.current.length; j++) {
          const b = particles.current[j];
          let dx = b.x - a.x;
          let dy = b.y - a.y;
          if (dx > width/2) dx -= width;
          if (dx < -width/2) dx += width;
          if (dy > height/2) dy -= height;
          if (dy < -height/2) dy += height;
          const dist = Math.hypot(dx, dy);
          if (dist < maxDist) {
            const alpha = 1 - dist / maxDist;
            const midX = (a.x + dx/2) - offsetX;
            const midY = (a.y + dy/2);
            let col = [0,0,0];
            if (midX >= 0 && midX < height && midY >= 0 && midY < height) {
              col = offCtx.getImageData(Math.floor(midX), Math.floor(midY),1,1).data;
            }
            ctx.strokeStyle = `rgba(${col[0]},${col[1]},${col[2]},${alpha})`;
            ctx.lineWidth = alpha;
            ctx.beginPath();
            ctx.moveTo(a.x, a.y);
            ctx.lineTo((a.x+dx+width)%width, (a.y+dy+height)%height);
            ctx.stroke();
          }
        }
      });

      // 4. draw particles
      particles.current.forEach(p => {
        const lx = p.x - offsetX;
        const ly = p.y;
        let col = [0,0,0];
        if (lx >= 0 && lx < height && ly >= 0 && ly < height) {
          col = offCtx.getImageData(Math.floor(lx), Math.floor(ly),1,1).data;
        }
        ctx.fillStyle = `rgba(${col[0]},${col[1]},${col[2]},0.9)`;
        ctx.beginPath(); ctx.arc(p.x,p.y,2,0,Math.PI*2); ctx.fill();
      });

      // 5. flashlight: big blurred circle then small sharp circle
      if (mouse.current.x !== null) {
        // loop blur steps from outer to inner, draw only logo without dark overlay
        const steps = 8;
        for (let i = 0; i < steps; i++) {
          const t = i / (steps - 1);
          const r = revealRadius * (1 - 0.2 * t);
          const blur = 8 * (1 - t);
          const grayPct = blur / 16 * 100;
          ctx.save();
          ctx.beginPath();
          ctx.arc(mouse.current.x, mouse.current.y, r, 0, Math.PI * 2);
          ctx.clip();
          ctx.filter = `blur(${blur}px) grayscale(${grayPct}%)`;
          ctx.drawImage(logoImg, offsetX, 0, height, height);
          ctx.filter = 'none';
          ctx.restore();
        }
      }

      requestAnimationFrame(animate);
    }
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
    // redirect into 42 OAuth
    window.location.href = "/auth/42/login";
  };

  const handleMagicLink = async (email) => {
    // call your backend to send the magic link
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
      <LoginCard
        onLogin={handleLogin}
        onMagicLink={handleMagicLink}
      />
    </div>
  );
}