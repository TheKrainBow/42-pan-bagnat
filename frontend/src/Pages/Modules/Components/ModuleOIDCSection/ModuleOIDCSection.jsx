import React, { useEffect, useMemo, useState } from 'react';
import Button from 'Global/Button/Button';
import { fetchWithAuth } from 'Global/utils/Auth';
import { toast } from 'react-toastify';
import './ModuleOIDCSection.css';

const SCOPE_LABELS = ['openid', 'profile', 'email', 'roles'];

function CopyIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d="M17 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7l-3-5Zm-5 18a3 3 0 1 1 0-6 3 3 0 0 1 0 6Zm4-12H6V4h10v4Z" />
    </svg>
  );
}

function CopyButton({ onClick, label = 'Copy' }) {
  return (
    <button
      type="button"
      className="oidc-copy-btn"
      onClick={onClick}
      aria-label={label}
      title={label}
    >
      <CopyIcon />
    </button>
  );
}

export default function ModuleOIDCSection({ moduleId }) {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [newRedirectURI, setNewRedirectURI] = useState('');
  const [visibleSecret, setVisibleSecret] = useState('');
  const [showScopesInfo, setShowScopesInfo] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/oidc`);
      if (!res) return;
      const json = await res.json();
      setData(json);
    } catch (err) {
      console.error(err);
      setData(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [moduleId]);

  const patch = async (next) => {
    const payload = {
      enabled: next.enabled,
      allowed_redirect_uris: next.allowed_redirect_uris,
      allowed_scopes: next.allowed_scopes,
    };
    const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/oidc`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res) return null;
    const json = await res.json();
    setData(json);
    return json;
  };

  const copyText = async (text, label) => {
    try {
      await navigator.clipboard.writeText(text);
      toast.success(`${label} copied`);
    } catch {
      toast.error(`Unable to copy ${label.toLowerCase()}`);
    }
  };

  const toggleScope = async (scope) => {
    if (!data) return;
    if (scope === 'openid') return;
    const scopes = new Set(data.allowed_scopes || []);
    if (scopes.has(scope)) scopes.delete(scope);
    else scopes.add(scope);
    await patch({ ...data, allowed_scopes: SCOPE_LABELS.filter((s) => s === 'openid' || scopes.has(s)) });
  };

  const toggleEnabled = async () => {
    if (!data) return;
    await patch({ ...data, enabled: !data.enabled });
  };

  const addRedirectURI = async () => {
    const uri = newRedirectURI.trim();
    if (!uri) return;
    try {
      new URL(uri);
    } catch {
      toast.error('Invalid redirect URI');
      return;
    }
    const next = [...(data?.allowed_redirect_uris || []), uri];
    await patch({ ...data, allowed_redirect_uris: next });
    setNewRedirectURI('');
  };

  const removeRedirectURI = async (uri) => {
    const next = (data?.allowed_redirect_uris || []).filter((item) => item !== uri);
    await patch({ ...data, allowed_redirect_uris: next });
  };

  const generateSecret = async () => {
    const message = hasSecret
      ? 'Rotate the client secret? The old secret will stop working.'
      : 'Generate a new client secret? It will be shown only once.';
    if (!window.confirm(message)) return;
    const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/oidc/secret/rotate`, {
      method: 'POST',
    });
    if (!res) return;
    const json = await res.json();
    setVisibleSecret(json.client_secret || '');
    await load();
  };

  const copyFullConfig = async () => {
    if (!data) return;
    const secretLine = visibleSecret
      ? `Client Secret: ${visibleSecret}`
      : 'Client Secret: hidden - rotate secret to generate a new one';
    const config = [
      `Client ID: ${data.client_id}`,
      secretLine,
      `Issuer: ${data.issuer}`,
      `Authorization URL: ${data.authorization_url}`,
      `Token URL: ${data.token_url}`,
      `User Info URL: ${data.userinfo_url}`,
      `JWKS URL: ${data.jwks_url}`,
      `Scopes: ${(data.allowed_scopes || []).join(' ')}`,
    ].join('\n');
    await copyText(config, 'Full config');
  };

  const endpointRows = useMemo(() => ([
    ['Issuer', data?.issuer],
    ['Discovery URL', data?.discovery_url],
    ['Authorization URL', data?.authorization_url],
    ['Token URL', data?.token_url],
    ['UserInfo URL', data?.userinfo_url],
    ['JWKS URL', data?.jwks_url],
  ]), [data]);

  const userInfoExample = useMemo(() => {
    const scopes = new Set(data?.allowed_scopes || []);
    const payload = {
      sub: 'users_01HZXAMPLESTABLEID',
    };

    if (scopes.has('profile')) {
      payload.name = 'Full Name';
      payload.preferred_username = 'login42';
      payload.picture = 'https://example.com/avatar.png';
    }

    if (scopes.has('email')) {
      payload.email = 'user@example.com';
      payload.email_verified = true;
    }

    if (scopes.has('roles')) {
      payload.module = {
        id: data?.module_id || 'module_01HZXAMPLEMODULE',
        slug: data?.module_slug || 'example-module',
        name: 'Example Module',
      };
      payload.roles = ['PB Admin', 'Student'];
      payload.role_slugs = ['pb-admin', 'student'];
    }

    return JSON.stringify(payload, null, 2);
  }, [data]);

  if (loading) {
    return <div className="oidc-panel">Loading OIDC client...</div>;
  }

  if (!data) {
    return <div className="oidc-panel">OIDC client unavailable.</div>;
  }

  const hasSecret = !!data.has_client_secret;

  return (
    <div className="oidc-panel oidc-scroll">
      <div className="oidc-header">
        <div>
          <h3>OIDC Client</h3>
          <p className="oidc-subtitle">Manage the OpenID Connect client for this module.</p>
        </div>
        <label className="oidc-toggle">
          <span>OIDC Enabled</span>
          <input type="checkbox" checked={!!data.enabled} onChange={toggleEnabled} />
        </label>
      </div>

      <div className="oidc-grid">
        <div className="oidc-card">
          <h4>Client</h4>
          <div className="oidc-field oidc-copy-row">
            <span>Status</span>
            <strong>{data.enabled ? 'Enabled' : 'Disabled'}</strong>
          </div>
          <div className="oidc-field oidc-copy-row">
            <span>Client ID</span>
            <div className="oidc-copy-value">
              <code>{data.client_id}</code>
              <CopyButton onClick={() => copyText(data.client_id, 'Client ID')} label="Copy Client ID" />
            </div>
          </div>
          <div className="oidc-field oidc-copy-row">
            <span>Client Type</span>
            <strong>{data.client_type}</strong>
          </div>
          <div className="oidc-field oidc-copy-row">
            <span>Client Secret status</span>
            <strong>{hasSecret ? 'Generated' : 'Secret non généré'}</strong>
          </div>
          {visibleSecret ? (
            <div className="oidc-secret-inline">
              <div className="oidc-field oidc-copy-row">
                <span>Client Secret</span>
                <div className="oidc-copy-value">
                  <code>{visibleSecret}</code>
                  <CopyButton onClick={() => copyText(visibleSecret, 'Client Secret')} label="Copy Client Secret" />
                </div>
              </div>
              <p>Copy this client secret now. It will not be shown again.</p>
            </div>
          ) : (
            <div className="oidc-field oidc-copy-row">
              <span>Client Secret</span>
              <strong>{hasSecret ? 'Hidden' : 'Secret non généré'}</strong>
            </div>
          )}
          <div className="oidc-field oidc-copy-row">
            <span>Last secret rotation</span>
            <strong>{data.last_secret_rotated_at || 'Never'}</strong>
          </div>
          <div className="oidc-actions">
            <Button
              label={hasSecret ? 'Rotate Secret' : 'Generate Secret'}
              color="blue"
              onClick={generateSecret}
            />
            <Button label="Copy Full Config" color="gray" onClick={copyFullConfig} />
          </div>
        </div>

        <div className="oidc-card">
          <h4>Endpoints</h4>
          {endpointRows.map(([label, value]) => (
            <div className="oidc-field oidc-copy-row" key={label}>
              <span>{label}</span>
              <div className="oidc-copy-value">
                <code>{value}</code>
                <CopyButton onClick={() => copyText(value || '', label)} label={`Copy ${label}`} />
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="oidc-card">
        <div className="oidc-section-header">
          <h4>Redirect URIs</h4>
          {!(data.allowed_redirect_uris || []).length && (
            <span className="oidc-warning">No redirect URI configured. This OIDC client cannot be used yet.</span>
          )}
        </div>
        <div className="oidc-list">
          {(data.allowed_redirect_uris || []).map((uri) => (
            <div className="oidc-list-row" key={uri}>
              <code>{uri}</code>
              <Button label="Remove" color="red" onClick={() => removeRedirectURI(uri)} />
            </div>
          ))}
        </div>
        <div className="oidc-inline-form">
          <input
            type="text"
            placeholder="https://example.com/auth/callback"
            value={newRedirectURI}
            onChange={(e) => setNewRedirectURI(e.target.value)}
          />
          <Button label="Add URI" color="blue" onClick={addRedirectURI} />
        </div>
      </div>

      <div className="oidc-card">
        <div className="oidc-section-header">
          <h4>Scopes</h4>
          <Button
            label="More information"
            color="gray"
            onClick={() => setShowScopesInfo((show) => !show)}
          />
        </div>
        {showScopesInfo && (
          <div className="oidc-scope-help">
            <p><strong>openid</strong>: required. Enables the OIDC flow and the issuance of an ID token. It exposes the stable <code>sub</code> identifier only.</p>
            <p><strong>profile</strong>: user profile data only. Currently includes <code>name</code>, <code>preferred_username</code>, and <code>picture</code>.</p>
            <p><strong>email</strong>: email address data. In Pan Bagnat today, no authoritative email source is stored, so this scope does not emit email claims yet. It is intentionally kept separate from <strong>profile</strong> in the OIDC spec.</p>
            <p><strong>roles</strong>: module-scoped access data. Includes the linked <code>module</code> object plus <code>roles</code> and <code>role_slugs</code> for that module only.</p>
          </div>
        )}
        <div className="oidc-scopes">
          {SCOPE_LABELS.map((scope) => (
            <label key={scope} className={`oidc-scope ${scope === 'openid' ? 'locked' : ''}`}>
              <input
                type="checkbox"
                checked={(data.allowed_scopes || []).includes(scope) || scope === 'openid'}
                disabled={scope === 'openid'}
                onChange={() => toggleScope(scope)}
              />
              <span>{scope}</span>
            </label>
          ))}
        </div>
        <div className="oidc-userinfo-example">
          <div className="oidc-section-header">
            <h4>UserInfo example</h4>
            <Button
              label="Copy JSON"
              color="gray"
              onClick={() => copyText(userInfoExample, 'UserInfo example')}
            />
          </div>
          <p>This is a dynamic example based on the scopes currently enabled for this client.</p>
          <pre>{userInfoExample}</pre>
          {data.allowed_scopes?.includes('email') && (
            <p className="oidc-warning">
              The <code>email</code> scope is enabled in the UI, but Pan Bagnat currently has no authoritative email source, so the live endpoint will not emit email claims yet.
            </p>
          )}
        </div>
      </div>

      {visibleSecret && (
        <div className="oidc-secret-box">
          <div>
            <h4>Client secret visible</h4>
            <p>Copy this client secret now. It will not be shown again after closing.</p>
          </div>
          <div className="oidc-actions">
            <Button label="Close" color="gray" onClick={() => setVisibleSecret('')} />
          </div>
        </div>
      )}
    </div>
  );
}
