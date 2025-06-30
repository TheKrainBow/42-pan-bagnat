// ModuleDetails.jsx
import React, { useEffect, useState, useRef } from 'react';
import { useParams, Link } from 'react-router-dom';
import './ModuleDetails.css';
import AppIcon from 'Global/AppIcon';
import Button from 'Global/Button';
import ModuleLogs from 'Modules/components/ModuleLogs';
import ModuleSettings from 'Modules/components/ModuleSettings';
import ModuleWarningSection from 'Modules/components/ModuleWarningSection';
import ModuleStatusBadge from 'Modules/components/ModuleStatusBadge';
import ModuleAboutSection from './components/ModuleAboutSection';
import ModuleConfigViewer from './components/ModuleConfigPanel';

const ModuleDetails = () => {
  const { moduleId } = useParams();
  const [module, setModule] = useState(null);
  const [loading, setLoading] = useState(true);
  const [statusUpdating, setStatusUpdating] = useState(false);
  const [activeTab, setActiveTab] = useState('logs');
  const [showWarning, setShowWarning] = useState(false);
  const logsRef = useRef();

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

  const handleAfterRetry = () => {
    fetchModule();            // re‐fetch module info
    logsRef.current?.refresh(); // reload logs
  };

  useEffect(() => {
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

  useEffect(() => {
    if (module) {
      const cloned = module.last_update && new Date(module.last_update).getFullYear() > 2000;
      setShowWarning(!cloned);
    }
  }, [module]);

  if (loading) return <div className="loading">Loading...</div>;
  if (!module) return <div className="error">Module not found.</div>;

  const isCloned = module.last_update && new Date(module.last_update).getFullYear() > 2000;

  return (
    <div className="module-detail-container">
      <Link to="/admin/modules" className="custom-btn link">
        <img src="/icons/arrow.png" alt="Back" style={{ width: "16px", marginRight: "8px", marginLeft: "-5px", verticalAlign: "middle" }} />
        Back to Modules
      </Link>

      <div className="module-header">
        <AppIcon app={{ icon_url: module.icon_url, name: module.name }} fallback="/icons/modules.png" />
        <h2>{module.name}</h2>
        <ModuleStatusBadge status={module.status} />
      </div>

      {/* Version Info */}
      {showWarning && (
        <ModuleWarningSection
          sshKey={module.ssh_public_key}
          moduleID={module.id}
          onRetrySuccess={() => {
            setShowWarning(false);
            fetchModule();
          }}
          onRetry={handleAfterRetry}
        />
      )}
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
            label="Config"
            className={`custom-btn ${activeTab === 'config' ? 'blue' : 'gray'}`}
            onClick={() => setActiveTab('config')}
          />
          <Button
            label="Settings"
            className={`custom-btn ${activeTab === 'settings' ? 'blue' : 'gray'}`}
            onClick={() => setActiveTab('settings')}
          />
        </div>

        <div className="tab-content">
          {activeTab === 'logs' && <ModuleLogs ref={logsRef} moduleId={module.id}/>}
          {activeTab === 'settings' && <ModuleSettings
              roles={module.roles}
              status={module.status}
              onToggleStatus={toggleModuleStatus}
              statusUpdating={statusUpdating}
            />}
          {activeTab === 'config' && <ModuleConfigViewer moduleId={module.id}/>}
        </div>
      </div>
    </div>
  );
};

export default ModuleDetails;
