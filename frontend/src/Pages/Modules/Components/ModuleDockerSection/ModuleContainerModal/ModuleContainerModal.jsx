import React, { useEffect, useMemo, useState } from 'react';
import Button from 'Global/Button/Button';
import './ModuleContainerModal.css';

function formatPort(port) {
  if (!port) return '';
  const proto = port.protocol ? `/${port.protocol}` : '';
  return `${port.container_port}${proto}`;
}

function formatSummary(containerName, port, network) {
  if (!containerName) return 'Pas de conteneur';
  const portLabel = port ? formatPort(port) : '—';
  const base = `${containerName}:${portLabel}`;
  return network ? `${base} • ${network}` : base;
}

export default function ModuleContainerModal({
  open,
  containers,
  networks,
  value,
  onClose,
  onSave,
}) {
  const [containerName, setContainerName] = useState(value?.targetContainer || '');
  const [portValue, setPortValue] = useState(
    typeof value?.targetPort === 'number' ? String(value.targetPort) : ''
  );
  const [networkName, setNetworkName] = useState(value?.network || '');

  useEffect(() => {
    if (!open) return;
    setContainerName(value?.targetContainer || '');
    setPortValue(typeof value?.targetPort === 'number' ? String(value.targetPort) : '');
    setNetworkName(value?.network || '');
  }, [open, value]);

  const selectedContainer = useMemo(
    () => containers.find((container) => container.name === containerName) || null,
    [containers, containerName]
  );

  const availablePorts = useMemo(() => {
    if (!selectedContainer || !Array.isArray(selectedContainer.ports)) return [];
    const unique = new Map();
    for (const port of selectedContainer.ports) {
      if (!Number.isInteger(port?.container_port) || port.container_port <= 0) continue;
      const normalized = {
        ...port,
        protocol: String(port.protocol || 'tcp').trim().toLowerCase() || 'tcp',
      };
      const key = `${normalized.container_port}/${normalized.protocol}`;
      if (!unique.has(key)) {
        unique.set(key, normalized);
      }
    }
    return Array.from(unique.values());
  }, [selectedContainer]);

  useEffect(() => {
    if (!open) return;
    if (!containerName) {
      setPortValue('');
      return;
    }
    const current = availablePorts.find((port) => String(port.container_port) === portValue);
    if (current) return;
    const firstPort = availablePorts[0];
    setPortValue(firstPort ? String(firstPort.container_port) : '');
  }, [open, containerName, availablePorts, portValue]);

  if (!open) return null;

  const previewPort = availablePorts.find((port) => String(port.container_port) === portValue) || null;
  const summary = formatSummary(containerName, previewPort, networkName.trim());

  const handleSave = () => {
    onSave?.({
      targetContainer: containerName.trim() || '',
      targetPort: portValue ? Number(portValue) : null,
      network: networkName.trim() || '',
    });
  };

  return (
    <div className="modal-backdrop" onMouseDown={onClose}>
      <div className="modal module-container-modal" onMouseDown={(e) => e.stopPropagation()}>
        <div className="modal-title module-container-modal-title">Container setup</div>
        <div className="module-container-modal-preview">{summary}</div>

        <div className="module-container-modal-grid">
          <label className="module-container-field">
            <span>Container</span>
            <select value={containerName} onChange={(e) => setContainerName(e.target.value)}>
              <option value="">No container</option>
              {containers.map((container) => (
                <option key={container.name} value={container.name}>
                  {container.name}
                </option>
              ))}
            </select>
          </label>

          <label className="module-container-field">
            <span>Port</span>
            <select
              value={portValue}
              onChange={(e) => setPortValue(e.target.value)}
              disabled={!containerName || availablePorts.length === 0}
            >
              <option value="">
                {!containerName
                  ? 'Select a container first'
                  : availablePorts.length === 0
                    ? 'No ports detected'
                    : 'Select a port'}
              </option>
              {availablePorts.map((port, idx) => (
                <option key={`${port.container_port}-${port.host_port || 0}-${port.protocol || 'tcp'}-${idx}`} value={port.container_port}>
                  {formatPort(port)}
                </option>
              ))}
            </select>
          </label>

          <label className="module-container-field">
            <span>Network</span>
            <select value={networkName} onChange={(e) => setNetworkName(e.target.value)}>
              <option value="">No network</option>
              {networks.map((network) => (
                <option key={network} value={network}>
                  {network}
                </option>
              ))}
            </select>
          </label>
        </div>

        <div className="modal-actions">
          <Button label="Cancel" color="gray" onClick={onClose} />
          <Button label="Apply" color="green" onClick={handleSave} />
        </div>
      </div>
    </div>
  );
}
