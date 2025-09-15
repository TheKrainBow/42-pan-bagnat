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
      const label = key === 'orphans' ? 'Pan Bagnat' : key;
      if (!map.has(key)) map.set(key, { project: label, key, module_id: it.module_id, module_name: it.module_name, items: [] });
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
        if (n === 'pan-bagnat-net') return; // never choose internal infra as primary
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
    const gapX = 680, gapY = 420;
    const positions = new Map();
    const modules = groups.filter(g => g.key !== 'orphans');
    modules.forEach((g, idx) => {
      const row = Math.floor(idx / cols);
      const col = idx % cols;
      positions.set(g.project, { x: col * gapX, y: row * gapY });
    });
    const rows = Math.ceil(modules.length / cols);
    const pb = groups.find(g => g.key === 'orphans');
    if (pb) {
      positions.set(pb.project, { x: 0, y: rows * gapY + 200 });
    }
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
    const modulesOnly = groups.filter(g => g.key !== 'orphans');
    const current = modulesOnly.find(g => g.module_id === moduleId) || modulesOnly[0] || groups[0];
    if (!current) return;
    // defer to next frame so refs are set
    requestAnimationFrame(() => fitGroup(current.project));
  }, [moduleId, groups, layout]);

  const onWheel = (e) => {
    const vp = viewportRef.current;
    if (!vp) return;
    const rect = vp.getBoundingClientRect();
    const mouseX = e.clientX - rect.left;
    const mouseY = e.clientY - rect.top;
    // world coords before zoom
    const wx = (mouseX - tx) / scale;
    const wy = (mouseY - ty) / scale;
    const delta = -e.deltaY;
    const factor = delta > 0 ? 1.1 : 0.9;
    const newScale = Math.min(2, Math.max(0.5, scale * factor));
    // adjust translation so that (wx,wy) stays under the cursor
    const ntx = mouseX - wx * newScale;
    const nty = mouseY - wy * newScale;
    setTx(ntx);
    setTy(nty);
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

  const zoomAt = (factor, cx, cy) => {
    const vp = viewportRef.current;
    if (!vp) return;
    const rect = vp.getBoundingClientRect();
    const x = (typeof cx === 'number') ? (cx - rect.left) : rect.width / 2;
    const y = (typeof cy === 'number') ? (cy - rect.top) : rect.height / 2;
    const wx = (x - tx) / scale;
    const wy = (y - ty) / scale;
    const newScale = Math.min(2, Math.max(0.5, scale * factor));
    setTx(x - wx * newScale);
    setTy(y - wy * newScale);
    setScale(newScale);
  };

  const resetToModule = () => {
    const current = groups.find(g => g.module_id === moduleId) || groups[0];
    if (!current) return;
    fitGroup(current.project);
  };

  const composeAction = async (action) => {
    if (!moduleId) return;
    if (action === 'down' && !confirm('Compose down will stop and remove project containers. Continue?')) return;
    const url = action === 'rebuild'
      ? `/api/v1/admin/modules/${moduleId}/docker/compose/rebuild`
      : `/api/v1/admin/modules/${moduleId}/docker/compose/down`;
    await fetchWithAuth(url, { method: 'POST' });
    // Refresh list after action
    setLoading(true);
    fetchWithAuth('/api/v1/admin/docker/ls')
      .then(r => r.json())
      .then(setItems)
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  };

  if (loading) return <div>Loading containers…</div>;
  if (error) return <div style={{ color: 'tomato' }}>Error: {error}</div>;

  return (
    <div className="cg-root">
      <div className="cg-toolbar">
        <Button label="Back to current module" onClick={resetToModule} />
        <div className="spacer" />
        <Button label="-" onClick={() => zoomAt(0.9)} />
        <Button label="+" onClick={() => zoomAt(1.1)} />
        {/* module-level compose buttons removed; use per-card Deploy/Prune instead */}
      </div>
      <div className="cg-viewport" ref={viewportRef} onWheel={onWheel} onMouseDown={onMouseDown} onMouseMove={onMouseMove} onMouseUp={onMouseUp}>
        <div className="cg-canvas" style={{ transform: `translate(${tx}px, ${ty}px) scale(${scale})` }}>
          {[...groups.filter(g => g.key !== 'orphans'), ...groups.filter(g => g.key === 'orphans')].map(g => {
            const p = layout.positions.get(g.project) || { x: 0, y: 0 };
            return (
              <div
                key={g.project}
                className={`cg-group ${g.module_id === moduleId ? 'current' : ''}`}
                style={{ left: p.x, top: p.y }}
                ref={el => { if (el) groupRefs.current.set(g.project, el); }}
              >
                <div
                  className={`cg-group-title ${g.key !== 'orphans' && g.module_id && g.module_id !== moduleId ? 'link' : ''}`}
                  onClick={() => { if (g.key !== 'orphans' && g.module_id && g.module_id !== moduleId) window.location.href = `/admin/modules/${g.module_id}`; }}
                  title={g.key !== 'orphans' && g.module_id && g.module_id !== moduleId ? 'Open module page' : undefined}
                >
                  {g.project || 'external'}
                  {g.key !== 'orphans' && primaryNetworkByProject.get(g.key) && (
                    <span className="cg-primary-net" style={{ borderColor: networkColors.get(primaryNetworkByProject.get(g.key)) || '#8ab4f8', color: networkColors.get(primaryNetworkByProject.get(g.key)) || '#8ab4f8' }}>
                      {primaryNetworkByProject.get(g.key)}
                    </span>
                  )}
                  <span className="spacer" />
                  {g.key !== 'orphans' && g.module_id && (
                    <span className="cg-group-actions">
                      <Button label="Deploy" color="green" onClick={async () => { await fetchWithAuth(`/api/v1/admin/modules/${g.module_id}/docker/compose/deploy`, { method: 'POST' }); try { localStorage.setItem(`logs.selection.${g.module_id}`, JSON.stringify({ module: true, containers: {} })) } catch {}; window.location.href = `/admin/modules/${g.module_id}?tab=logs`; }} />
                      <Button label="Prune" color="red" onClick={async () => { if (!confirm('Compose down will stop and remove project containers. Continue?')) return; await fetchWithAuth(`/api/v1/admin/modules/${g.module_id}/docker/compose/down`, { method: 'POST' }); }} />
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
                      primaryNet={primaryNetworkByProject.get(g.key)}
                      networkColors={networkColors}
                      missing={c.missing}
                      orphan={c.orphan}
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

function ContainerCard({ c, currentModuleId, netStats, primaryNet, networkColors, missing, orphan }) {
  const name = c.name.includes('-1') ? c.name.replace(/-1$/, '') : c.name;
  const moduleId = c.module_id;
  const displayName = moduleId ? name.replace(`${c.project}-`, '') : name;
  const canAct = !!moduleId;
  const orphanCore = orphan && (Array.isArray(c.networks) && c.networks.includes('pan-bagnat-core'));
  const actionDisabled = !!missing || orphanCore;

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

  const gotoDeploy = () => {
    if (!moduleId) return;
    try { localStorage.setItem(`logs.selection.${moduleId}`, JSON.stringify({ module: true, containers: {} })) } catch {}
    window.location.href = `/admin/modules/${moduleId}?tab=logs`;
  };

  const pruneProject = () => {
    if (!moduleId) return;
    if (!confirm('Prune (compose down) will stop and remove project containers. Continue?')) return;
    fetchWithAuth(`/api/v1/admin/modules/${moduleId}/docker/compose/down`, { method: 'POST' })
      .catch(err => console.error('down failed', err));
  };

  const disabledMessageMissing = 'This container has never been built. Use Deploy button first';
  const disabledMessageCore = 'Cannot interact with Pan bagnat Core containers from Webapp';
  const disabledMessageOrphan = 'Only delete is available for orphan containers';

  const canDeleteOrphan = orphan && !orphanCore && !missing && !canAct;

  const globalDelete = async () => {
    await fetchWithAuth(`/api/v1/admin/docker/${encodeURIComponent(c.name)}/delete`, { method: 'DELETE' });
  };

  const statusLabel = (s) => {
    switch ((s || '').toLowerCase()) {
      case 'running': return 'running';
      case 'exited': return 'stopped';
      case 'paused': return 'paused';
      case 'created': return 'created';
      case 'restarting': return 'restarting';
      case 'dead': return 'dead';
      default: return 'unknown';
    }
  };

  return (
    <div className={`cg-card status-${c.status} ${missing ? 'missing' : ''} ${orphan ? 'orphan' : ''}`} style={tintColor ? { backgroundColor: tint, borderColor: tintBorder } : undefined}>
      <div className="cg-card-name" title={c.name}>
        {displayName}{missing ? ' (never built)' : ''}
        {!missing && (
          <span className={`cg-status ${statusLabel(c.status)}`}>({statusLabel(c.status)})</span>
        )}
      </div>
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
          <Button icon="/icons/button-play.png" color="yellow" onClick={() => act('start')} disabled={actionDisabled || !canAct} disabledMessage={orphanCore ? disabledMessageCore : disabledMessageMissing} />
        )}
        {c.status === 'running' && (
          <Button icon="/icons/button-stop.png" color="yellow" onClick={() => act('stop')} disabled={actionDisabled || !canAct} disabledMessage={orphanCore ? disabledMessageCore : disabledMessageMissing} />
        )}
        <Button icon="/icons/button-refresh.png" color="yellow" onClick={() => act('restart')} disabled={actionDisabled || !canAct} disabledMessage={orphanCore ? disabledMessageCore : disabledMessageMissing} />
        <Button
          icon="/icons/button-delete.png"
          color="red"
          onClick={() => { if (canAct) act('delete'); else if (canDeleteOrphan) globalDelete(); }}
          disabled={actionDisabled || (!canAct && !canDeleteOrphan)}
          disabledMessage={orphanCore ? disabledMessageCore : (missing ? disabledMessageMissing : disabledMessageOrphan)}
        />
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
