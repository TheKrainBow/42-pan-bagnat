// src/Pages/Login.jsx
import React, { useEffect, useRef } from "react";
import "./Login.css";
import LoginCard from "./LoginCard";

export default function LoginPage() {
  const canvasRef = useRef(null);
  const particles = useRef([]);
  const mouse = useRef({ x: null, y: null });
  const revealAnimation = useRef(null);
  const revealRadius = 100;
  const numParticles = 200;
  const maxDist = 50;

  useEffect(() => {
    const canvas = canvasRef.current;
    const ctx = canvas.getContext("2d");
    let width, height;
    let offsetX;
    let topY, bottomY;

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
      topY = 220;
      bottomY = height - 220;
      offCtx.clearRect(0, 0, height, height);
      offCtx.drawImage(logoImg, 0, 0, height, height);
      offsetX = (width - height) / 2;
      // init particles in logo square
      particles.current = Array.from({ length: numParticles }).map(() => {
        const x = offsetX + randomBetween(0, height);
        const y = randomBetween(bottomY, topY);
        const vx = randomBetween(-0.5, 0.5);
        const vy = randomBetween(-0.5, 0.5);
        const speed = randomBetween(0.5, 1);
        const size = randomBetween(0, 70);
        return { x, y, vx, vy, initVx: vx, initVy: vy, speed: speed, size: size};
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
        p.x += (p.vx) * p.speed;
        p.y += (p.vy) * p.speed;
        // wrap X in square
        if (p.x < offsetX) p.x = offsetX + height;
        if (p.x > offsetX + height) p.x = offsetX;
        // wrap Y
        if (p.y < topY) p.y = bottomY;
        if (p.y > bottomY) p.y = topY;
      });

      // 3. draw links
      particles.current.forEach((a, i) => {
        for (let j = i + 1; j < particles.current.length; j++) {
          const b = particles.current[j];
          let dx = b.x - a.x;
          let dy = b.y - a.y;
          const dist = Math.hypot(dx, dy);
          const localMaxDist = maxDist + a.size + b.size;
          if (dist < localMaxDist ) {
            const alpha = (1 - dist / localMaxDist) / 2 + 0.35;
            const x1 = a.x - offsetX;
            const y1 = a.y;
            const x2 = b.x - offsetX;
            const y2 = b.y;

            let col1 = offCtx.getImageData(Math.floor(x1), Math.floor(y1), 1, 1).data;
            let col2 = offCtx.getImageData(Math.floor(x2), Math.floor(y2), 1, 1).data;

            if (col1[0] === 0 && col1[1] === 0 && col1[2] === 0) col1 = [72, 60, 60, col1[3]];
            if (col2[0] === 0 && col2[1] === 0 && col2[2] === 0) col2 = [72, 60, 60, col2[3]];
        
            const gradient = ctx.createLinearGradient(a.x, a.y, b.x, b.y);
            gradient.addColorStop(0, `rgba(${col1[0]}, ${col1[1]}, ${col1[2]}, ${alpha})`);
            gradient.addColorStop(1, `rgba(${col2[0]}, ${col2[1]}, ${col2[2]}, ${alpha})`);

            ctx.strokeStyle = gradient;
            ctx.lineWidth = alpha;
            ctx.beginPath();
            ctx.moveTo(a.x, a.y);
            ctx.lineTo(b.x, b.y);
            ctx.stroke();
          }
        }
      });

      let centerX = mouse.current.x ?? width / 2;
      let centerY = mouse.current.y ?? height / 2;

      // 5. flashlight: big blurred circle then small sharp circle
      if (mouse.current.x !== null || revealAnimation.current !== null) {
          const maxRadius = Math.hypot(width, height); // full screen diagonal
          const currentRadius = revealAnimation.current ?? revealRadius;

          if (revealAnimation.current !== null) {
            revealAnimation.current += 10; // grow speed
            centerX = width / 2;
            centerY = height / 2;
            if (revealAnimation.current >= maxRadius) {
              revealAnimation.current = null; // done
            }
          }

          const steps = 8;
          for (let i = 0; i < steps; i++) {
            const t = i / (steps - 1);
            const r = currentRadius * (1 - 0.2 * t);
            const blur = 8 * (1 - t);
            const grayPct = blur / 16 * 100;
            ctx.save();
            ctx.beginPath();
            ctx.arc(centerX, centerY, r, 0, Math.PI * 2);
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
    revealAnimation.current = revealRadius;
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