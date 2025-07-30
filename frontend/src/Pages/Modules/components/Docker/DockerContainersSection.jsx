import React, { useEffect, useState } from 'react';
import Button from 'Global/Button';
import LogViewer from '../LogViewer';
import './DockerContainersSection.css';

export default function DockerContainers({ moduleId }) {
  const [containers, setContainers] = useState([]);
  const [selectedName, setSelectedName] = useState(null);

  useEffect(() => {
    fetchContainers();
  }, [moduleId]);

  const fetchContainers = () => {
    fetch(`http://localhost:8080/api/v1/modules/${moduleId}/containers`)
      .then(res => res.json())
      .then(setContainers)
      .catch(err => console.error('Failed to fetch containers:', err));
  };

  const handleAction = (name, action) => {
    fetch(
      `http://localhost:8080/api/v1/modules/${moduleId}/containers/${name}/${action}`,
      { method: action === 'delete' ? 'DELETE' : 'POST' }
    )
      .then(fetchContainers)
      .catch(err => console.error(`${action} failed:`, err));
  };

  return (
    <div className="docker-containers">
      <table className="container-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Status</th>
            <th>Since</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {containers.map(c => (
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
                    <Button label="Start" onClick={(e) => { e.stopPropagation(); handleAction(c.name, 'start'); }} />
                  )}
                  {c.status === 'running' && (
                    <Button label="Stop" onClick={(e) => { e.stopPropagation(); handleAction(c.name, 'stop'); }} />
                  )}
                  <Button label="Restart" onClick={(e) => { e.stopPropagation(); handleAction(c.name, 'restart'); }} />
                  <Button label="Delete" danger onClick={(e) => { e.stopPropagation(); handleAction(c.name, 'delete'); }} />
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {selectedName && (
        <div className="log-viewer-wrapper">
          {/* <LogViewer logType="module" moduleId={moduleId}/> */}
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
  const color = {
    running: 'green',
    exited: 'gray',
    restarting: 'orange',
    paused: 'blue',
    unknown: 'red',
  }[status] || 'black';

  return <span style={{ color }}>{status}</span>;
}
