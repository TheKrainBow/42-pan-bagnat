import React, { useEffect, useReducer, useRef, useState } from 'react';
import { Canvas, PencilBrush, Textbox, FabricImage, ActiveSelection } from 'fabric';
import { Wheel } from '@uiw/react-color';
import { toast } from 'react-toastify';
import Button from 'Global/Button/Button';
import { fetchWithAuth } from 'Global/utils/Auth';
import { DASHBOARD_WIDTH, DASHBOARD_HEIGHT, resolveCssColor, resolveBaseFontFamily, computeImageOverlays } from './dashboardCanvas';
import './Dashboard.css';

const TOOLS = { SELECT: 'select', DRAW: 'draw', TEXT: 'text' };

// verticalAlign/pbBoxHeight are our own properties, not native Fabric ones —
// pass them explicitly wherever a Textbox gets serialized (toJSON) or cloned
// so they survive save/load, undo/redo and copy/paste.
const CUSTOM_TEXT_PROPS = ['verticalAlign', 'pbBoxHeight'];

// A Textbox's `height` normally always snaps back to fit its text exactly
// (see Fabric's Textbox.initDimensions, which sets height = calcTextHeight()
// on every change) and its text always renders anchored to the top of that
// height. To offer real top/middle/bottom alignment we need a box that can
// be taller than its text and text that renders shifted within it — neither
// of which Fabric supports out of the box for Textbox, so we patch each
// instance: `pbBoxHeight` lets the box stay taller than its content, and the
// overridden _getTopOffset shifts the rendered/cursor position within that
// extra space based on `verticalAlign`.
function setupTextboxVerticalAlign(tb) {
  if (!tb || tb.type !== 'textbox' || tb.__pbVerticalAlignPatched) return;
  tb.__pbVerticalAlignPatched = true;
  if (tb.verticalAlign === undefined) tb.verticalAlign = 'top';
  if (tb.pbBoxHeight === undefined) tb.pbBoxHeight = null;

  const originalInitDimensions = tb.initDimensions.bind(tb);
  tb.initDimensions = function pbInitDimensions() {
    originalInitDimensions();
    if (this.pbBoxHeight != null && this.pbBoxHeight > this.height) {
      this.height = this.pbBoxHeight;
    }
  };

  tb._getTopOffset = function pbGetTopOffset() {
    const extra = Math.max(0, this.height - this.calcTextHeight());
    const shift = this.verticalAlign === 'middle' ? extra / 2 : this.verticalAlign === 'bottom' ? extra : 0;
    return -this.height / 2 + shift;
  };
}

// Giphy's share links (giphy.com/gifs/some-slug-<id>) point at an HTML page,
// not an image — loading one straight into <img>/canvas always fails. The
// id is always the last "-"-separated token of the page's last path
// segment, and converts directly to a real .gif asset URL, so resolve it
// instead of making the admin dig up the direct link themselves.
function resolveGiphyPageUrl(url) {
  let parsed;
  try {
    parsed = new URL(url);
  } catch {
    return url;
  }
  if (!/(^|\.)giphy\.com$/i.test(parsed.hostname) || /^(media|i)\./i.test(parsed.hostname)) {
    return url;
  }
  const segments = parsed.pathname.split('/').filter(Boolean);
  const last = segments[segments.length - 1];
  const id = last?.split('-').pop();
  if (!id || !/^[a-zA-Z0-9]{5,}$/.test(id)) return url;
  return `https://media.giphy.com/media/${id}/giphy.gif`;
}

