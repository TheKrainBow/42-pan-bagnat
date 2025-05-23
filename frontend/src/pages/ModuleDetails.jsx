import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import './ModuleDetails.css';
import { AppIcon } from '../components/AppIcon';
import { RoleBadge } from './Roles';

const ModuleDetails = () => {
  const { moduleId } = useParams();
  const [module, setModule] = useState(null);
  const [loading, setLoading] = useState(true);

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

  if (loading) return <div className="loading">Loading...</div>;
  if (!module) return <div className="error">Module not found.</div>;

  return (
    <div className="module-detail-container">
      <Link to="/modules" className="back-button">← Back to Modules</Link>

      <div className="module-detail-header">
        <AppIcon app={{ icon_url: module.icon_url, name: module.name }} fallback="/icons/modules.png" />
        <div className="module-header-text">
          <h2>{module.name}</h2>
          <span className={`status-badge ${module.status}`}>{module.status.toUpperCase()}</span>
        </div>
      </div>

      <div className="module-info-section">
        <div><span>📦</span><strong>Version:</strong> {module.version}</div>
        <div><span>🔄</span><strong>Latest:</strong> {module.latest_version}</div>
        <div><span>🧱</span><strong>Late Commits:</strong> {module.late_commits}</div>
        <div><span>🕒</span><strong>Updated:</strong> {new Date(module.last_update).toLocaleString()}</div>
        <div><span>🔗</span><strong>Repo:</strong> <a className="module-link" href={module.url} target="_blank" rel="noreferrer">{module.url}</a></div>
      </div>

      <div className="module-roles-section">
        <strong>🎭 Roles:</strong>
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
    </div>
  );
};

export default ModuleDetails;
