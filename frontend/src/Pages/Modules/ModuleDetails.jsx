// ModuleDetails.jsx
import React, { useContext, useEffect, useState, useRef } from 'react';
import { useParams, Link } from 'react-router-dom';
import './ModuleDetails.css';
import AppIcon from 'Global/AppIcon';
import Button from 'Global/Button';
import LogViewer from 'Pages/Modules/components/LogViewer';
import ModuleSettings from 'Modules/components/ModuleSettings';
import ModuleWarningSection from 'Modules/components/ModuleWarningSection';
import ModuleStatusBadge from 'Modules/components/ModuleStatusBadge';
import ModuleDockerSection from './components/Docker/ModuleDockerSection';
import { setModuleStatusUpdater } from 'Global/SocketService';

const ModuleDetails = () => {
  const { moduleId } = useParams();
  const [module, setModule] = useState(null);
  const [loading, setLoading] = useState(true);
  const [statusUpdating, setStatusUpdating] = useState(false);
  const [activeTab, setActiveTab] = useState('logs');
  const [showWarning, setShowWarning] = useState(false);
  const fetchedRef = useRef(false);

  useEffect(() => {
    // Register live update handler
    setModuleStatusUpdater((id, newStatus) => {
      if (id === moduleId) {
        setModule(prev => ({ ...prev, status: newStatus }));
      }
    });

    return () => {
      // Unregister when component unmounts
      setModuleStatusUpdater(null);
    };
  }, [moduleId]);

  const fetchModule = async () => {
    try {
      const res = await fetch(`/api/v1/modules/${moduleId}`);
      const data = await res.json();
      setModule(data);
    } catch (err) {
      console.error(err);
      setModule(null);
    } finally {
      setLoading(false);
    }
  };

  const handleUninstall = () => {
    fetch(`/api/v1/modules/${moduleId}`, { method: 'DELETE' })
      .catch(err => console.error('Failed to uninstall:', err));
  };

  const handleAfterRetry = () => {
    fetchModule();            // re‐fetch module info
  };

  useEffect(() => {
    if (fetchedRef.current) return;
    fetchedRef.current = true;
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

      {/* Running Info */}
      <div className="module-running-section">
        <div className="tabs">
          <Button
            label="Docker"
            className={`custom-btn ${activeTab === 'docker' ? 'blue' : 'gray'}`}
            onClick={() => setActiveTab('docker')}
          />
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
          {activeTab === 'logs' && <LogViewer logType="module" moduleId={module.id}/>}
          {activeTab === 'settings' && <ModuleSettings
              module={module}
              statusUpdating={statusUpdating}
              onToggleStatus={toggleModuleStatus}
              onUninstall={handleUninstall}
            />}
          {activeTab === 'docker' && <ModuleDockerSection moduleId={module.id}/>}
        </div>
      </div>
    </div>
  );
};

export default ModuleDetails;
