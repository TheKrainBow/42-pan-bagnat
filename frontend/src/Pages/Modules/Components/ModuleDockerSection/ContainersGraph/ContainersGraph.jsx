import React, { useEffect, useMemo, useRef, useState } from 'react';
import { fetchWithAuth } from 'Global/utils/Auth';
import Button from 'Global/Button/Button';
import './ContainersGraph.css';

export default function ContainersGraph({ moduleId }) {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const [scale, setScale] = useState(1);
  const [tx, setTx] = useState(0);
  const [ty, setTy] = useState(0);
  const dragRef = useRef(null);
  const viewportRef = useRef(null);
  const groupRefs = useRef(new Map());

  const groups = useMemo(() => {
    // Hide Pan Bagnat infra containers so users don't break the site here
    const filtered = (items || []).filter(it => !/(^|\b)pan[- ]?bagnat(\b|$)/i.test(it.project || ''));
    const map = new Map();
    filtered.forEach(it => {
      const key = it.project || 'external';
      if (!map.has(key)) map.set(key, { project: key, module_id: it.module_id, module_name: it.module_name, items: [] });
      map.get(key).items.push(it);
    });
    return Array.from(map.values()).sort((a,b) => a.project.localeCompare(b.project));
  }, [items]);

  // Build network statistics to detect shared (external) networks and project primary networks
  const netStats = useMemo(() => {
    const stats = new Map(); // name -> { projects:Set, count:number }
    (items || []).forEach(it => {
      (it.networks || []).forEach(n => {
        if (!stats.has(n)) stats.set(n, { projects: new Set(), count: 0 });
        const st = stats.get(n);
        st.projects.add(it.project || '');
        st.count += 1;
      });
    });
    return stats;
  }, [items]);

  const primaryNetworkByProject = useMemo(() => {
    const map = new Map(); // project -> {name,count}
    (items || []).forEach(it => {
      const p = it.project || '';
      (it.networks || []).forEach(n => {
        const curr = map.get(p) || { name: '', count: 0, counts: {} };
        curr.counts[n] = (curr.counts[n] || 0) + 1;
        map.set(p, curr);
      });
    });
    const out = new Map();
    map.forEach((v, p) => {
      let best = '', bestC = -1;
      Object.entries(v.counts || {}).forEach(([n, c]) => { if (c > bestC) { best = n; bestC = c; } });
      out.set(p, best);
    });
    return out;
  }, [items]);

  // Deterministic color per network (stable across renders)
  const networkColors = useMemo(() => {
    const palette = ['#8ab4f8','#7bffb2','#ff7b72','#ffd27b','#d07bff','#7bffd2','#ff7bd2','#9ccc65','#64b5f6','#ba68c8','#ff8a65','#4dd0e1'];
    const hash = (s) => {
      let h = 0; for (let i = 0; i < s.length; i++) { h = (h*31 + s.charCodeAt(i)) >>> 0; } return h;
    };
    const colors = new Map();
    (items || []).forEach(it => (it.networks || []).forEach(n => {
      if (!colors.has(n)) {
        const idx = hash(n) % palette.length;
        colors.set(n, palette[idx]);
      }
    }));
    return colors;
  }, [items]);

  useEffect(() => {
    setLoading(true);
    fetchWithAuth('/api/v1/admin/docker/ls')
      .then(r => r.json())
      .then(setItems)
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, [moduleId]);

  // layout groups in a grid
  const layout = useMemo(() => {
    const cols = 3;
    // Give more space between module groups
    const gapX = 520, gapY = 380;
    const positions = new Map();
    groups.forEach((g, idx) => {
      const row = Math.floor(idx / cols);
      const col = idx % cols;
      positions.set(g.project, { x: col * gapX, y: row * gapY });
    });
    return { positions, gapX, gapY };
  }, [groups]);

  // Fit a given group into the viewport with 50px margin
  const fitGroup = (project) => {
    const vp = viewportRef.current;
    const el = groupRefs.current.get(project);
    const pos = layout.positions.get(project) || { x: 0, y: 0 };
    if (!vp || !el) {
      // fallback to simple center based on layout pos
      setScale(1);
      setTx(200 - pos.x);
      setTy(120 - pos.y);
      return;
    }
    const vpW = vp.clientWidth, vpH = vp.clientHeight;
    const rectW = el.offsetWidth, rectH = el.offsetHeight;
    const margin = 50;
    // scale so that there is at least 50px padding on both axes
    const scaleX = (vpW - 2 * margin) / rectW;
    const scaleY = (vpH - 2 * margin) / rectH;
    const s = Math.max(0.3, Math.min(2, Math.min(scaleX, scaleY)));
    setScale(s);
    // center the group within the viewport after scaling
    const dx = (vpW - s * rectW) / 2;
    const dy = (vpH - s * rectH) / 2;
    setTx(dx - pos.x * s);
    setTy(dy - pos.y * s);
  };

  // Center on current module group initially
  useEffect(() => {
    const current = groups.find(g => g.module_id === moduleId) || groups[0];
    if (!current) return;
    // defer to next frame so refs are set
    requestAnimationFrame(() => fitGroup(current.project));
  }, [moduleId, groups, layout]);

  const onWheel = (e) => {
    e.preventDefault();
    const delta = -e.deltaY;
    const factor = delta > 0 ? 1.1 : 0.9;
    const newScale = Math.min(2, Math.max(0.5, scale * factor));
    setScale(newScale);
  };

  const onMouseDown = (e) => {
    dragRef.current = { startX: e.clientX, startY: e.clientY, baseX: tx, baseY: ty };
  };
  const onMouseMove = (e) => {
    if (!dragRef.current) return;
    const dx = e.clientX - dragRef.current.startX;
    const dy = e.clientY - dragRef.current.startY;
    setTx(dragRef.current.baseX + dx);
    setTy(dragRef.current.baseY + dy);
  };
  const onMouseUp = () => { dragRef.current = null; };

  const resetToModule = () => {
    const current = groups.find(g => g.module_id === moduleId) || groups[0];
    if (!current) return;
    fitGroup(current.project);
  };

  if (loading) return <div>Loading containers…</div>;
  if (error) return <div style={{ color: 'tomato' }}>Error: {error}</div>;

  return (
    <div className="cg-root">
      <div className="cg-toolbar">
        <Button label="Back to current module" onClick={resetToModule} />
        <div className="spacer" />
        <Button label="-" onClick={() => setScale(s => Math.max(0.5, s * 0.9))} />
        <Button label="+" onClick={() => setScale(s => Math.min(2, s * 1.1))} />
      </div>
      <div className="cg-viewport" ref={viewportRef} onWheel={onWheel} onMouseDown={onMouseDown} onMouseMove={onMouseMove} onMouseUp={onMouseUp}>
        <div className="cg-canvas" style={{ transform: `translate(${tx}px, ${ty}px) scale(${scale})` }}>
          {groups.map(g => {
            const p = layout.positions.get(g.project) || { x: 0, y: 0 };
            return (
              <div
                key={g.project}
                className={`cg-group ${g.module_id === moduleId ? 'current' : ''}`}
                style={{ left: p.x, top: p.y }}
                ref={el => { if (el) groupRefs.current.set(g.project, el); }}
              >
                <div
                  className={`cg-group-title ${g.module_id && g.module_id !== moduleId ? 'link' : ''}`}
                  onClick={() => { if (g.module_id && g.module_id !== moduleId) window.location.href = `/admin/modules/${g.module_id}`; }}
                  title={g.module_id && g.module_id !== moduleId ? 'Open module page' : undefined}
                >
                  {g.project || 'external'}
                  {primaryNetworkByProject.get(g.project) && (
                    <span className="cg-primary-net" style={{ borderColor: networkColors.get(primaryNetworkByProject.get(g.project)) || '#8ab4f8', color: networkColors.get(primaryNetworkByProject.get(g.project)) || '#8ab4f8' }}>
                      {primaryNetworkByProject.get(g.project)}
                    </span>
                  )}
                </div>
                <div className="cg-cards">
                  {g.items.map(c => (
                    <ContainerCard
                      key={c.name}
                      c={c}
                      currentModuleId={moduleId}
                      netStats={netStats}
                      primaryNet={primaryNetworkByProject.get(g.project)}
                      networkColors={networkColors}
                    />
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function ContainerCard({ c, currentModuleId, netStats, primaryNet, networkColors }) {
  const name = c.name.includes('-1') ? c.name.replace(/-1$/, '') : c.name;
  const moduleId = c.module_id;
  const displayName = moduleId ? name.replace(`${c.project}-`, '') : name;
  const canAct = !!moduleId;

  const act = (action) => {
    if (!canAct) return;
    fetchWithAuth(`/api/v1/admin/modules/${moduleId}/docker/${displayName}/${action}`, { method: action === 'delete' ? 'DELETE' : 'POST' })
      .catch(err => console.error(action, 'failed:', err));
  };

  // choose a tint color: prefer project primaryNet if container is in it, else first network
  const tintNet = (c.networks || []).includes(primaryNet) ? primaryNet : (c.networks || [])[0];
  const tintColor = networkColors.get(tintNet || '') || undefined;
  const tint = tintColor ? tintFromHex(tintColor, 0.10) : undefined;
  const tintBorder = tintColor ? tintFromHex(tintColor, 0.45) : undefined;

  return (
    <div className={`cg-card status-${c.status}`} style={tintColor ? { backgroundColor: tint, borderColor: tintBorder } : undefined}>
      <div className="cg-card-name" title={c.name}>{displayName}</div>
      {Array.isArray(c.networks) && c.networks.length > 0 && (
        <div className="cg-networks" title={c.networks.join(', ')}>
          {c.networks.map(n => {
            const st = netStats.get(n);
            const shared = st && st.projects && st.projects.size > 1;
            const classes = ['chip'];
            const col = networkColors.get(n);
            return (
              <span key={n} className={classes.join(' ')} style={{
                backgroundColor: col ? tintFromHex(col, 0.10) : undefined,
                borderColor: col ? tintFromHex(col, shared ? 0.65 : 0.45) : undefined,
                color: col || undefined,
                boxShadow: primaryNet && n === primaryNet ? `0 0 0 2px ${tintFromHex(col || '#8ab4f8', 0.25)} inset` : undefined,
              }}>{n}</span>
            );
          })}
        </div>
      )}
      <div className="cg-card-actions">
        {c.status === 'exited' && (
          <Button icon="/icons/button-play.png" color="warning" onClick={() => act('start')} />
        )}
        {c.status === 'running' && (
          <Button icon="/icons/button-stop.png" color="warning" onClick={() => act('stop')} />
        )}
        <Button icon="/icons/button-refresh.png" color="warning" onClick={() => act('restart')} />
        <Button icon="/icons/button-delete.png" color="warning" onClick={() => act('delete')} />
      </div>
    </div>
  );
}

// convert hex like #rrggbb to rgba with alpha
function tintFromHex(hex, alpha) {
  if (!hex || hex[0] !== '#' || (hex.length !== 7 && hex.length !== 4)) return hex;
  let r, g, b;
  if (hex.length === 7) {
    r = parseInt(hex.slice(1,3), 16);
    g = parseInt(hex.slice(3,5), 16);
    b = parseInt(hex.slice(5,7), 16);
  } else {
    r = parseInt(hex[1] + hex[1], 16);
    g = parseInt(hex[2] + hex[2], 16);
    b = parseInt(hex[3] + hex[3], 16);
  }
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}
