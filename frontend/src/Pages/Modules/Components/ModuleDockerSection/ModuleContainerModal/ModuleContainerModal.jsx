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

const MAX_BODY_SIZE_RE = /^[1-9][0-9]*[kKmMgG]?$/;
const MAX_BODY_SIZE_CAP_BYTES = 2 * 1024 * 1024 * 1024; // 2 GiB, mirrors the backend's hard ceiling
const DEFAULT_PROXY_TIMEOUT_SECONDS = 60;
const MIN_PROXY_TIMEOUT_SECONDS = 1;
const MAX_PROXY_TIMEOUT_SECONDS = 600;
const MAX_RATE_LIMIT_RPS = 1000;
const MAX_RATE_LIMIT_BURST = 10000;

function maxBodySizeToBytes(value) {
  const match = /^([1-9][0-9]*)([kKmMgG]?)$/.exec(value);
  if (!match) return 0;
  const n = Number(match[1]);
  const unit = match[2].toLowerCase();
  const multiplier = unit === 'k' ? 1024 : unit === 'm' ? 1024 * 1024 : unit === 'g' ? 1024 * 1024 * 1024 : 1;
  return n * multiplier;
}

// Mirrors modules-proxy/cmd/net-controller's gatewayCommand(): renders the
// nginx location block that will actually be generated for this page's
// gateway container, so admins can see the effect of their settings before
// applying them.
function buildNginxPreview({
  containerName,
  port,
  gatewayPort,
  maxUploadBodySize,
  proxyTimeoutSeconds,
  rateLimitRPS,
  rateLimitBurst,
  disableRequestBuffering,
}) {
  const target = containerName && port ? `http://${containerName}:${port}/` : 'http://<container>:<port>/';
  const rps = Number(rateLimitRPS) || 0;
  const burst = Number(rateLimitBurst) || 0;
  const timeout = Number(proxyTimeoutSeconds) || DEFAULT_PROXY_TIMEOUT_SECONDS;
  const bodySize = maxUploadBodySize || '1m';

  const lines = [];
  lines.push('http {');
  lines.push('    map $http_upgrade $connection_upgrade { ... }');
  lines.push('    map $host $module_target_host { ... }');
  if (rps > 0) {
    lines.push(`    limit_req_zone $binary_remote_addr zone=gw_limit:10m rate=${rps}r/s;`);
  }
  lines.push(`    server {`);
  lines.push(`        listen ${gatewayPort};`);
  lines.push('        location / {');
  lines.push(`            client_max_body_size ${bodySize};`);
  if (rps > 0) {
    lines.push(`            limit_req zone=gw_limit burst=${burst} nodelay;`);
  }
  lines.push(`            proxy_read_timeout ${timeout}s;`);
  lines.push(`            proxy_send_timeout ${timeout}s;`);
  if (disableRequestBuffering) {
    lines.push('            proxy_request_buffering off;');
  }
  lines.push(`            proxy_pass ${target};`);
  lines.push('            proxy_http_version 1.1;');
  lines.push('            proxy_set_header Host $module_target_host;');
  lines.push('            proxy_set_header X-Forwarded-Host $host;');
  lines.push('            ...');
  lines.push('            proxy_buffering off;');
  lines.push('        }');
  lines.push('    }');
  lines.push('}');
  return lines.join('\n');
}

