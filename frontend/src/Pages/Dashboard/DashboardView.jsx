import React, { useEffect, useRef, useState } from 'react';
import { Canvas } from 'fabric';
import { fetchWithAuth } from 'Global/utils/Auth';
import { DASHBOARD_WIDTH, DASHBOARD_HEIGHT, resolveCssColor, computeImageOverlays } from './dashboardCanvas';
import './Dashboard.css';

// Read-only render of the shared dashboard canvas: no selection, no editing,
// just the staff-authored content. Landing page for logged-in users instead
// of auto-opening their first module.
export default function DashboardView() {
  const canvasElRef = useRef(null);
  const wrapperRef = useRef(null);
  const [scale, setScale] = useState(1);
  const [empty, setEmpty] = useState(false);
  const [imageOverlays, setImageOverlays] = useState([]);

  useEffect(() => {
    const canvas = new Canvas(canvasElRef.current, {
      width: DASHBOARD_WIDTH,
      height: DASHBOARD_HEIGHT,
      selection: false,
      backgroundColor: resolveCssColor('--background', '#141414'),
    });

    let cancelled = false;
    fetchWithAuth('/api/v1/dashboard')
      .then(r => r.json())
      .then(async (data) => {
        if (cancelled) return;
        const hasContent = data?.canvas_json && Array.isArray(data.canvas_json.objects) && data.canvas_json.objects.length > 0;
        if (!hasContent) {
          setEmpty(true);
          return;
        }
        await canvas.loadFromJSON(data.canvas_json);
        if (cancelled) return;
        canvas.getObjects().forEach(o => {
          o.selectable = false;
          o.evented = false;
        });
        canvas.renderAll();
        setImageOverlays(computeImageOverlays(canvas));
      })
      .catch(err => {
        console.error('Failed to load dashboard', err);
      });

    return () => {
      cancelled = true;
      canvas.dispose();
    };
  }, []);

  useEffect(() => {
    const handleResize = () => {
      const w = wrapperRef.current?.clientWidth || DASHBOARD_WIDTH;
      setScale(Math.min(1, w / DASHBOARD_WIDTH));
    };
    handleResize();
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  return (
    <div className="dashboard-view-page">
      {empty && (
        <div className="dashboard-view-empty">Nothing here yet — check back later.</div>
      )}
      <div
        className="dashboard-canvas-viewport"
        ref={wrapperRef}
        style={{ display: empty ? 'none' : 'flex' }}
      >
        <div
          className="dashboard-canvas-wrapper"
          style={{ width: DASHBOARD_WIDTH * scale, height: DASHBOARD_HEIGHT * scale }}
        >
          <div className="dashboard-canvas-scaler" style={{ transform: `scale(${scale})` }}>
            <canvas ref={canvasElRef} />
            {imageOverlays.map(o => (
              <img
                key={o.key}
                src={o.src}
                alt=""
                className="dashboard-gif-overlay"
                style={{ left: o.left, top: o.top, width: o.width, height: o.height, transform: `rotate(${o.angle}deg)` }}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
