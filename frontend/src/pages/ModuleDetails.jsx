// ModuleDetails.jsx
import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import './ModuleDetails.css';
import { AppIcon } from '../components/AppIcon';
import { RoleBadge } from '../components/RoleBadge';

const ModuleDetails = () => {
  const { moduleId } = useParams();
  const [module, setModule] = useState(null);
  const [loading, setLoading] = useState(true);
  const [statusUpdating, setStatusUpdating] = useState(false);
  const [activeTab, setActiveTab] = useState('logs');

  useEffect(() => {
    const fetchModule = async () => {
      try {
        const res = await fetch(`http://localhost:8080/api/v1/modules/${moduleId}`);
        const data = await res.json();
        setModule(data);
      } catch (err) {
        console.error(err);
        setModule(null);
      } finally {
        setLoading(false);
      }
    };

    fetchModule();
  }, [moduleId]);

  const toggleModuleStatus = async () => {
    if (!module) return;
    setStatusUpdating(true);
    try {
      const newStatus = module.status === 'enabled' ? 'disabled' : 'enabled';
      setModule({ ...module, status: newStatus }); // optimistic update
    } catch (err) {
      console.error(err);
    } finally {
      setStatusUpdating(false);
    }
  };

  if (loading) return <div className="loading">Loading...</div>;
  if (!module) return <div className="error">Module not found.</div>;

  return (
    <div className="module-detail-container">
      <Link to="/modules" className="back-button">← Back to Modules</Link>

      <div className="module-header">
        <AppIcon app={{ icon_url: module.icon_url, name: module.name }} fallback="/icons/modules.png" />
        <h2>{module.name}</h2>
        <span className={`status-badge ${module.status}`}>{module.status.toUpperCase()}</span>
      </div>

      {/* Version Info */}
      <div className="module-version-section">
        <div className="version-info">
          <div><strong>📦 Version:</strong> {module.version}</div>
          <div><strong>🔄 Latest:</strong> {module.latest_version}</div>
          <div><strong>🧱 Late Commits:</strong> {module.late_commits}</div>
          <div><strong>🕒 Last Update:</strong> {new Date(module.last_update).toLocaleString()}</div>
          <div><strong>🔗 Repo:</strong> <a className="module-link" href={module.url} target="_blank" rel="noreferrer">{module.url}</a></div>
        </div>
        <div className="version-actions">
          <button className="update-btn">Update Module</button>
          <button className="uninstall-btn">Uninstall Module</button>
        </div>
      </div>

      {/* Running Info */}
      <div className="module-running-section">
        <div className="module-status">
          <strong>Status:</strong>
            <label className="switch">
              <input
                type="checkbox"
                checked={module.status === 'enabled'}
                onChange={toggleModuleStatus}
                disabled={statusUpdating}
              />
              <span className="slider round"></span>
            </label>
          <label className="switch-label">
            <span>Toggle</span>
          </label>
        </div>

        <div className="tabs">
          <div className={`tab ${activeTab === 'logs' ? 'active' : ''}`} onClick={() => setActiveTab('logs')}>Logs</div>
          <div className={`tab ${activeTab === 'settings' ? 'active' : ''}`} onClick={() => setActiveTab('settings')}>Settings</div>
        </div>

        <div className="tab-content">
          {activeTab === 'logs' ? (
            <pre className="log-box">[Sample logs will be displayed here]</pre>
          ) : (
            <div>
              <strong>Linked Roles:</strong>
              <div className="role-badges">
                {Array.isArray(module.roles) && module.roles.length > 0 ? (
                  module.roles.map(role => (
                    <RoleBadge key={role.id} hexColor={role.color}>{role.name}</RoleBadge>
                  ))
                ) : (
                  <span style={{ opacity: 0.5 }}>No roles assigned</span>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ModuleDetails;
