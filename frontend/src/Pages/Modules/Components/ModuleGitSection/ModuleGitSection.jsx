import React, { useEffect, useMemo, useState } from 'react'
import './ModuleGitSection.css'
import Button from 'Global/Button/Button'
import { fetchWithAuth } from 'Global/utils/Auth'
import { toast } from 'react-toastify'
import { socketService } from 'Global/SocketService/SocketService'
import { useSearchParams } from 'react-router-dom'

export default function ModuleGitSection({ moduleId }) {
  const [searchParams, setSearchParams] = useSearchParams()
  const [status, setStatus] = useState(null)
  const [commits, setCommits] = useState([])
  const [branches, setBranches] = useState([])
  const [branchesLoading, setBranchesLoading] = useState(true)
  const [commitsLoading, setCommitsLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [currentCommit, setCurrentCommit] = useState('')
  const [selectedBranch, setSelectedBranch] = useState('')

  const loadBranches = async () => {
    setBranchesLoading(true)
    try {
      const br = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/branches`).then(r => r.json()).catch(() => [])
      const list = Array.isArray(br) ? br : []
      setBranches(list)
      const cur = list.find(b => b.current)
      if (cur) setSelectedBranch(cur.name)
      return list
    } finally { setBranchesLoading(false) }
  }

  const loadCommits = async (ref) => {
    setCommitsLoading(true)
    try {
      const useRef = ref || selectedBranch || ''
      const cs = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/commits?limit=20${useRef ? `&ref=${encodeURIComponent(useRef)}` : ''}`).then(r => r.json()).catch(() => [])
      setCommits(Array.isArray(cs) ? cs : [])
    } finally { setCommitsLoading(false) }
  }

  const loadAll = async () => {
    setBranchesLoading(true)
    setCommitsLoading(true)
    try {
      const st = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/status`).then(r => r.json())
      setStatus(st)
      const list = await loadBranches()
      const current = list.find(b => b.current)
      const ref = current ? current.name : ''
      if (ref) setSelectedBranch(ref)
      await loadCommits(ref)
      if (st?.head) setCurrentCommit(st.head)
      try { const ev = new CustomEvent('ide:conflicts', { detail: { paths: (st?.conflicts || []) } }); window.dispatchEvent(ev) } catch {}
    } catch (e) { toast.error('Failed to load Git data') }
  }

  useEffect(() => { loadAll() }, [moduleId])
  // Live updates via WebSocket
  useEffect(() => {
    if (!moduleId) return
    socketService.subscribeTopic(`module:${moduleId}`)
    const unsub = socketService.subscribe((msg) => {
      if (msg?.eventType === 'git_status' && msg?.module_id === moduleId) {
        const p = msg.payload || {}
        setStatus({
          is_merging: !!p.is_merging,
          conflicts: Array.isArray(p.conflicts) ? p.conflicts : [],
          modified: Array.isArray(p.modified) ? p.modified : [],
          last_pull: p.last_pull || '',
          last_fetch: p.last_fetch || '',
          branch: p.branch || ''
        })
        if (p.head) setCurrentCommit(p.head)
        // Refresh lists with minimal queries and ensure commits use resolved current branch
        loadBranches().then((list) => {
          const current = list.find(b => b.current)
          const ref = (current ? current.name : '') || ((p.branch && typeof p.branch === 'string' && p.branch !== 'HEAD' && p.branch) || '')
          if (ref) setSelectedBranch(ref)
          loadCommits(ref)
        })
        // If a merge is in progress, jump to IDE tab
        if (p.is_merging) {
          setSearchParams({ tab: 'ide' })
        }
      }
    })
    return () => { socketService.unsubscribeTopic(`module:${moduleId}`); unsub() }
  }, [moduleId])

  const currentBranch = useMemo(() => branches.find(b => b.current), [branches])

  const fetchRemote = async () => { setBusy(true); try { await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/fetch`, { method: 'POST' }); } finally { setBusy(false); } }
  const pullMerge = async () => { setBusy(true); try { const r = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/pull`, { method: 'POST' }); if (!r?.ok) toast.warn('Pull failed — check logs or conflicts') } finally { setBusy(false); } }
  const finishMerge = async () => {
    try {
      await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/add`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ paths: [] }) })
      const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/merge/continue`, { method:'POST' })
      if (res && res.ok) toast.success('Merge completed'); else toast.error('Merge not completed — conflicts remain')
    } catch { toast.error('Merge failed') } finally { loadAll() }
  }
  const abortMerge = async () => { try { await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/merge/abort`, { method:'POST' }); toast.info('Merge aborted') } catch {} finally { loadAll() } }

  const checkoutRef = async (ref) => {
    setBusy(true)
    try {
      const r = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/checkout`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ ref }) })
      if (!r?.ok) {
        toast.error('Checkout failed')
      } else {
        // Try to use response payload to update UI immediately
        let payload = null
        try { payload = await r.json() } catch {}
        if (payload && typeof payload === 'object') {
          if (payload.status) {
            setStatus(prev => ({ ...(prev||{}), ...payload.status }))
            const b = payload.status.branch
            if (b && b !== 'HEAD') setSelectedBranch(String(b).replace(/^origin\//, ''))
            if (payload.status.head) setCurrentCommit(payload.status.head)
          }
          if (Array.isArray(payload.branches)) setBranches(payload.branches)
          if (Array.isArray(payload.commits)) setCommits(payload.commits)
        } else {
          // Fallback: Only update selectedBranch when ref looks like a branch name
          const isHash = /^[0-9a-f]{7,40}$/i.test(ref)
          if (!isHash) setSelectedBranch(ref.replace(/^origin\//, ''))
        }
      }
    } finally { setBusy(false) }
  }
  // Branch creation/deletion removed: Branches = switch only
  const stageFile = async (p) => { try { await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/add`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ paths: [p] }) }) } catch {} finally { loadAll() } }

  // page renders while branches/commits load separately

  return (
    <div className="git-root">
      <div className="git-toolbar">
        <Button label={busy ? 'Fetching…' : 'Fetch'} color="gray" onClick={fetchRemote} />
        <Button label={busy ? 'Pulling…' : 'Pull (merge)'} color="gray" onClick={pullMerge} />
        {status?.is_merging && (
          <>
            <span className="git-badge">Merge in progress — {status?.conflicts?.length || 0} conflict(s)</span>
            <Button label="Finish merge" color="blue" onClick={finishMerge} />
            <Button label="Abort" color="red" onClick={abortMerge} />
          </>
        )}
        <div className="flex-spacer" />
        <span className="git-meta">Last pull: {status?.last_pull || 'never'}</span>
        <span className="git-meta">Last fetch: {status?.last_fetch || 'never'}</span>
      </div>

      {status?.is_merging && (
        <div className="git-section">
          <div className="git-section-title">Conflicts</div>
          {status.conflicts.length === 0 ? (
            <div className="git-empty">No conflicts detected</div>
          ) : (
            <ul className="git-list">
              {status.conflicts.map((p) => (
                <li key={p} className="git-row">
                  <span className="git-file">{p}</span>
                  <div className="git-row-actions">
                    <Button label="Open" color="gray" onClick={() => { try { const ev = new CustomEvent('ide:open', { detail: { path: p } }); window.dispatchEvent(ev) } catch {} }} />
                    <Button label="Mark resolved" color="gray" onClick={() => stageFile(p)} />
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      <div className="git-columns">
        <div className="git-section">
          <div className="git-section-title">Branches {branchesLoading && <span className="git-meta">(Loading)</span>}</div>
          <ul className="git-list">
            {branches.map((b) => (
              <li key={b.name} className={`git-row ${b.current ? 'current' : ''}`}>
                <span className="git-branch">{b.current ? '• ' : ''}{b.name}</span>
                <div className="git-row-actions">
                  {!b.current && <Button label="Checkout" color="gray" onClick={() => checkoutRef(b.name)} />}
                </div>
              </li>
            ))}
          </ul>
        </div>
        <div className="git-section">
          <div className="git-section-title">Branch Commits {commitsLoading && <span className="git-meta">(Loading)</span>}</div>
          <ul className="git-list">
            {commits.map((c, idx) => (
              <li key={c.hash} className={`git-row ${currentCommit === c.hash ? 'current-commit' : ''}`}>
                <div className="git-commit">
                  <div className="git-commit-subject">{c.subject}</div>
                  <div className="git-commit-meta">{c.hash.slice(0,7)} · {c.author} · {c.date}</div>
                </div>
                <div className="git-row-actions">
                  <Button label={currentCommit === c.hash ? 'Current' : 'Set current'} color={currentCommit === c.hash ? 'green' : 'gray'} onClick={async () => { const ref = (idx === 0 && currentBranch?.name) ? currentBranch.name : c.hash; await checkoutRef(ref); setCurrentCommit(c.hash) }} />
                </div>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  )
}
