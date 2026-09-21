import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchWithAuth } from 'Global/utils/Auth';
import { StatTile, DailyActivityChart, ActivityHeatmap } from './StatsCharts';
import './Stats.css';

const RANGE_OPTIONS = [
  { label: '7 days', days: 7 },
  { label: '30 days', days: 30 },
  { label: '90 days', days: 90 },
];

const Stats = () => {
  const [rangeDays, setRangeDays] = useState(30);
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const { from, to } = useMemo(() => {
    const to = new Date();
    const from = new Date(to.getTime() - rangeDays * 24 * 60 * 60 * 1000);
    return { from: from.toISOString(), to: to.toISOString() };
  }, [rangeDays]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');
    (async () => {
      try {
        const res = await fetchWithAuth(`/api/v1/admin/stats?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`);
        if (!res || !res.ok) throw new Error('Failed to load stats');
        const json = await res.json();
        if (!cancelled) setData(json);
      } catch (err) {
        if (!cancelled) setError(err.message || 'Failed to load stats');
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [from, to]);

  return (
    <div className="stats-page">
      <div className="stats-header">
        <div>
          <h2>Stats</h2>
          <p>Module usage activity across Pan Bagnat. An activity is counted per user per module after 60 minutes of inactivity.</p>
        </div>
        <div className="stats-range">
          {RANGE_OPTIONS.map(opt => (
            <button
              key={opt.days}
              className={`stats-range-btn ${rangeDays === opt.days ? 'active' : ''}`}
              onClick={() => setRangeDays(opt.days)}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {loading ? (
        <div className="stats-empty">Loading stats…</div>
      ) : error ? (
        <div className="stats-empty">{error}</div>
      ) : (
        <>
          <div className="stats-tiles">
            <StatTile label="Activities" value={data.summary.activity_count} hint="Sessions of usage, 60min gap rule" />
            <StatTile label="Active users" value={data.summary.unique_users} />
            <StatTile label="Active modules" value={data.summary.active_modules} />
          </div>

          <div className="stats-panel">
            <h3>Daily activity</h3>
            <DailyActivityChart data={data.daily} />
          </div>

          <div className="stats-panel">
            <h3>Peak usage (day / hour)</h3>
            <ActivityHeatmap cells={data.heatmap} />
          </div>

          <div className="stats-panel">
            <h3>Modules</h3>
            {(!data.modules || data.modules.length === 0) ? (
              <div className="stats-empty">No module activity for this period.</div>
            ) : (
              <div className="stats-modules-list">
                {data.modules.map(m => (
                  <div key={m.module_id} className="stats-module-row" onClick={() => navigate(`/admin/stats/${m.module_id}`)}>
                    <img src={m.icon_url || '/icons/modules.png'} alt="" />
                    <span className="stats-module-name">{m.module_name}</span>
                    <div className="stats-module-meta">
                      <span>{m.activity_count} activities</span>
                      <span>{m.unique_users} users</span>
                      <span>Last active {m.last_active_at ? new Date(m.last_active_at).toLocaleString() : 'Never'}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
};

export default Stats;
