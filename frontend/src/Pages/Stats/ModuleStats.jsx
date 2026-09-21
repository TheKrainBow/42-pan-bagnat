import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { fetchWithAuth } from 'Global/utils/Auth';
import { DailyActivityChart, ActivityHeatmap } from './StatsCharts';
import './Stats.css';

const RANGE_OPTIONS = [
  { label: '7 days', days: 7 },
  { label: '30 days', days: 30 },
  { label: '90 days', days: 90 },
];

const ModuleStats = () => {
  const { moduleId } = useParams();
  const navigate = useNavigate();
  const [rangeDays, setRangeDays] = useState(30);
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

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
        const res = await fetchWithAuth(`/api/v1/admin/stats/modules/${moduleId}?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`);
        if (!res || !res.ok) throw new Error('Failed to load module stats');
        const json = await res.json();
        if (!cancelled) setData(json);
      } catch (err) {
        if (!cancelled) setError(err.message || 'Failed to load module stats');
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [moduleId, from, to]);

  return (
    <div className="stats-page">
      <div className="stats-back-link" onClick={() => navigate('/admin/stats')}>← Back to Stats</div>
      <div className="stats-header">
        <div>
          <h2>{data?.module?.name || 'Module stats'}</h2>
          <p>Per-module activity, 60 minutes of inactivity starts a new activity.</p>
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
          <div className="stats-panel">
            <h3>Daily activity</h3>
            <DailyActivityChart data={data.daily} />
          </div>

          <div className="stats-panel">
            <h3>Peak usage (day / hour)</h3>
            <ActivityHeatmap cells={data.heatmap} />
          </div>

          <div className="stats-panel">
            <h3>Users</h3>
            {(!data.users || data.users.length === 0) ? (
              <div className="stats-empty">No user activity for this period.</div>
            ) : (
              <table className="stats-users-table">
                <thead>
                  <tr>
                    <th>User</th>
                    <th>Activities</th>
                    <th>Last active</th>
                  </tr>
                </thead>
                <tbody>
                  {data.users.map(u => (
                    <tr key={u.user_id}>
                      <td>
                        <div className="stats-user-cell">
                          <img src={u.photo_url || '/icons/users.png'} alt="" />
                          <span>{u.ft_login}</span>
                        </div>
                      </td>
                      <td>{u.activity_count}</td>
                      <td>{new Date(u.last_active_at).toLocaleString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </>
      )}
    </div>
  );
};

export default ModuleStats;