export default function DashboardEditor() {
  const canvasElRef = useRef(null);
  const fabricRef = useRef(null);
  const wrapperRef = useRef(null);
  const fontFamilyRef = useRef('sans-serif');
  const clipboardRef = useRef(null);
  const handleSaveRef = useRef(null);
  const pushHistoryRef = useRef(null);
  const historyRef = useRef([]); // stack of canvas.toJSON() snapshots, as strings
  const historyIndexRef = useRef(-1);
  const isRestoringHistoryRef = useRef(false);

  // Scale-to-fit-width, exactly like DashboardView — without this the canvas
  // rendered at its native 1600x800 and admins had to scroll to reach the
  // bottom, and what they saw didn't match the proportions real viewers get.
  // Fabric reads the canvas element's actual on-screen (CSS-scaled) box for
  // pointer math, so drawing/dragging still line up correctly at any scale.
  const [scale, setScale] = useState(1);

  const [tool, setTool] = useState(TOOLS.SELECT);
  const [penColor, setPenColor] = useState('#7bffb2');
  const [penSize, setPenSize] = useState(4);
  const [showColorPicker, setShowColorPicker] = useState(false);
  const [activeText, setActiveText] = useState(null); // the selected Textbox, if any
  const [fontSize, setFontSize] = useState(28);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [gifURL, setGifURL] = useState('');
  const [showGifInput, setShowGifInput] = useState(false);
  // Fabric mutates the active Textbox's selectionStart/selectionEnd in place
  // while the user highlights text — that doesn't fire any canvas-level
  // 'selection:*' event, so React never learns about it on its own. Bumping
  // this on Fabric's 'text:selection:changed' forces a re-render so the
  // Bold/Italic toolbar state (and what a click will toggle) tracks the
  // live highlighted range instead of a stale one.
  const [, bumpSelectionTick] = useReducer(c => c + 1, 0);
  // Recomputed whenever an image object moves/resizes/rotates or the canvas
  // is (re)loaded, so the GIF <img> overlays (see computeImageOverlays) stay
  // aligned with their underlying Fabric object during editing.
  const [, bumpImageTick] = useReducer(c => c + 1, 0);

  // Init canvas once.
  useEffect(() => {
    const canvas = new Canvas(canvasElRef.current, {
      width: DASHBOARD_WIDTH,
      height: DASHBOARD_HEIGHT,
      backgroundColor: resolveCssColor('--background', '#141414'),
    });
    fabricRef.current = canvas;
    fontFamilyRef.current = resolveBaseFontFamily();

    const syncActiveText = () => {
      const obj = canvas.getActiveObject();
      if (obj && obj.type === 'textbox') {
        setActiveText(obj);
        setFontSize(obj.fontSize || 28);
      } else {
        setActiveText(null);
      }
    };
    canvas.on('selection:created', syncActiveText);
    canvas.on('selection:updated', syncActiveText);
    canvas.on('selection:cleared', () => setActiveText(null));
    canvas.on('text:selection:changed', bumpSelectionTick);
    canvas.on('text:changed', bumpSelectionTick);
    canvas.on('object:added', bumpImageTick);
    canvas.on('object:removed', bumpImageTick);
    canvas.on('object:moving', bumpImageTick);
    canvas.on('object:scaling', bumpImageTick);
    canvas.on('object:rotating', bumpImageTick);

    // Undo/redo: a plain stack of full canvas snapshots. Simple and
    // reliable for a canvas this size — no need for granular diffing.
    const MAX_HISTORY = 100;
    const pushHistory = () => {
      if (isRestoringHistoryRef.current) return;
      const json = JSON.stringify(canvas.toJSON(CUSTOM_TEXT_PROPS));
      if (json === historyRef.current[historyIndexRef.current]) return;
      historyRef.current = historyRef.current.slice(0, historyIndexRef.current + 1);
      historyRef.current.push(json);
      if (historyRef.current.length > MAX_HISTORY) {
        historyRef.current.shift();
      }
      historyIndexRef.current = historyRef.current.length - 1;
    };
    pushHistoryRef.current = pushHistory;

    const restoreHistory = async (index) => {
      if (index < 0 || index >= historyRef.current.length) return;
      isRestoringHistoryRef.current = true;
      canvas.discardActiveObject();
      await canvas.loadFromJSON(JSON.parse(historyRef.current[index]));
      canvas.getObjects().forEach(setupTextboxVerticalAlign);
      canvas.renderAll();
      historyIndexRef.current = index;
      isRestoringHistoryRef.current = false;
      bumpImageTick();
    };
    const undo = () => restoreHistory(historyIndexRef.current - 1);
    const redo = () => restoreHistory(historyIndexRef.current + 1);

    canvas.on('object:added', pushHistory);
    canvas.on('object:removed', pushHistory);
    canvas.on('object:modified', pushHistory);
    canvas.on('text:editing:exited', pushHistory);

    fetchWithAuth('/api/v1/dashboard')
      .then(r => r.json())
      .then(async (data) => {
        if (data?.canvas_json && Array.isArray(data.canvas_json.objects) && data.canvas_json.objects.length > 0) {
          await canvas.loadFromJSON(data.canvas_json);
          canvas.getObjects().forEach(setupTextboxVerticalAlign);
          canvas.renderAll();
          bumpImageTick();
        }
      })
      .catch(err => console.error('Failed to load dashboard', err))
      .finally(() => {
        setLoading(false);
        // Baseline snapshot so the very first undo has something to land on.
        historyRef.current = [JSON.stringify(canvas.toJSON(CUSTOM_TEXT_PROPS))];
        historyIndexRef.current = 0;
      });

    const handleKeyDown = (e) => {
      const key = e.key.toLowerCase();
      const withMeta = e.ctrlKey || e.metaKey;

      // Ctrl/Cmd+S always saves and never lets the browser's "Save page"
      // dialog open, regardless of focus (toolbar input or text editing).
      if (withMeta && key === 's') {
        e.preventDefault();
        handleSaveRef.current?.();
        return;
      }

      // Ctrl/Cmd+Z to undo, Ctrl/Cmd+Shift+Z (or +Y) to redo.
      if (withMeta && key === 'z') {
        e.preventDefault();
        if (e.shiftKey) {
          redo();
        } else {
          undo();
        }
        return;
      }
      if (withMeta && key === 'y') {
        e.preventDefault();
        redo();
        return;
      }

      // Don't hijack normal typing in the toolbar's own inputs (font size,
      // GIF URL, ...) or while a Textbox is being edited in-place — let
      // native text editing/copy/paste behave normally there.
      const tag = document.activeElement?.tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA') return;
      const active = canvas.getActiveObject();
      if (active?.isEditing) return;

      if (e.key === 'Delete' || e.key === 'Backspace') {
        if (!active) return;
        e.preventDefault();
        const targets = active instanceof ActiveSelection ? active.getObjects() : [active];
        targets.forEach(obj => canvas.remove(obj));
        canvas.discardActiveObject();
        canvas.requestRenderAll();
        return;
      }

      if (withMeta && key === 'c') {
        if (!active) return;
        e.preventDefault();
        active.clone(CUSTOM_TEXT_PROPS).then(cloned => {
          clipboardRef.current = cloned;
        });
        return;
      }

      if (withMeta && key === 'v') {
        if (!clipboardRef.current) return;
        e.preventDefault();
        clipboardRef.current.clone(CUSTOM_TEXT_PROPS).then(cloned => {
          canvas.discardActiveObject();
          cloned.set({
            left: (cloned.left ?? 0) + 24,
            top: (cloned.top ?? 0) + 24,
            evented: true,
          });
          if (cloned instanceof ActiveSelection) {
            cloned.canvas = canvas;
            cloned.forEachObject(obj => { setupTextboxVerticalAlign(obj); canvas.add(obj); });
            cloned.setCoords();
          } else {
            setupTextboxVerticalAlign(cloned);
            canvas.add(cloned);
          }
          canvas.setActiveObject(cloned);
          canvas.requestRenderAll();
          // Repeated Ctrl+V cascades diagonally instead of stacking exactly.
          clipboardRef.current = cloned;
        });
      }
    };
    window.addEventListener('keydown', handleKeyDown);

    return () => {
      window.removeEventListener('keydown', handleKeyDown);
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

  // Keep drawing mode/brush in sync with the selected tool and pen settings.
  useEffect(() => {
    const canvas = fabricRef.current;
    if (!canvas) return;
    canvas.isDrawingMode = tool === TOOLS.DRAW;
    if (tool === TOOLS.DRAW) {
      canvas.freeDrawingBrush = new PencilBrush(canvas);
      canvas.freeDrawingBrush.color = penColor;
      canvas.freeDrawingBrush.width = penSize;
    }
  }, [tool, penColor, penSize]);

  const addText = () => {
    const canvas = fabricRef.current;
    if (!canvas) return;
    const tb = new Textbox('Double-click to edit', {
      left: DASHBOARD_WIDTH / 2 - 150,
      top: DASHBOARD_HEIGHT / 2 - 20,
      width: 300,
      fontSize,
      fontFamily: fontFamilyRef.current,
      fill: '#f0f0f0',
    });
    setupTextboxVerticalAlign(tb);
    canvas.add(tb);
    canvas.setActiveObject(tb);
    setTool(TOOLS.SELECT);
    canvas.requestRenderAll();
  };

  const addGif = async () => {
    const trimmed = gifURL.trim();
    if (!trimmed) return;
    const url = resolveGiphyPageUrl(trimmed);
    const canvas = fabricRef.current;
    if (!canvas) return;
    try {
      const img = await FabricImage.fromURL(url, { crossOrigin: 'anonymous' });
      if (img.width > 480) img.scaleToWidth(480);
      img.set({ left: DASHBOARD_WIDTH / 2 - (img.getScaledWidth() / 2), top: DASHBOARD_HEIGHT / 2 - (img.getScaledHeight() / 2) });
      canvas.add(img);
      canvas.setActiveObject(img);
      canvas.requestRenderAll();
      setGifURL('');
      setShowGifInput(false);
    } catch (err) {
      toast.error('Failed to load image from that URL');
      console.error(err);
    }
  };

  // Applies a style patch to the active Textbox. While editing with a range
  // highlighted, it only touches that range — never the whole box. With no
  // highlight while editing there's no well-defined target, so it's a no-op.
  // Only when the box itself is selected (not in text-edit mode at all) does
  // it fall back to styling the whole box.
  const applyTextStyle = (patch) => {
    const canvas = fabricRef.current;
    const obj = activeText;
    if (!canvas || !obj) return;
    if (obj.isEditing) {
      if (obj.selectionStart === obj.selectionEnd) return; // nothing highlighted
      obj.setSelectionStyles(patch, obj.selectionStart, obj.selectionEnd);
      obj.dirty = true;
    } else {
      obj.set(patch);
    }
    canvas.requestRenderAll();
    // Adding/removing objects and finishing a text edit push history
    // automatically via canvas events; direct property mutations like this
    // one don't fire any of those, so record it explicitly.
    pushHistoryRef.current?.();
    // Same story for React: mutating the Fabric object in place doesn't
    // trigger a re-render, so the Bold/Italic buttons' active state (and
    // what the next click will toggle) would stay stale until the user
    // reselects. Force one so it reflects the style we just applied.
    bumpSelectionTick();
  };

  const hasHighlight = activeText?.isEditing && activeText.selectionStart !== activeText.selectionEnd;
  const isBoldActive = activeText && (
    hasHighlight
      ? activeText.getSelectionStyles(activeText.selectionStart, activeText.selectionEnd).every(s => s.fontWeight === 'bold')
      : (!activeText.isEditing && activeText.fontWeight === 'bold')
  );
  const isItalicActive = activeText && (
    hasHighlight
      ? activeText.getSelectionStyles(activeText.selectionStart, activeText.selectionEnd).every(s => s.fontStyle === 'italic')
      : (!activeText.isEditing && activeText.fontStyle === 'italic')
  );
  const canToggleStyle = activeText && (hasHighlight || !activeText.isEditing);

  const imageOverlays = fabricRef.current ? computeImageOverlays(fabricRef.current) : [];

  // Paragraph-level properties (alignment, box height) apply to the whole
  // Textbox regardless of any in-progress highlight — unlike applyTextStyle,
  // there's no per-character variant of "which edge is this box anchored to".
  const setTextAlign = (align) => {
    const canvas = fabricRef.current;
    const obj = activeText;
    if (!canvas || !obj) return;
    obj.set('textAlign', align);
    canvas.requestRenderAll();
    pushHistoryRef.current?.();
    bumpSelectionTick();
  };

  const setVerticalAlign = (align) => {
    const canvas = fabricRef.current;
    const obj = activeText;
    if (!canvas || !obj) return;
    obj.verticalAlign = align;
    obj.dirty = true;
    canvas.requestRenderAll();
    pushHistoryRef.current?.();
    bumpSelectionTick();
  };

  const setBoxHeight = (height) => {
    const canvas = fabricRef.current;
    const obj = activeText;
    if (!canvas || !obj) return;
    obj.pbBoxHeight = height;
    obj.initDimensions();
    obj.setCoords();
    canvas.requestRenderAll();
    pushHistoryRef.current?.();
    bumpSelectionTick();
  };

  const handleSave = async () => {
    const canvas = fabricRef.current;
    if (!canvas) return;
    setSaving(true);
    try {
      const json = canvas.toJSON(CUSTOM_TEXT_PROPS);
      const res = await fetchWithAuth('/api/v1/admin/dashboard', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ canvas_json: json }),
      });
      if (!res.ok) throw new Error(await res.text());
      toast.success('Dashboard saved');
    } catch (err) {
      toast.error(err.message || 'Failed to save dashboard');
    } finally {
      setSaving(false);
    }
  };
  handleSaveRef.current = handleSave;

  return (
    <div className="dashboard-editor-page">
      <h2 className="dashboard-editor-title">Dashboard</h2>

      <div className="dashboard-editor-toolbar">
        <div className="dashboard-toolbar-group">
          <button
            type="button"
            className={`dashboard-tool-btn ${tool === TOOLS.SELECT ? 'active' : ''}`}
            onClick={() => setTool(TOOLS.SELECT)}
          >
            Select
          </button>
          <button
            type="button"
            className={`dashboard-tool-btn ${tool === TOOLS.DRAW ? 'active' : ''}`}
            onClick={() => setTool(TOOLS.DRAW)}
          >
            ✏️ Pen
          </button>
        </div>

        <div className="dashboard-toolbar-group">
          <div className="dashboard-color-swatch">
            <button
              type="button"
              className="dashboard-color-swatch-btn"
              style={{ background: penColor }}
              onClick={() => setShowColorPicker(v => !v)}
              title="Pen color"
            />
            {showColorPicker && (
              <div className="dashboard-color-popover" onMouseLeave={() => setShowColorPicker(false)}>
                <Wheel color={penColor} onChange={c => setPenColor(c.hex)} width={160} height={160} />
              </div>
            )}
          </div>
          <label htmlFor="dashboard-pen-size">Size</label>
          <input
            id="dashboard-pen-size"
            className="dashboard-pen-size"
            type="range"
            min="1"
            max="40"
            value={penSize}
            onChange={e => setPenSize(Number(e.target.value))}
          />
        </div>

        <div className="dashboard-toolbar-group">
          <button type="button" className="dashboard-tool-btn" onClick={addText}>+ Text</button>
          {activeText && (
            <>
              <button
                type="button"
                className={`dashboard-tool-btn bold ${isBoldActive ? 'active' : ''}`}
                // Prevent the default mousedown behavior (shifting focus to
                // this button), which would otherwise blur the Textbox and
                // collapse its highlighted range before onClick even runs.
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => applyTextStyle({ fontWeight: isBoldActive ? 'normal' : 'bold' })}
                disabled={!canToggleStyle}
                title={canToggleStyle ? 'Bold' : 'Highlight some text first'}
              >
                B
              </button>
              <button
                type="button"
                className={`dashboard-tool-btn italic ${isItalicActive ? 'active' : ''}`}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => applyTextStyle({ fontStyle: isItalicActive ? 'normal' : 'italic' })}
                disabled={!canToggleStyle}
                title={canToggleStyle ? 'Italic' : 'Highlight some text first'}
              >
                I
              </button>
              <label htmlFor="dashboard-font-size">Size</label>
              <input
                id="dashboard-font-size"
                className="dashboard-font-size"
                type="number"
                min="8"
                max="200"
                value={fontSize}
                onChange={e => {
                  const size = Number(e.target.value) || 28;
                  setFontSize(size);
                  applyTextStyle({ fontSize: size });
                }}
              />
            </>
          )}
        </div>

        {activeText && (
          <div className="dashboard-toolbar-group">
            <button
              type="button"
              className={`dashboard-tool-btn ${activeText.textAlign === 'left' ? 'active' : ''}`}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => setTextAlign('left')}
              title="Align left"
            >
              L
            </button>
            <button
              type="button"
              className={`dashboard-tool-btn ${activeText.textAlign === 'center' ? 'active' : ''}`}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => setTextAlign('center')}
              title="Align center"
            >
              C
            </button>
            <button
              type="button"
              className={`dashboard-tool-btn ${activeText.textAlign === 'right' ? 'active' : ''}`}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => setTextAlign('right')}
              title="Align right"
            >
              R
            </button>
            <button
              type="button"
              className={`dashboard-tool-btn ${activeText.verticalAlign === 'top' ? 'active' : ''}`}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => setVerticalAlign('top')}
              title="Align top"
            >
              ⬆
            </button>
            <button
              type="button"
              className={`dashboard-tool-btn ${activeText.verticalAlign === 'middle' ? 'active' : ''}`}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => setVerticalAlign('middle')}
              title="Align middle"
            >
              ⬍
            </button>
            <button
              type="button"
              className={`dashboard-tool-btn ${activeText.verticalAlign === 'bottom' ? 'active' : ''}`}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => setVerticalAlign('bottom')}
              title="Align bottom"
            >
              ⬇
            </button>
            <label htmlFor="dashboard-box-height">Box H</label>
            <input
              id="dashboard-box-height"
              className="dashboard-font-size"
              type="number"
              min="1"
              value={Math.round(activeText.height || 0)}
              onMouseDown={(e) => e.stopPropagation()}
              onChange={e => setBoxHeight(Number(e.target.value) || 0)}
              title="Grow the box beyond its text to make middle/bottom alignment visible"
            />
          </div>
        )}

        <div className="dashboard-toolbar-group">
          {!showGifInput ? (
            <button type="button" className="dashboard-tool-btn" onClick={() => setShowGifInput(true)}>+ GIF / Image</button>
          ) : (
            <>
              <input
                type="text"
                placeholder="https://…/image.gif"
                value={gifURL}
                onChange={e => setGifURL(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && addGif()}
                style={{ width: 220 }}
                className="dashboard-font-size"
                autoFocus
              />
              <button type="button" className="dashboard-tool-btn" onClick={addGif}>Add</button>
              <button type="button" className="dashboard-tool-btn" onClick={() => { setShowGifInput(false); setGifURL(''); }}>Cancel</button>
            </>
          )}
        </div>
        <Button label={saving ? 'Saving…' : 'Save'} color="blue" onClick={handleSave} disabled={saving || loading} />
      </div>

      <div className="dashboard-editor-canvas-area" ref={wrapperRef}>
        <div className="dashboard-canvas-wrapper" style={{ width: DASHBOARD_WIDTH * scale, height: DASHBOARD_HEIGHT * scale }}>
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
