import Link from 'Global/Link/Link';
import './ModuleAboutSection.css';

import { useEffect, useState } from 'react'
import { fetchWithAuth } from 'Global/utils/Auth'

const ModuleAboutSection = ({ module }) => {
  const isCloned = new Date(module.last_update).getFullYear() > 2000;
  const [current, setCurrent] = useState({ hash: module.current_commit_hash || '', subject: module.current_commit_subject || '' })
  const [latest, setLatest] = useState({ hash: module.latest_commit_hash || '', subject: module.latest_commit_subject || '' })
  const [behind, setBehind] = useState(typeof module.late_commits === 'number' ? module.late_commits : null)
  const [statusLoading, setStatusLoading] = useState(true)

  useEffect(() => {
    let dead = false
    const load = async () => {
      setStatusLoading(true)
      try {
        const st = await fetchWithAuth(`/api/v1/admin/modules/${module.id}/git/status`).then(r => r.json()).catch(() => null)
        if (!dead && st) {
          setCurrent({ hash: st.head || '', subject: st.head_subject || '' })
          setLatest({ hash: st.latest_hash || '', subject: st.latest_subject || '' })
          if (typeof st.behind === 'number') setBehind(st.behind)
        }
      } catch {}
      if (!dead) setStatusLoading(false)
    }
    if (module?.id) load()
    return () => { dead = true }
  }, [module?.id])

  return (
    <div className="module-version-section">
      <div>
        <div>
          <strong>🪪 Slug:</strong>{' '}
          {module.slug}
        </div>
        <div>
          <strong>🏷️ Current commit:</strong>{' '}
          {statusLoading && !current.hash ? (
            <em>Loading…</em>
          ) : current.hash ? (
            <>
              <code>{current.hash.slice(0,7)}</code>{' '}
              <span title={current.subject}>{current.subject}</span>
            </>
          ) : <em>Unknown</em>}
        </div>
        <div>
          <strong>🌟 Latest commit:</strong>{' '}
          {statusLoading && !latest.hash ? (
            <em>Loading…</em>
          ) : latest.hash ? (
            <>
              <code>{latest.hash.slice(0,7)}</code>{' '}
              <span title={latest.subject}>{latest.subject}</span>
            </>
          ) : <em>Unknown</em>}
        </div>
        <div>
          <strong>🧱 Late Commits:</strong>{' '}
          {statusLoading && behind === null ? <em>Loading…</em> : (behind !== null ? behind : (isCloned ? 0 : <em>Waiting for clone</em>))}
        </div>
        <div>
          <strong>🕒 Last Update:</strong>{' '}
          {isCloned
            ? new Date(module.last_update).toLocaleString()
            : <em>Waiting for clone</em>}
        </div>
        <div>
          <strong>🔗 Git Repo:</strong> <Link url={module.git_url} />
        </div>
        <div>
          <strong>🌿 Git Branch:</strong>{' '}
          {module.git_branch}
        </div>
        <div>
          <strong>🔑 SSH Key:</strong>{' '}
          <Link url={module.ssh_public_key} shorten={42} />
        </div>
        <div className="module-last-deploy" style={{ margin: '8px 0', color: 'var(--text-muted)' }}>
          <strong>Last deploy:</strong>{' '}
          {module.last_deploy && new Date(module.last_deploy).getFullYear() > 2000
            ? new Date(module.last_deploy).toLocaleString()
            : 'Never'}
          {module.last_deploy_status ? ` (${module.last_deploy_status})` : ''}
        </div>
      </div>
    </div>
  );
};

export default ModuleAboutSection;