export default function ModuleContainerModal({
  open,
  containers,
  networks,
  value,
  gatewayPort = 8080,
  onClose,
  onSave,
}) {
  const [containerName, setContainerName] = useState(value?.targetContainer || '');
  const [portValue, setPortValue] = useState(
    typeof value?.targetPort === 'number' ? String(value.targetPort) : ''
  );
  const [networkName, setNetworkName] = useState(value?.network || '');
  const [maxUploadBodySize, setMaxUploadBodySize] = useState(value?.maxUploadBodySize || '1m');
  const [proxyTimeoutSeconds, setProxyTimeoutSeconds] = useState(
    String(value?.proxyTimeoutSeconds || DEFAULT_PROXY_TIMEOUT_SECONDS)
  );
  const [rateLimitRPS, setRateLimitRPS] = useState(String(value?.rateLimitRPS || 0));
  const [rateLimitBurst, setRateLimitBurst] = useState(String(value?.rateLimitBurst || 0));
  const [disableRequestBuffering, setDisableRequestBuffering] = useState(!!value?.disableRequestBuffering);
  const [advancedOpen, setAdvancedOpen] = useState(false);

  useEffect(() => {
    if (!open) return;
    setContainerName(value?.targetContainer || '');
    setPortValue(typeof value?.targetPort === 'number' ? String(value.targetPort) : '');
    setNetworkName(value?.network || '');
    setMaxUploadBodySize(value?.maxUploadBodySize || '1m');
    setProxyTimeoutSeconds(String(value?.proxyTimeoutSeconds || DEFAULT_PROXY_TIMEOUT_SECONDS));
    setRateLimitRPS(String(value?.rateLimitRPS || 0));
    setRateLimitBurst(String(value?.rateLimitBurst || 0));
    setDisableRequestBuffering(!!value?.disableRequestBuffering);
    setAdvancedOpen(false);
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
    // Only auto-pick when there's a single unambiguous option; otherwise
    // force an explicit choice instead of silently guessing.
    setPortValue(availablePorts.length === 1 ? String(availablePorts[0].container_port) : '');
  }, [open, containerName, availablePorts, portValue]);

  useEffect(() => {
    if (!open) return;
    if (networkName) return;
    if (networks.length === 1) {
      setNetworkName(networks[0]);
    }
  }, [open, networkName, networks]);

  if (!open) return null;

  const previewPort = availablePorts.find((port) => String(port.container_port) === portValue) || null;
  const summary = formatSummary(containerName, previewPort, networkName.trim());

  const trimmedMaxBodySize = maxUploadBodySize.trim();
  const maxBodySizeInvalid =
    trimmedMaxBodySize !== '' &&
    (!MAX_BODY_SIZE_RE.test(trimmedMaxBodySize) || maxBodySizeToBytes(trimmedMaxBodySize.toLowerCase()) > MAX_BODY_SIZE_CAP_BYTES);

  const timeoutNum = Number(proxyTimeoutSeconds);
  const timeoutInvalid =
    proxyTimeoutSeconds.trim() !== '' &&
    (!Number.isInteger(timeoutNum) || timeoutNum < MIN_PROXY_TIMEOUT_SECONDS || timeoutNum > MAX_PROXY_TIMEOUT_SECONDS);

  const rpsNum = Number(rateLimitRPS);
  const rpsInvalid =
    rateLimitRPS.trim() !== '' && (!Number.isInteger(rpsNum) || rpsNum < 0 || rpsNum > MAX_RATE_LIMIT_RPS);

  const burstNum = Number(rateLimitBurst);
  const burstInvalid =
    rateLimitBurst.trim() !== '' && (!Number.isInteger(burstNum) || burstNum < 0 || burstNum > MAX_RATE_LIMIT_BURST);

  const hasError = maxBodySizeInvalid || timeoutInvalid || rpsInvalid || burstInvalid;

  const nginxPreview = buildNginxPreview({
    containerName: containerName.trim(),
    port: previewPort?.container_port || null,
    gatewayPort,
    maxUploadBodySize: trimmedMaxBodySize || '1m',
    proxyTimeoutSeconds: timeoutNum || DEFAULT_PROXY_TIMEOUT_SECONDS,
    rateLimitRPS: rpsNum || 0,
    rateLimitBurst: burstNum || 0,
    disableRequestBuffering,
  });

  const handleSave = () => {
    if (hasError) return;
    onSave?.({
      targetContainer: containerName.trim() || '',
      targetPort: portValue ? Number(portValue) : null,
      network: networkName.trim() || '',
      maxUploadBodySize: trimmedMaxBodySize || '1m',
      proxyTimeoutSeconds: timeoutNum || DEFAULT_PROXY_TIMEOUT_SECONDS,
      rateLimitRPS: rpsNum || 0,
      rateLimitBurst: burstNum || 0,
      disableRequestBuffering,
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

        <button
          type="button"
          className="module-container-advanced-toggle"
          onClick={() => setAdvancedOpen((v) => !v)}
        >
          {advancedOpen ? '▾' : '▸'} Advanced settings (gateway)
          {!advancedOpen && hasError && <span className="field-warning" title="Invalid advanced setting">!</span>}
        </button>

        {advancedOpen && (
          <div className="module-container-advanced">
            <div className="module-container-modal-grid module-container-modal-grid--advanced">
              <label className="module-container-field">
                <span>Max upload size</span>
                <input
                  type="text"
                  className={maxBodySizeInvalid ? 'field-invalid' : ''}
                  placeholder="1m"
                  value={maxUploadBodySize}
                  onChange={(e) => setMaxUploadBodySize(e.target.value)}
                />
              </label>

              <label className="module-container-field">
                <span>Proxy timeout (s)</span>
                <input
                  type="text"
                  className={timeoutInvalid ? 'field-invalid' : ''}
                  placeholder={String(DEFAULT_PROXY_TIMEOUT_SECONDS)}
                  value={proxyTimeoutSeconds}
                  onChange={(e) => setProxyTimeoutSeconds(e.target.value)}
                />
              </label>

              <label className="module-container-field">
                <span>Rate limit (req/s)</span>
                <input
                  type="text"
                  className={rpsInvalid ? 'field-invalid' : ''}
                  placeholder="0 = disabled"
                  value={rateLimitRPS}
                  onChange={(e) => setRateLimitRPS(e.target.value)}
                />
              </label>

              <label className="module-container-field">
                <span>Rate limit burst</span>
                <input
                  type="text"
                  className={burstInvalid ? 'field-invalid' : ''}
                  placeholder="0"
                  value={rateLimitBurst}
                  onChange={(e) => setRateLimitBurst(e.target.value)}
                  disabled={!rpsNum}
                />
              </label>

              <label className="module-container-field module-container-field--checkbox" title="Streams uploads directly to the module instead of buffering the full request body first">
                <span>Stream uploads</span>
                <div className="module-container-field--checkbox-row">
                  <input
                    type="checkbox"
                    checked={disableRequestBuffering}
                    onChange={(e) => setDisableRequestBuffering(e.target.checked)}
                  />
                  <span className="module-container-field--checkbox-hint">no request buffering</span>
                </div>
              </label>
            </div>

            {(maxBodySizeInvalid || timeoutInvalid || rpsInvalid || burstInvalid) && (
              <div className="module-container-modal-error">
                {maxBodySizeInvalid && <div>Max upload size: nombre suivi de k, m ou g, 2g maximum (ex. "10m").</div>}
                {timeoutInvalid && <div>Proxy timeout: entier entre {MIN_PROXY_TIMEOUT_SECONDS} et {MAX_PROXY_TIMEOUT_SECONDS}.</div>}
                {rpsInvalid && <div>Rate limit: entier entre 0 et {MAX_RATE_LIMIT_RPS}.</div>}
                {burstInvalid && <div>Rate limit burst: entier entre 0 et {MAX_RATE_LIMIT_BURST}.</div>}
              </div>
            )}

            <div className="module-container-preview">
              <div className="module-container-preview-title">nginx.conf preview (gateway-{containerName ? containerName : '…'})</div>
              <pre className="module-container-preview-code">{nginxPreview}</pre>
            </div>
          </div>
        )}

        <div className="modal-actions">
          <Button label="Cancel" color="gray" onClick={onClose} />
          <Button label="Apply" color="green" onClick={handleSave} disabled={hasError} />
        </div>
      </div>
    </div>
  );
}
