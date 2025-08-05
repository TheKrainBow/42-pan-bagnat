// ModuleDetails.jsx
import React, { useEffect, useState, useRef } from 'react';
import { useNavigate, useSearchParams, useParams, Link } from 'react-router-dom';
import './ModuleDetails.css';
import AppIcon from 'Global/AppIcon/AppIcon';
import Button from 'Global/Button/Button';
import LogViewer from 'Global/LogViewer/LogViewer';
import ModuleSettings from 'Pages/Modules/Components/ModuleSettings/ModuleSettings';
import ModuleWarningSection from 'Pages/Modules/Components/ModuleWarningSection/ModuleWarningSection';
import ModuleStatusBadge from 'Pages/Modules/Components/ModuleStatusBadge/ModuleStatusBadge';
import ModuleDockerSection from '../Components/ModuleDockerSection/ModuleDockerSection';
import { setModuleStatusUpdater } from 'Global/SocketService/SocketService';
import ModuleUninstallModal from 'Pages/Modules/Components/ModuleUninstallModal/ModuleUninstallModal';
import { fetchWithAuth } from 'Global/utils/Auth';

const ModuleDetails = () => {
  const { moduleId } = useParams();
  const [module, setModule] = useState(null);
  const [loading, setLoading] = useState(true);
  const [statusUpdating, setStatusUpdating] = useState(false);
  const [showWarning, setShowWarning] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();
  const [showConfirmUninstall, setShowConfirmUninstall] = useState(false);

  const tab = searchParams.get('tab') || 'logs';
  const subtab = searchParams.get('subtab') || 'compose';

  const [activeTab, setActiveTab] = useState(tab);
  const [activeSubTab, setActiveSubTab] = useState(subtab);
  const fetchedRef = useRef(false);

  const retryButtonRef = useRef();

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
      const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}`);
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
    setShowConfirmUninstall(false);
    setActiveTab("logs")
    fetchWithAuth(`/api/v1/admin/modules/${moduleId}`, { method: 'DELETE' })
      .catch(err => console.error('Failed to uninstall:', err));
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

  // useEffect(() => {
  //   if (!module) {
  //     useNavigate('/admin/modules');
  //   }
  // }, [module]);

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
            ref={retryButtonRef}
            sshKey={module.ssh_public_key}
            moduleID={module.id}
            onRetrySuccess={() => {
              setShowWarning(false);
              fetchModule();
            }}
          />
        )}

        {/* Running Info */}
        <div className="module-running-section">
          <div className="tabs">
            <Button
              label="Logs"
              className={`custom-btn ${activeTab === 'logs' ? 'blue' : 'gray'}`}
              onClick={() => {
                setActiveTab('logs'); // or 'docker', 'settings'
                setSearchParams({ tab: 'logs' }); // or the corresponding value
                triggerAttention();
              }}
            />
            <Button
              label="Docker"
              color={`${activeTab === 'docker' ? 'blue' : 'gray'}`}
              onClick={() => {
                setActiveTab('docker'); // or 'docker', 'settings'
                setSearchParams({ tab: 'docker' }); // or the corresponding value
              }}
              disabled={showWarning}
              disabledMessage={"You must resolve git issues first"}
              onClickDisabled={() => {
                retryButtonRef.current?.callToAction();
              }}
            />
            <Button
              label="Settings"
              color={`${activeTab === 'settings' ? 'blue' : 'gray'}`}
              onClick={() => {
                setActiveTab('settings'); // or 'docker', 'settings'
                setSearchParams({ tab: 'settings' }); // or the corresponding value
              }}
            />
          </div>

          <div className="tab-content">
            {activeTab === 'logs' && <LogViewer logType="module" moduleId={module.id}/>}
            {activeTab === 'settings' && <ModuleSettings
                module={module}
                statusUpdating={statusUpdating}
                onToggleStatus={toggleModuleStatus}
                onUninstall={() => setShowConfirmUninstall(true)}
              />}
            {activeTab === 'docker' &&
              <ModuleDockerSection
                moduleId={module.id}
                dockerTab={activeSubTab}
                setDockerTab={(newTab) => {
                  setActiveSubTab(newTab);
                  setSearchParams({ tab: 'docker', subtab: newTab });
                }}
              />}
              {showConfirmUninstall && (
                <ModuleUninstallModal
                  onConfirm={handleUninstall}
                  onCancel={() => setShowConfirmUninstall(false)}
                />
              )}
          </div>
        </div>
      </div>
  );
};

export default ModuleDetails;
