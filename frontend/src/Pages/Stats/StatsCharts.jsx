import React from 'react';
import {
  ResponsiveContainer, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend,
} from 'recharts';

const ACTIVITY_COLOR = 'var(--blue)';
const USERS_COLOR = 'var(--green-dark)';

export const StatTile = ({ label, value, hint }) => (
  <div className="stats-tile">
    <div className="stats-tile-value">{value}</div>
    <div className="stats-tile-label">{label}</div>
    {hint && <div className="stats-tile-hint">{hint}</div>}
  </div>
);

const formatDay = (iso) => {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
};

export const DailyActivityChart = ({ data }) => {
  const points = (data || []).map(p => ({
    day: formatDay(p.day),
    activities: p.activity_count,
    users: p.unique_users,
  }));

  if (points.length === 0) {
    return <div className="stats-empty">No activity recorded for this period.</div>;
  }

  return (
    <ResponsiveContainer width="100%" height={280}>
      <LineChart data={points} margin={{ top: 8, right: 16, bottom: 0, left: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
        <XAxis dataKey="day" stroke="var(--text-primary)" tick={{ fontSize: 12 }} />
        <YAxis allowDecimals={false} stroke="var(--text-primary)" tick={{ fontSize: 12 }} />
        <Tooltip
          contentStyle={{ background: 'var(--tooltip-bg)', border: '1px solid var(--tooltip-border)', color: 'var(--tooltip-text)' }}
        />
        <Legend />
        <Line type="monotone" dataKey="activities" name="Activities" stroke={ACTIVITY_COLOR} strokeWidth={2} dot={false} />
        <Line type="monotone" dataKey="users" name="Unique users" stroke={USERS_COLOR} strokeWidth={2} dot={false} />
      </LineChart>
    </ResponsiveContainer>
  );
};

const WEEKDAY_LABELS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

export const ActivityHeatmap = ({ cells }) => {
  const grid = {};
  let max = 0;
  (cells || []).forEach(c => {
    grid[`${c.weekday}-${c.hour}`] = c.pings;
    if (c.pings > max) max = c.pings;
  });

  if (max === 0) {
    return <div className="stats-empty">No activity recorded for this period.</div>;
  }

  return (
    <div className="stats-heatmap">
      <div className="stats-heatmap-hours">
        <div className="stats-heatmap-corner" />
        {Array.from({ length: 24 }, (_, h) => (
          <div key={h} className="stats-heatmap-hour-label">{h % 3 === 0 ? h : ''}</div>
        ))}
      </div>
      {WEEKDAY_LABELS.map((label, weekday) => (
        <div className="stats-heatmap-row" key={weekday}>
          <div className="stats-heatmap-day-label">{label}</div>
          {Array.from({ length: 24 }, (_, hour) => {
            const count = grid[`${weekday}-${hour}`] || 0;
            const opacity = count === 0 ? 0 : 0.15 + 0.85 * (count / max);
            return (
              <div
                key={hour}
                className="stats-heatmap-cell"
                title={`${label} ${hour}:00 — ${count} ping${count === 1 ? '' : 's'}`}
                style={{ backgroundColor: `color-mix(in srgb, var(--blue) ${Math.round(opacity * 100)}%, transparent)` }}
              />
            );
          })}
        </div>
      ))}
    </div>
  );
};
