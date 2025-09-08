import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import './Tour.css';
import { tours } from './tours';

const TourCtx = createContext(null);

export function TourProvider({ children }) {
  const registry = useRef(new Map()); // id -> HTMLElement
  const [activeId, setActiveId] = useState(null);
  const [stepIndex, setStepIndex] = useState(0);

  // restore persisted tour
  useEffect(() => {
    const tid = localStorage.getItem('pb:tutorial');
    if (tid && tours[tid]) {
      const si = parseInt(localStorage.getItem('pb:tutorialStep') || '1', 10) - 1;
      setActiveId(tid);
      setStepIndex(Number.isFinite(si) ? Math.max(0, Math.min(si, (tours[tid].length - 1))) : 0);
    }
  }, []);

  const steps = activeId ? tours[activeId] : [];
  const step = steps[stepIndex] || null;

  const register = useCallback((id, el) => {
    if (!id || !el) return () => {};
    registry.current.set(id, el);
    return () => registry.current.delete(id);
  }, []);

  const start = useCallback((id) => {
    if (!tours[id]) return;
    setActiveId(id);
    setStepIndex(0);
    localStorage.setItem('pb:tutorial', id);
    localStorage.setItem('pb:tutorialStep', '1');
  }, []);

  const stop = useCallback(() => {
    setActiveId(null);
    setStepIndex(0);
    localStorage.removeItem('pb:tutorial');
    localStorage.removeItem('pb:tutorialStep');
  }, []);

  const next = useCallback(() => {
    if (!activeId) return;
    setStepIndex((i) => {
      const last = steps.length - 1;
      if (i >= last) {
        setTimeout(() => stop(), 0);
        return i;
      }
      const n = Math.min(i + 1, last);
      localStorage.setItem('pb:tutorialStep', String(n + 1));
      return n;
    });
  }, [activeId, steps.length, stop]);

  const prev = useCallback(() => {
    if (!activeId) return;
    setStepIndex((i) => {
      const n = Math.max(i - 1, 0);
      localStorage.setItem('pb:tutorialStep', String(n + 1));
      return n;
    });
  }, [activeId]);

  // proceedOn: 'click' → attach temporary listener
  useEffect(() => {
    if (!activeId || !step || step.proceedOn !== 'click') return;
    const el = registry.current.get(step.anchor);
    if (!el) return;
    const handler = () => next();
    el.addEventListener('click', handler, { once: true });
    return () => el.removeEventListener('click', handler);
  }, [activeId, step, next]);

  const value = useMemo(() => ({
    activeId, stepIndex, steps,
    start, stop, next, prev,
    register,
  }), [activeId, stepIndex, steps, start, stop, next, prev, register]);

  return (
    <TourCtx.Provider value={value}>
      {children}
      <TourOverlay />
    </TourCtx.Provider>
  );
}

export function useTour() {
  return useContext(TourCtx);
}

export function TourAnchor({ id, children }) {
  const ref = useRef(null);
  const { register } = useTour() || {};
  useEffect(() => (register ? register(id, ref.current) : () => {}), [register, id]);
  return <div ref={ref}>{children}</div>;
}

function TourOverlay() {
  const ctx = useTour();
  if (!ctx || !ctx.activeId) return null;
  const step = ctx.steps[ctx.stepIndex];
  const [rect, setRect] = useState(null);
  useEffect(() => {
    function update() {
      const el = document.querySelector(`[data-tour-anchor=\"${step.anchor}\"]`);
      const b = el ? el.getBoundingClientRect() : null;
      setRect(b);
    }
    update();
    const ro = new ResizeObserver(update);
    const el = document.querySelector(`[data-tour-anchor=\"${step.anchor}\"]`);
    if (el) ro.observe(el);
    window.addEventListener('scroll', update, true);
    window.addEventListener('resize', update);
    return () => {
      ro.disconnect();
      window.removeEventListener('scroll', update, true);
      window.removeEventListener('resize', update);
    };
  }, [step.anchor]);

  const style = rect ? {
    top: `${Math.max(0, rect.top + window.scrollY - 8)}px`,
    left: `${Math.max(0, rect.left + window.scrollX - 8)}px`,
    width: `${rect.width + 16}px`,
    height: `${rect.height + 16}px`,
  } : { top: '120px', left: '120px', width: '320px', height: '40px' };

  const vw = Math.max(document.documentElement.clientWidth || 0, window.innerWidth || 0);
  const vh = Math.max(document.documentElement.clientHeight || 0, window.innerHeight || 0);
  const b = rect || { top: 120, left: 120, width: 320, height: 40 };
  const topBlock = { top: 0, left: 0, width: vw, height: (b.top + window.scrollY) };
  const leftBlock = { top: (b.top + window.scrollY), left: 0, width: (b.left + window.scrollX), height: b.height };
  const rightBlock = { top: (b.top + window.scrollY), left: (b.left + window.scrollX) + b.width, width: vw - ((b.left + window.scrollX) + b.width), height: b.height };
  const bottomBlock = { top: (b.top + window.scrollY) + b.height, left: 0, width: vw, height: vh - ((b.top + window.scrollY) + b.height) };

  const hideNext = step?.nextHidden || step?.proceedOn === 'click';
  const goPrev = () => ctx.prev();
  const goNext = () => ctx.next();
  const exit = () => ctx.stop();

  return (
    <div className="tour-root">
      <div className="tour-block" style={{ top: topBlock.top, left: topBlock.left, width: topBlock.width, height: topBlock.height }} />
      <div className="tour-block" style={{ top: leftBlock.top, left: leftBlock.left, width: leftBlock.width, height: leftBlock.height }} />
      <div className="tour-block" style={{ top: rightBlock.top, left: rightBlock.left, width: rightBlock.width, height: rightBlock.height }} />
      <div className="tour-block" style={{ top: bottomBlock.top, left: bottomBlock.left, width: bottomBlock.width, height: bottomBlock.height }} />
      <div className="tour-highlight" style={style} />
      <div className="tour-panel" style={{ top: style.top, left: style.left }}>
        <div className="tour-title">{step?.title || 'Step'}</div>
        <div className="tour-desc">{step?.desc || ''}</div>
        <div className="tour-steps">Step {ctx.stepIndex + 1} / {ctx.steps.length}</div>
        <div className="tour-actions">
          <button className="tour-btn" onClick={exit}>Leave tutorial ✕</button>
          <div style={{ flex: 1 }} />
          <button className="tour-btn" onClick={goPrev} disabled={ctx.stepIndex <= 0}>Previous</button>
          {!hideNext && (
            <button className="tour-btn primary" onClick={goNext}>Next</button>
          )}
        </div>
      </div>
    </div>
  );
}

export function dataAnchorProps(id) {
  return { 'data-tour-anchor': id };
}
