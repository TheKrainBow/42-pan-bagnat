// ModuleDetails.jsx
import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import './ModuleDetails.css';
import AppIcon from 'Global/AppIcon';
import Button from 'Global/Button';
import ModuleLogs from 'Modules/components/ModuleLogs';
import ModuleSettings from 'Modules/components/ModuleSettings';
import ModuleWarningSection from 'Modules/components/ModuleWarningSection';
import ModuleStatusBadge from 'Modules/components/ModuleStatusBadge';
import ModuleAboutSection from './components/ModuleAboutSection';

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
      setModule({ ...module, status: newStatus });
    } catch (err) {
      console.error(err);
    } finally {
      setStatusUpdating(false);
    }
  };

  if (loading) return <div className="loading">Loading...</div>;
  if (!module) return <div className="error">Module not found.</div>;

  const isCloned = module.last_update && new Date(module.last_update).getFullYear() > 2000;

  return (
    <div className="module-detail-container">
      <Link to="/modules" className="custom-btn link">
        <img src="/icons/arrow.png" alt="Back" style={{ width: "16px", marginRight: "8px", marginLeft: "-5px", verticalAlign: "middle" }} />
        Back to Modules
      </Link>

      <div className="module-header">
        <AppIcon app={{ icon_url: module.icon_url, name: module.name }} fallback="/icons/modules.png" />
        <h2>{module.name}</h2>
        <ModuleStatusBadge status={module.status} />
      </div>

      {/* Version Info */}
      {!isCloned && <ModuleWarningSection sshKey={module.ssh_public_key} moduleID={module.id}> </ModuleWarningSection>}
      <ModuleAboutSection module={module}></ModuleAboutSection>

      {/* Running Info */}
      <div className="module-running-section">
        <div className="tabs">
          <Button
            label="Logs"
            className={`custom-btn ${activeTab === 'logs' ? 'blue' : 'gray'}`}
            onClick={() => setActiveTab('logs')}
          />
          <Button
            label="Settings"
            className={`custom-btn ${activeTab === 'settings' ? 'blue' : 'gray'}`}
            onClick={() => setActiveTab('settings')}
          />
        </div>

        <div className="tab-content">
          {activeTab === 'logs' ? (
            <ModuleLogs />
          ) : (
            <ModuleSettings
              roles={module.roles}
              status={module.status}
              onToggleStatus={toggleModuleStatus}
              statusUpdating={statusUpdating}
            />
          )}
        </div>
      </div>
    </div>
  );
};

export default ModuleDetails;
