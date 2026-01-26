import React, { useEffect, useState } from 'react';
import Button from 'Global/Button/Button';
import LogViewer from '../../../../../Global/LogViewer/LogViewer';
import ContainersGraph from '../ContainersGraph/ContainersGraph';
import './DockerContainersSection.css';
import { fetchWithAuth } from 'Global/utils/Auth';
import { socketService } from 'Global/SocketService/SocketService';

const STATUS_LABELS = {
  running: 'running',
  restarted: 'running',
  restarting: 'restarting',
  exited: 'stopped',
  stopped: 'stopped',
  dead: 'died',
  paused: 'paused',
  created: 'never started',
  unknown: 'unknown',
  building: 'building',
};

const STATUS_COLORS = {
  running: '#4caf50',
  restarted: '#4caf50',
  restarting: '#ffb300',
  exited: '#9aa0a6',
  stopped: '#9aa0a6',
  dead: '#f06292',
  paused: '#4fc3f7',
  created: '#bdbdbd',
  unknown: '#e87171',
  building: '#fdd835',
};

const normalizeStatusKey = (status) => {
  if (!status && status !== 0) return '';
  return String(status).trim().toLowerCase();
};

const normalizeContainer = (input) => {
  if (!input) return null;
  const name = input.name || input.Name || '';
  if (!name) return null;
  return {
    name,
    status: normalizeStatusKey(input.status || input.Status) || 'unknown',
    since: input.since || input.Since || '',
    reason: input.reason || input.Reason || '',
  };
};

const normalizeContainers = (items) => {
  if (!Array.isArray(items)) return [];
  return items.map(normalizeContainer).filter(Boolean);
};

const getStatusLabel = (status) => {
  const key = normalizeStatusKey(status);
  if (!key) return 'unknown';
  return STATUS_LABELS[key] || key;
};

const getStatusColor = (status) => {
  const key = normalizeStatusKey(status);
  if (!key) return '#9aa0a6';
  return STATUS_COLORS[key] || '#9aa0a6';
};

export default function DockerContainers({ moduleId }) {
  const [containers, setContainers] = useState([]);
  const [selectedName, setSelectedName] = useState("");

  useEffect(() => {
    fetchContainers();
  }, [moduleId]);

  const fetchContainers = () => {
    fetchWithAuth(`/api/v1/admin/modules/${moduleId}/docker/ls`)
      .then(res => res.json())
      .then(data => setContainers(normalizeContainers(data)))
      .catch(err => console.error('Failed to fetch containers:', err));
  };

  // Subscribe to WS container updates for live list
  useEffect(() => {
    const topic = `containers:${moduleId}`;
    socketService.subscribeTopic(topic);
    const unsub = socketService.subscribe(msg => {
      if (msg.eventType === 'containers_updated' && msg.module_id === moduleId && Array.isArray(msg.payload)) {
        setContainers(normalizeContainers(msg.payload));
      }
    });
    return () => { socketService.unsubscribeTopic(topic); unsub(); };
  }, [moduleId]);

  useEffect(() => {
    const unsub = socketService.subscribe(msg => {
      if (msg.eventType !== 'container_status' || msg.module_id !== moduleId) {
        return;
      }
      const normalized = normalizeContainer(msg.payload);
      if (!normalized) return;
      setContainers(prev => {
        const list = Array.isArray(prev) ? [...prev] : [];
        const idx = list.findIndex(item => item.name === normalized.name);
        if (idx >= 0) {
          list[idx] = { ...list[idx], ...normalized };
        } else {
          list.push(normalized);
        }
        return list;
      });
    });
    return () => unsub();
  }, [moduleId]);

  useEffect(() => {
    if (selectedName && !containers.find(c => c.name === selectedName)) {
      setSelectedName("");
    }
  }, [containers, selectedName]);

  const handleAction = (name, action) => {
    fetchWithAuth(
      `/api/v1/admin/modules/${moduleId}/docker/${name}/${action}`,
      { method: action === 'delete' ? 'DELETE' : 'POST' }
    )
      .then(fetchContainers)
      .catch(err => console.error(`${action} failed:`, err));
  };
  return (
    <div className="docker-containers">
      <ContainersGraph moduleId={moduleId} />
      <table className="container-table" style={{ display: 'none' }}>
        <thead>
          <tr>
            <th>Name</th>
            <th>Status</th>
            <th>Since</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {!containers || containers.length === 0 ? (
            <tr>
              <td colSpan="4" >
                <div className="no-pages">No containers found.</div>
              </td>
            </tr>
          ) : (
            containers.map(c => (
              <tr
                key={c.name}
                onClick={() => setSelectedName(c.name)}
                className={selectedName === c.name ? 'selected' : ''}
              >
                <td>{c.name}</td>
                <td><StatusBadge status={c.status} /></td>
                <td>{c.since || '—'}</td>
                <td>
                  <div className="action-buttons">
                    {c.status === 'exited' && (
                      <Button icon="/icons/button-play.png" color="warning" onClick={(e) => { e.stopPropagation(); handleAction(c.name, 'start'); }} />
                    )}
                    {c.status === 'running' && (
                      <Button icon="/icons/button-stop.png" color="warning" onClick={(e) => { e.stopPropagation(); handleAction(c.name, 'stop'); }} />
                    )}
                    <Button icon="/icons/button-refresh.png" color="warning" onClick={(e) => { e.stopPropagation(); handleAction(c.name, 'restart'); }} />
                    <Button icon="/icons/button-delete.png" color="warning" onClick={(e) => { e.stopPropagation(); handleAction(c.name, 'delete'); }} />
                  </div>
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
      {selectedName && (
        <div className="log-viewer-wrapper">
          <LogViewer
            logType="container"
            moduleId={moduleId}
            containerName={selectedName}
          />
        </div>
      )}
    </div>
  );

}

function StatusBadge({ status }) {
  const label = getStatusLabel(status);
  const color = getStatusColor(status);
  return <span className="container-status-badge" style={{ color }}>{label}</span>;
}
