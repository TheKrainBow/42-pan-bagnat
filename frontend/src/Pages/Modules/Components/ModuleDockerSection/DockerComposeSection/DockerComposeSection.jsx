import React, { useEffect, useRef, useState } from 'react'
import './DockerComposeSection.css'
import CodeMirror from '@uiw/react-codemirror'
import { linter } from '@codemirror/lint'
import { ViewPlugin, Decoration } from '@codemirror/view'
import { yaml } from '@codemirror/lang-yaml'
import { StreamLanguage } from '@codemirror/language'
import '@vscode/codicons/dist/codicon.css'
import { getIconForFile, DEFAULT_FILE } from 'vscode-icons-ts'
import { toast } from 'react-toastify'
import Button from 'Global/Button/Button'
import { fetchWithAuth } from 'Global/utils/Auth'
import { socketService } from 'Global/SocketService/SocketService'

export default function DockerComposeSection({ moduleId }) {
  const [currentPath, setCurrentPath] = useState('docker-compose.yml')
  const [content, setContent] = useState('')
  const [dirty, setDirty] = useState(false)
  const [unsaved, setUnsaved] = useState({}) // path -> content not yet saved to server
  const [saveMode, setSaveMode] = useState(() => localStorage.getItem('ide:saveMode') || 'onFocusChange') // 'manual' | 'onFocusChange'
  const [extensions, setExtensions] = useState([])
  const [isBinary, setIsBinary] = useState(false)
  const [isDeploying, setIsDeploying] = useState(false)
  const [remoteDeploying, setRemoteDeploying] = useState(false)
  const [repoRoot, setRepoRoot] = useState('')
  const [composeLintExt, setComposeLintExt] = useState(null)
  const [moduleSlug, setModuleSlug] = useState('')
  const editorRef = useRef(null)
  const [gitStatus, setGitStatus] = useState({ is_merging: false, conflicts: [], modified: [], last_fetch: '', last_pull: '' })
  const [gitBusy, setGitBusy] = useState(false)
  const [conflicting, setConflicting] = useState(false)
  const [isModified, setIsModified] = useState(false)

  // React to open-file events from tree
  useEffect(() => {
    const onOpen = (e) => setCurrentPath(e.detail.path)
    window.addEventListener('ide:open', onOpen)
    return () => window.removeEventListener('ide:open', onOpen)
  }, [])

  // React to rename/move events from tree to keep editor path in sync
  useEffect(() => {
    const onRenamed = (e) => {
      const { from, to, isDir } = e.detail || {}
      if (!from || !to) return
      setCurrentPath((prev) => {
        if (!prev) return prev
        if (!isDir) {
          // File rename: update if currently open file matches
          return prev === from ? to : prev
        }
        // Folder rename: update if current file lies under the renamed folder
        if (prev === from) return to
        if (prev.startsWith(from + '/')) {
          return to + prev.slice(from.length)
        }
        return prev
      })
      // Update unsaved cache keys
      setUnsaved((prev) => {
        const out = { ...prev }
        if (!isDir) {
          if (Object.prototype.hasOwnProperty.call(out, from)) { out[to] = out[from]; delete out[from] }
          return out
        }
        for (const k of Object.keys(out)) {
          if (k === from) { out[to] = out[k]; delete out[k]; continue }
          if (k.startsWith(from + '/')) {
            const nk = to + k.slice(from.length)
            out[nk] = out[k]
            delete out[k]
          }
        }
        return out
      })
    }
    window.addEventListener('ide:renamed', onRenamed)
    return () => window.removeEventListener('ide:renamed', onRenamed)
  }, [])

  // React to delete events to clear unsaved cache
  useEffect(() => {
    const onDeleted = (e) => {
      const { path, isDir } = e.detail || {}
      if (!path) return
      setUnsaved((prev) => {
        const out = { ...prev }
        if (!isDir) { delete out[path]; return out }
        for (const k of Object.keys(out)) {
          if (k === path || k.startsWith(path + '/')) delete out[k]
        }
        return out
      })
    }
    window.addEventListener('ide:deleted', onDeleted)
    return () => window.removeEventListener('ide:deleted', onDeleted)
  }, [])

  // (legacy cwd/tree effect removed)

  // Load current file
  useEffect(() => {
    if (!currentPath || currentPath.endsWith('/')) return
    // Prefer local unsaved cache if exists
    const cached = unsaved[currentPath]
    if (typeof cached === 'string') {
      setContent(cached)
      setDirty(true)
      setIsBinary(false)
      return
    }
    fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/read?path=${encodeURIComponent(currentPath)}`)
      .then(r => r?.json?.())
      .then(d => {
        const txt = (d && d.content) || ''
        setIsBinary(isProbablyBinary(txt))
        setContent(txt)
        setDirty(false)
      })
      .catch(() => { setContent(''); setDirty(false) })
  }, [moduleId, currentPath])

  // Realtime deployment status via WebSocket (also git status)
  useEffect(() => {
    if (!moduleId) return
    socketService.subscribeTopic(`module:${moduleId}`)
    const unsubscribe = socketService.subscribe(msg => {
      if (msg?.eventType === 'module_deploy_status' && msg?.module_id === moduleId) {
        const p = msg.payload || {}
        setRemoteDeploying(!!p.is_deploying)
      }
      if (msg?.eventType === 'git_status' && msg?.module_id === moduleId) {
        const p = msg.payload || {}
        const st = {
          is_merging: !!p.is_merging,
          conflicts: Array.isArray(p.conflicts) ? p.conflicts : [],
          modified: Array.isArray(p.modified) ? p.modified : [],
          last_fetch: p.last_fetch || '',
          last_pull: p.last_pull || '',
        }
        setGitStatus(st)
        setConflicting(st.conflicts.includes(currentPath))
        setIsModified(st.modified.includes(currentPath))
        try { const ev = new CustomEvent('ide:conflicts', { detail: { paths: st.conflicts } }); window.dispatchEvent(ev) } catch {}
        try { const ev2 = new CustomEvent('ide:modified', { detail: { paths: st.modified } }); window.dispatchEvent(ev2) } catch {}
      }
    })
    return () => { socketService.unsubscribeTopic(`module:${moduleId}`); unsubscribe() }
  }, [moduleId])

  // Git status polling and broadcaster
  const refreshGitStatus = async () => {
    try {
      const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/status`)
      const js = await res.json().catch(() => ({}))
      const st = js || { is_merging: false, conflicts: [], modified: [], last_fetch: '', last_pull: '' }
      setGitStatus(st)
      setConflicting(!!(st?.conflicts || []).includes(currentPath))
      setIsModified(!!(st?.modified || []).includes(currentPath))
      try { const ev = new CustomEvent('ide:conflicts', { detail: { paths: (st?.conflicts || []) } }); window.dispatchEvent(ev) } catch {}
      try { const ev2 = new CustomEvent('ide:modified', { detail: { paths: (st?.modified || []) } }); window.dispatchEvent(ev2) } catch {}
    } catch {
      // ignore
    }
  }

  useEffect(() => { if (moduleId) refreshGitStatus() }, [moduleId])
  useEffect(() => {
    setConflicting(!!(gitStatus?.conflicts || []).includes(currentPath))
    setIsModified(!!(gitStatus?.modified || []).includes(currentPath))
  }, [currentPath, gitStatus])
  // Remove polls; status comes via WebSocket

  // Ctrl+S to save
  useEffect(() => {
    const onKey = (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
        e.preventDefault(); saveFile();
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [content, currentPath, dirty])

  // Warn before unload if there are unsaved files
  useEffect(() => {
    const onBeforeUnload = (e) => {
      if (Object.keys(unsaved).length > 0) {
        e.preventDefault();
        e.returnValue = ''
      }
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [unsaved])

  // Broadcast unsaved paths to TreeView for visual indicators
  useEffect(() => {
    try { const ev = new CustomEvent('ide:unsaved', { detail: { paths: Object.keys(unsaved) } }); window.dispatchEvent(ev) } catch {}
  }, [unsaved])

  // Auto-save on blur/visibility change when mode is onFocusChange
  useEffect(() => {
    if (saveMode !== 'onFocusChange') return
    const saveAll = async () => {
      const entries = Object.entries(unsaved)
      for (const [path, data] of entries) {
        await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/write`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path, content: data })
        })
      }
      setUnsaved({})
      setDirty(false)
      await refreshGitStatus()
    }
    const onBlur = () => { if (Object.keys(unsaved).length > 0) saveAll() }
    const onVis = () => { if (document.hidden && Object.keys(unsaved).length > 0) saveAll() }
    window.addEventListener('blur', onBlur)
    document.addEventListener('visibilitychange', onVis)
    return () => { window.removeEventListener('blur', onBlur); document.removeEventListener('visibilitychange', onVis) }
  }, [unsaved, saveMode, moduleId])

  // pick language extensions based on file extension
  useEffect(() => {
    (async () => {
      setExtensions(await detectExtensionsDynamic(currentPath))
    })()
  }, [currentPath])

  // Load repo root and attach linter when editing docker-compose.yml
  useEffect(() => {
    const isCompose = (currentPath || '').split('/').pop()?.toLowerCase() === 'docker-compose.yml'
    if (!isCompose) { setComposeLintExt(null); return }
    let cancelled = false
    const load = async () => {
      try {
        const [resRoot, resMod] = await Promise.all([
          fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/root`),
          fetchWithAuth(`/api/v1/admin/modules/${moduleId}`)
        ])
        const j = await resRoot.json().catch(() => null)
        const mod = await resMod.json().catch(() => null)
        const root = j?.abs_path || ''
        if (!cancelled) {
          setRepoRoot(root)
          setModuleSlug(mod?.slug || '')
          setComposeLintExt(makeComposeLinter({ absRoot: root, moduleSlug: mod?.slug || '' }))
        }
      } catch {
        if (!cancelled) {
          setRepoRoot('')
          setModuleSlug('')
          setComposeLintExt(makeComposeLinter({ absRoot: '', moduleSlug: '' }))
        }
      }
    }
    load()
    return () => { cancelled = true }
  }, [moduleId, currentPath])

  const saveFile = async () => {
    if (!dirty) return
    await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/write`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: currentPath, content })
    })
    setDirty(false)
    setUnsaved(prev => { const { [currentPath]:_, ...rest } = prev; return rest })
    await refreshGitStatus()
  }

  // (removed legacy openEntry/renderTree)
  const reloadCurrentFile = async () => {
    if (!currentPath || currentPath.endsWith('/')) return
    const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/read?path=${encodeURIComponent(currentPath)}`)
    const d = await res.json().catch(() => ({}))
    const txt = d?.content || ''
    setIsBinary(isProbablyBinary(txt))
    setContent(txt)
    setDirty(false)
  }

  const quickResolveBoth = () => {
    const resolved = resolveConflictMarkers(content, 'both')
    setContent(resolved)
    setDirty(true)
    setUnsaved(prev => ({ ...prev, [currentPath]: resolved }))
  }

  return (
    <div className="ide-root">
      <div className="ide-tree">
        <TreeView moduleId={moduleId} selectedPath={currentPath} />
      </div>
      <div className="ide-editor">
        <div className="pane-header">
          <span>
            {currentPath}
            {gitStatus?.conflicts?.includes(currentPath) && (
              <span className="conflict-dot" title="Merge conflict in this file" />
            )}
          </span>
          <div className="header-actions">
            <div className="git-actions">
              {isModified && (
                <Button
                  label="Checkout"
                  color="gray"
                  onClick={async () => {
                    await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/file/checkout`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ path: currentPath, ref: '' }) })
                    await reloadCurrentFile();
                    await refreshGitStatus();
                  }}
                />
              )}
              {conflicting && (
                <>
                  <Button label="Accept current" color="gray" onClick={async ()=>{ await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/file/resolve/ours`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ path: currentPath }) }); await reloadCurrentFile(); await refreshGitStatus(); }} />
                  <Button label="Accept incoming" color="gray" onClick={async ()=>{ await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/file/resolve/theirs`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ path: currentPath }) }); await reloadCurrentFile(); await refreshGitStatus(); }} />
                  <Button label="Accept both" color="gray" onClick={()=>{ quickResolveBoth() }} />
                  <Button label="Mark resolved" color="blue" onClick={async ()=>{ await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/add`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ paths: [currentPath] }) }); await refreshGitStatus(); }} />
                  <Button label="Checkout file" color="red" onClick={async ()=>{ await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/file/checkout`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ path: currentPath, ref: '' }) }); await reloadCurrentFile(); await refreshGitStatus(); }} />
                </>
              )}
              {gitStatus?.is_merging && (
                <Button
                  label="Finish merge"
                  color="blue"
                  onClick={async () => {
                    try {
                      // Stage all changes and try to commit using merge message
                      await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/add`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ paths: [] }) })
                      const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/git/merge/continue`, { method:'POST' })
                      if (res && res.ok) { toast.success('Merge completed'); refreshGitStatus() }
                      else { toast.error('Merge not completed. Resolve remaining conflicts.') }
                    } catch {
                      toast.error('Merge failed')
                    }
                  }}
                />
              )}
            </div>
            {((currentPath || '').split('/').pop()?.toLowerCase() === 'docker-compose.yml') && (
              <Button
                label={isDeploying ? 'Deploying…' : 'Deploy'}
                color="green"
                onClick={async () => {
                  setIsDeploying(true)
                  try {
                    const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/docker/deploy`, {
                      method: 'POST',
                      headers: { 'Content-Type': 'application/json' },
                      body: JSON.stringify({ config: content })
                    })
                    if (!res || !res.ok) {
                      toast.error('Deploy failed')
                    } else {
                      toast.success('Deployment started')
                    }
                  } catch (e) {
                    toast.error('Deploy failed')
                  } finally {
                    setIsDeploying(false)
                  }
                }}
                disabled={isBinary || isDeploying || remoteDeploying || gitStatus?.is_merging}
              />
            )}
            <Button label={dirty ? 'Save (Ctrl+S)' : 'Saved'} color="blue" onClick={saveFile} disabled={!dirty || isBinary} />
            {isBinary && (
              <span style={{ marginLeft: 8, fontSize: 12, color: 'var(--text-muted)' }}>
                Read-only (binary)
              </span>
            )}
            <select
              value={saveMode}
              onChange={(e) => { setSaveMode(e.target.value); localStorage.setItem('ide:saveMode', e.target.value) }}
              title="Save mode"
              style={{ marginLeft: 8, background: 'var(--light-background)', color: 'var(--text-primary)', border: '1px solid var(--border)', borderRadius: 4, padding: '4px 6px', fontSize: 12 }}
            >
              <option value="manual">No auto save</option>
              <option value="onFocusChange">On focus change</option>
            </select>
          </div>
        </div>
        <div className="codemirror-wrapper">
          <CodeMirror
            value={content}
            extensions={[...extensions, ...(composeLintExt ? [composeLintExt] : []), ...(isEnvFile(currentPath) ? [dotenvColorizerExt()] : [])]}
            height="calc(100vh - 357px)"
            theme="dark"
            editable={!isBinary}
            onChange={(v) => { setContent(v); setDirty(true); setUnsaved(prev => ({ ...prev, [currentPath]: v })) }}
            basicSetup={{ lineNumbers: true }}
            onBlur={() => { if (saveMode === 'onFocusChange') saveFile() }}
          />
        </div>
      </div>
    </div>
  )
}

function TreeView({ moduleId, selectedPath, title }) {
  const [nodes, setNodes] = useState([{ name: '.', path: '.', is_dir: true }])
  const [open, setOpen] = useState({ '.': true })
  const [selected, setSelected] = useState('')

  const [children, setChildren] = useState({})
  const [menu, setMenu] = useState(null) // {x,y,type:'file'|'dir'|'empty', path, base}
  const [editing, setEditing] = useState('') // path being renamed or special key 'new@<base>'
  const [editName, setEditName] = useState('')
  const [editError, setEditError] = useState('') // 'duplicate' | ''
  const inputRef = useRef(null)
  const treeRootRef = useRef(null)
  const [isActive, setIsActive] = useState(false)
  const [dragPath, setDragPath] = useState('')
  const [dragIsDir, setDragIsDir] = useState(false)
  const [dropPath, setDropPath] = useState('') // path of folder/file hovered
  const [dropFolderPath, setDropFolderPath] = useState('') // active folder drop target for left border
  const openTimerRef = useRef(null)
  const hoverPathRef = useRef('')
  const [unsavedSet, setUnsavedSet] = useState(new Set())
  const [undoStack, setUndoStack] = useState([]) // array of actions
  const [redoStack, setRedoStack] = useState([])
  const [conflictSet, setConflictSet] = useState(new Set())
  const [modifiedSet, setModifiedSet] = useState(new Set())

  useEffect(() => { load('.') }, [])

  // Listen to unsaved map updates from parent to render dots
  useEffect(() => {
    const onUnsaved = (e) => {
      const paths = e.detail?.paths || []
      setUnsavedSet(new Set(paths))
    }
    window.addEventListener('ide:unsaved', onUnsaved)
    return () => window.removeEventListener('ide:unsaved', onUnsaved)
  }, [])

  // Listen to conflict paths from Git status
  useEffect(() => {
    const onConf = (e) => { const paths = e.detail?.paths || []; setConflictSet(new Set(paths)) }
    window.addEventListener('ide:conflicts', onConf)
    return () => window.removeEventListener('ide:conflicts', onConf)
  }, [])

  // Listen to modified paths from Git status
  useEffect(() => {
    const onMod = (e) => { const paths = e.detail?.paths || []; setModifiedSet(new Set(paths)) }
    window.addEventListener('ide:modified', onMod)
    return () => window.removeEventListener('ide:modified', onMod)
  }, [])

  const load = async (p) => {
    const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/tree?path=${encodeURIComponent(p)}`)
    if (!res) return
    const data = await res.json().catch(() => [])
    setChildren(prev => ({ ...prev, [p]: Array.isArray(data) ? data : [] }))
  }

  const toggle = async (p) => {
    setOpen(o => ({ ...o, [p]: !o[p] }))
    if (!children[p]) await load(p)
  }

  const select = async (n) => {
    setSelected(n.path)
    if (n.is_dir) await toggle(n.path)
    else {
      const event = new CustomEvent('ide:open', { detail: { path: n.path } })
      window.dispatchEvent(event)
    }
  }

  useEffect(() => {
    const onOpen = (e) => setSelected(e.detail.path)
    window.addEventListener('ide:open', onOpen)
    return () => window.removeEventListener('ide:open', onOpen)
  }, [])

  // Ensure selection follows the editor's current path on initial load and updates
  useEffect(() => {
    if (selectedPath) setSelected(selectedPath)
  }, [selectedPath])

  // Reset editName when entering edit mode
  useEffect(() => {
    if (!editing) { setEditName(''); setEditError(''); return }
    if (editing.startsWith('new@') || editing.startsWith('newdir@')) {
      setEditName('')
    } else {
      setEditName(baseName(editing))
    }
    setEditError('')
  }, [editing])

  // Debounced validation (200ms) for duplicate names in same folder
  useEffect(() => {
    if (!editing) return
    const base = editing.startsWith('new@') ? editing.slice('new@'.length)
      : editing.startsWith('newdir@') ? editing.slice('newdir@'.length)
      : parentOf(editing)
    const timer = setTimeout(() => {
      const list = children[base] || []
      const exists = !!list.find(it => it.name === editName && it.path !== editing)
      setEditError(exists && editName ? 'duplicate' : '')
    }, 100)
    return () => clearTimeout(timer)
  }, [editName, editing, children])

  // F2 to rename selected, Delete to delete selected (when tree is focused)
  useEffect(() => {
    const onKey = (e) => {
      if (!isActive) return
      // Undo / Redo
      const isZ = e.key.toLowerCase() === 'z'
      if ((e.ctrlKey || e.metaKey) && isZ && !e.shiftKey) {
        e.preventDefault();
        handleUndo();
        return
      }
      if ((e.ctrlKey || e.metaKey) && isZ && e.shiftKey) {
        e.preventDefault();
        handleRedo();
        return
      }
      if (e.key === 'F2' && selected) {
        e.preventDefault()
        setEditing(selected)
        setMenu(null)
        return
      }
      if (e.key === 'Delete' && selected) {
        e.preventDefault()
        if (confirm(`Delete ${selected}?`)) {
          // try to determine if it's a file or folder from loaded children
          const base = parentOf(selected)
          const list = children[base] || []
          const node = list.find(it => it.path === selected)
          if (node) handleDelete(node)
          else {
            // fallback: try read to determine
            apiRead(selected).then(content => {
              if (content !== null) {
                handleDelete({ path: selected, is_dir: false })
              } else {
                apiDelete(selected).then(() => load(base))
              }
            })
          }
        }
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [selected, isActive])

  const refreshParents = async (...paths) => {
    const uniq = Array.from(new Set(paths.map(parentOf)))
    await Promise.all(uniq.map((p) => load(p)))
  }

  const handleUndo = async () => {
    if (undoStack.length === 0) return
    const action = undoStack[undoStack.length - 1]
    setUndoStack(undoStack.slice(0, -1))
    try {
      switch (action.type) {
        case 'create-file':
          await apiDelete(action.path)
          await refreshParents(action.path)
          break
        case 'mkdir':
          await apiDelete(action.path)
          await refreshParents(action.path)
          break
        case 'move': {
          await apiRename(action.to, action.from)
          await refreshParents(action.from, action.to)
          setSelected(action.from)
          break
        }
        case 'delete-file': {
          await apiWrite(action.path, action.content || '')
          await refreshParents(action.path)
          setSelected(action.path)
          break
        }
        default:
          break
      }
    } finally {
      setRedoStack(prev => [...prev, action])
    }
  }

  const handleRedo = async () => {
    if (redoStack.length === 0) return
    const action = redoStack[redoStack.length - 1]
    setRedoStack(redoStack.slice(0, -1))
    try {
      switch (action.type) {
        case 'create-file':
          await apiWrite(action.path, action.content || '')
          await refreshParents(action.path)
          setSelected(action.path)
          break
        case 'mkdir':
          await apiMkdir(action.path)
          await refreshParents(action.path)
          break
        case 'move':
          await apiRename(action.from, action.to)
          await refreshParents(action.from, action.to)
          setSelected(action.to)
          break
        case 'delete-file':
          await apiDelete(action.path)
          await refreshParents(action.path)
          break
        default:
          break
      }
    } finally {
      setUndoStack(prev => [...prev, action])
    }
  }

  const handleDelete = async (node) => {
    if (!node) return
    const base = parentOf(node.path)
    if (node.is_dir) {
      await apiDelete(node.path)
      await load(base)
      try { const ev = new CustomEvent('ide:deleted', { detail: { path: node.path, isDir: true } }); window.dispatchEvent(ev) } catch {}
      return
    }
    // file: read and cache
    const content = await apiRead(node.path)
    await apiDelete(node.path)
    setUndoStack(prev => [...prev, { type: 'delete-file', path: node.path, content }])
    setRedoStack([])
    await load(base)
    try { const ev = new CustomEvent('ide:deleted', { detail: { path: node.path, isDir: false } }); window.dispatchEvent(ev) } catch {}
  }

  // Close menu on outside click
  useEffect(() => {
    if (!menu) return
    const onClick = () => setMenu(null)
    window.addEventListener('click', onClick)
    return () => window.removeEventListener('click', onClick)
  }, [menu])

  const apiRename = async (oldPath, newPath) => {
    const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/rename`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ old_path: oldPath, new_path: newPath })
    })
    if (!res || !res.ok) throw new Error('rename-failed')
  }
  const apiDelete = async (path) => {
    const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/delete`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path })
    })
    if (!res || !res.ok) return
  }
  const apiMkdir = async (path) => {
    const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/mkdir`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path })
    })
    if (!res || !res.ok) return
  }
  const apiWrite = async (path, data) => {
    const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/write`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path, content: data })
    })
    if (!res || !res.ok) return
  }
  const apiRead = async (path) => {
    const r = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/read?path=${encodeURIComponent(path)}`)
    if (!r || !r.ok) return null
    const j = await r.json().catch(() => null)
    return j?.content ?? null
  }

  const parentOf = (p) => {
    const idx = p.lastIndexOf('/')
    return idx === -1 ? '.' : p.slice(0, idx)
  }
  const join = (a,b) => (a === '.' ? b : `${a}/${b}`)
  const baseName = (p) => (p || '').split('/').pop() || ''
  const isDirPath = (p) => {
    if (!p) return false
    const list = (children[parentOf(p)] || [])
    return !!(list.find(it => it.path === p && it.is_dir))
  }
  const hasUnsaved = (p, isDir) => {
    if (!unsavedSet || unsavedSet.size === 0) return false
    if (!isDir) return unsavedSet.has(p)
    for (const up of unsavedSet) {
      if (up === p || up.startsWith(p + '/')) return true
    }
    return false
  }
  const hasConflict = (p, isDir) => {
    if (!conflictSet || conflictSet.size === 0) return false
    if (!isDir) return conflictSet.has(p)
    for (const cp of conflictSet) { if (cp === p || cp.startsWith(p + '/')) return true }
    return false
  }
  const hasModified = (p, isDir) => {
    if (!modifiedSet || modifiedSet.size === 0) return false
    if (!isDir) return modifiedSet.has(p)
    for (const mp of modifiedSet) { if (mp === p || mp.startsWith(p + '/')) return true }
    return false
  }
  const baseForNew = () => {
    if (selected) return isDirPath(selected) ? selected : parentOf(selected)
    return '.'
  }

  const ensureVisible = async (targetPath) => {
    // Open all ancestors so the inline editor becomes visible
    const chain = []
    let cur = targetPath
    while (cur && cur !== '.' && !chain.includes(cur)) {
      chain.push(cur)
      cur = parentOf(cur)
    }
    setOpen(o => {
      const next = { ...o, '.': true }
      for (let i = chain.length - 1; i >= 0; i--) {
        next[chain[i]] = true
      }
      return next
    })
    // Load directories if not loaded yet so the tree renders all levels
    if (!children['.']) await load('.')
    for (let i = chain.length - 1; i >= 0; i--) {
      const p = chain[i]
      if (!children[p]) await load(p)
    }
  }

  const startNewFile = async (base) => {
    await ensureVisible(base)
    setEditing(`new@${base}`)
    setMenu(null)
  }
  const startNewFolder = async (base) => {
    await ensureVisible(base)
    setEditing(`newdir@${base}`)
    setMenu(null)
  }

  const onCommitEdit = async (nodePath, newName, isDir) => {
    if (!newName) { setEditing(''); return }
    if (/[\\/]/.test(newName)) { setEditError('invalid'); inputRef.current?.focus(); return }
    // Front validation for duplicates
    const base = nodePath.startsWith('new@') ? nodePath.slice('new@'.length)
      : nodePath.startsWith('newdir@') ? nodePath.slice('newdir@'.length)
      : parentOf(nodePath)
    const list = children[base] || []
    const exists = !!list.find(it => it.name === newName && it.path !== nodePath)
    if (exists) { setEditError('duplicate'); inputRef.current?.focus(); return }
    try {
      if (nodePath.startsWith('new@')) {
        const base = nodePath.slice('new@'.length)
        const newPath = join(base, newName)
        await apiWrite(newPath, '')
        setUndoStack(prev => [...prev, { type: 'create-file', path: newPath }])
        setRedoStack([])
        await load(base)
        setSelected(newPath)
        const evt = new CustomEvent('ide:open', { detail: { path: newPath }})
        window.dispatchEvent(evt)
      } else if (nodePath.startsWith('newdir@')) {
        const base = nodePath.slice('newdir@'.length)
        const newPath = join(base, newName)
        await apiMkdir(newPath)
        setUndoStack(prev => [...prev, { type: 'mkdir', path: newPath }])
        setRedoStack([])
        await load(base)
        setOpen(o => ({ ...o, [newPath]: true }))
      } else {
        const base = parentOf(nodePath)
        const newPath = join(base, newName)
        if (newPath !== nodePath) {
          try {
            await apiRename(nodePath, newPath)
          } catch (e) {
            // fetchWithAuth already toasts 409; add a fallback toast
            if (e?.message !== 'rename-failed') toast.error('Rename failed')
            return
          }
          setUndoStack(prev => [...prev, { type: 'move', from: nodePath, to: newPath }])
          setRedoStack([])
          await Promise.all([load(base)])
          setSelected(newPath)
          // notify editor about rename so it updates its current path
          const ev = new CustomEvent('ide:renamed', { detail: { from: nodePath, to: newPath, isDir } })
          window.dispatchEvent(ev)
        }
      }
    } finally {
      setEditing('')
    }
  }

  const onDuplicate = async (path, name) => {
    const idx = name.lastIndexOf('.')
    const base = parentOf(path)
    const ext = (idx > 0 ? name.slice(idx+1) : '')
    const stem = (idx > 0 ? name.slice(0, idx) : name)
    const target = join(base, ext ? `${stem}-copy.${ext}` : `${stem}-copy`)
    const res = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/read?path=${encodeURIComponent(path)}`)
    if (!res || !res.ok) return
    const data = await res.json().catch(() => ({}))
    const wr = await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/fs/write`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: target, content: data.content || '' })
    })
    if (!wr || !wr.ok) return
    await load(base)
  }

  const menuPos = (e) => {
    const cont = e.target.closest('.ide-tree')
    if (!cont) return { x: e.clientX, y: e.clientY }
    const rect = cont.getBoundingClientRect()
    return { x: e.clientX - rect.left + cont.scrollLeft, y: e.clientY - rect.top + cont.scrollTop }
  }
  const openMenuForNode = (e, node) => {
    e.preventDefault(); e.stopPropagation()
    const pos = menuPos(e)
    setMenu({ x: pos.x, y: pos.y, type: node.is_dir ? 'dir' : 'file', path: node.path, name: node.name, base: parentOf(node.path) })
  }
  const openMenuForEmpty = (e, base) => {
    e.preventDefault(); e.stopPropagation()
    const pos = menuPos(e)
    setMenu({ x: pos.x, y: pos.y, type: 'empty', base })
  }

  // Preload VSCode icon asset URLs via Vite
  const ICON_URLS = React.useMemo(() => {
    // Path from this file: frontend/src/Pages/Modules/Components/ModuleDockerSection/DockerComposeSection
    // Up 6 levels to frontend/, then into node_modules
    const modules = import.meta.glob('../../../../../../node_modules/vscode-icons-ts/build/icons/*.svg', { eager: true, as: 'url' })
    const map = {}
    for (const k in modules) {
      const file = k.split('/').pop()
      map[file] = modules[k]
    }
    return map
  }, [])

  const fileIconUrl = (fileName) => {
    try {
      const base = (fileName || '').split('/').pop() || ''
      const lower = base.toLowerCase()
      // Override: any Dockerfile variant -> Docker icon
      if (lower.includes('dockerfile')) {
        const forced = 'file_type_docker.svg'
        return ICON_URLS[forced] || ICON_URLS[DEFAULT_FILE]
      }
      // Override: any docker-compose*.yml/.yaml (and compose*.yml) -> Docker icon
      if ((lower.includes('docker-compose') || lower.startsWith('compose')) && (lower.endsWith('.yml') || lower.endsWith('.yaml'))) {
        const forced = 'file_type_docker.svg'
        return ICON_URLS[forced] || ICON_URLS[DEFAULT_FILE]
      }
      const icon = getIconForFile(base) || DEFAULT_FILE
      return ICON_URLS[icon] || ICON_URLS[DEFAULT_FILE]
    } catch {
      return ICON_URLS[DEFAULT_FILE]
    }
  }

  const handleDragStart = (e, n) => {
    e.stopPropagation()
    setDragPath(n.path)
    setDragIsDir(!!n.is_dir)
    try { e.dataTransfer.setData('text/plain', n.path) } catch {}
    e.dataTransfer.effectAllowed = 'move'
  }
  const handleDragEnd = () => {
    setDragPath(''); setDragIsDir(false); setDropPath(''); setDropFolderPath(''); hoverPathRef.current=''
    if (openTimerRef.current) { clearTimeout(openTimerRef.current); openTimerRef.current = null }
  }
  const handleDragOverRow = (e, n) => {
    e.preventDefault(); e.dataTransfer.dropEffect = 'move'
    setDropPath(n.path)
    if (n.is_dir) {
      setDropFolderPath(n.path)
      if (hoverPathRef.current !== n.path) {
        hoverPathRef.current = n.path
        if (openTimerRef.current) clearTimeout(openTimerRef.current)
        openTimerRef.current = setTimeout(async () => {
          if (hoverPathRef.current === n.path) {
            setOpen(o => ({ ...o, [n.path]: true }))
            if (!children[n.path]) await load(n.path)
          }
        }, 400)
      }
    } else {
      setDropFolderPath(parentOf(n.path))
    }
  }
  const handleDragEnterRow = (e, n) => {
    e.preventDefault();
    setDropPath(n.path)
    if (n.is_dir) {
      setDropFolderPath(n.path)
      hoverPathRef.current = n.path
      if (openTimerRef.current) clearTimeout(openTimerRef.current)
      openTimerRef.current = setTimeout(async () => {
        if (hoverPathRef.current === n.path) {
          setOpen(o => ({ ...o, [n.path]: true }))
          if (!children[n.path]) await load(n.path)
        }
      }, 400)
    } else {
      setDropFolderPath(parentOf(n.path))
    }
  }
  const handleDragLeaveRow = (e, n) => {
    // cancel pending open if leaving the row entirely
    if (!e.currentTarget.contains(e.relatedTarget)) {
      if (openTimerRef.current) { clearTimeout(openTimerRef.current); openTimerRef.current = null }
      if (hoverPathRef.current === n.path) hoverPathRef.current = ''
      if (dropFolderPath === n.path) setDropFolderPath('')
    }
  }
  const performDrop = async (srcPath, targetNodePath, isDir) => {
    if (!srcPath) return
    let destDir = isDir ? targetNodePath : parentOf(targetNodePath)
    // Prevent moving into its own descendant
    if (srcPath === destDir || destDir.startsWith(srcPath + '/')) return
    const newPath = join(destDir, baseName(srcPath))
    if (newPath === srcPath) return
    // Ensure destination folder is opened so the moved item is visible
    setOpen(o => ({ ...o, [destDir]: true }))
    try {
      await apiRename(srcPath, newPath)
    } catch (e) {
      if (e?.message !== 'rename-failed') toast.error('Move failed')
      return
    }
    setUndoStack(prev => [...prev, { type: 'move', from: srcPath, to: newPath }])
    setRedoStack([])
    await Promise.all([load(parentOf(srcPath)), load(destDir)])
    setSelected(newPath)
    // For file moves, focus the moved file in editor; for folder moves, keep current file
    if (!dragIsDir) {
      try { const evt = new CustomEvent('ide:open', { detail: { path: newPath } }); window.dispatchEvent(evt) } catch {}
    } else {
      // Notify rename so editor updates paths only if current file lies under the moved folder
      try { const ev = new CustomEvent('ide:renamed', { detail: { from: srcPath, to: newPath, isDir: true } }); window.dispatchEvent(ev) } catch {}
    }
  }
  const handleDropOnRow = async (e, n) => {
    e.preventDefault();
    const src = dragPath || (() => { try { return e.dataTransfer.getData('text/plain') } catch { return '' } })()
    await performDrop(src, n.path, n.is_dir)
    handleDragEnd()
  }
  const handleDropOnFolderBody = async (e, folderPath) => {
    e.preventDefault();
    const src = dragPath || (() => { try { return e.dataTransfer.getData('text/plain') } catch { return '' } })()
    // drop into folder itself
    await performDrop(src, folderPath, true)
    handleDragEnd()
  }

  const reloadAll = async () => {
    const keys = Object.keys(children)
    if (keys.length === 0) { await load('.'); return }
    await Promise.all(keys.map(k => load(k)))
  }
  const collapseAll = () => {
    setOpen({})
  }

  const render = (p) => {
    const list = children[p] || []
    return (
      <ul
        className="tree-ul"
        onContextMenu={(e) => { if (!e.target.closest('.tree-row')) openMenuForEmpty(e, p) }}
        onDoubleClick={(e) => { if (!e.target.closest('.tree-row')) startNewFile(p) }}
        onDragOver={(e) => { if (!e.target.closest('.tree-row')) { e.preventDefault(); setDropPath(p); setDropFolderPath(p) } }}
        onDrop={(e) => { if (!e.target.closest('.tree-row')) handleDropOnFolderBody(e, p) }}
      >
        {list.map((n) => (
          <li key={n.path} className={`tree-li ${selected===n.path?'sel':''} ${dropPath===n.path ? 'drop-target':''}`}>
            <div className="tree-row" draggable onDragStart={(e) => handleDragStart(e, n)} onDragEnd={handleDragEnd} onDragOver={(e) => handleDragOverRow(e, n)} onDragEnter={(e) => handleDragEnterRow(e, n)} onDragLeave={(e) => handleDragLeaveRow(e, n)} onDrop={(e) => handleDropOnRow(e, n)} onClick={() => select(n)} onContextMenu={(e) => openMenuForNode(e, n)}>
              {n.is_dir ? (
                <span className={`tree-icon codicon ${open[n.path] ? 'codicon-chevron-down' : 'codicon-chevron-right'}`}></span>
              ) : (
                <img className="tree-icon" src={fileIconUrl(n.name)} alt="file" />
              )}
              {editing === n.path ? (
                <input
                  ref={inputRef}
                  className={`tree-edit-input ${editError ? 'error' : ''}`}
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  onKeyDown={(e) => { if (e.key==='Enter') onCommitEdit(n.path, editName, n.is_dir); if (e.key==='Escape') setEditing('') }}
                  onBlur={(e) => onCommitEdit(n.path, editName, n.is_dir)}
                  autoFocus
                />
              ) : (
                <span className="tree-name">{n.name}
                  {hasUnsaved(n.path, n.is_dir) && (<span className="unsaved-dot" title="Unsaved changes" />)}
                  {hasConflict(n.path, n.is_dir) && (<span className="conflict-dot" title="Merge conflict" />)}
                  {hasModified(n.path, n.is_dir) && (<span className="modified-badge" title="Modified">M</span>)}
                </span>
              )}
            </div>
            {editing === n.path && editError && (
              <div className="tree-edit-error">{editError === 'duplicate' ? (
                <>A file or folder <strong>{editName}</strong> already exists at this location.<br />Please choose a different name.</>
              ) : (
                <>Invalid name. Slashes are not allowed.</>
              )}</div>
            )}
            {n.is_dir && open[n.path] && (
              <div className="tree-children">{render(n.path)}</div>
            )}
          </li>
        ))}
        {editing.startsWith(`new@${p}`) && (
          <li className="tree-li">
            <div className="tree-row">
              <span className="tree-icon codicon codicon-symbol-file" />
              <input
                ref={inputRef}
                className={`tree-edit-input ${editError ? 'error' : ''}`}
                placeholder="new file name"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                onKeyDown={(e) => { if (e.key==='Enter') onCommitEdit(`new@${p}`, editName, false); if (e.key==='Escape') setEditing('') }}
                onBlur={(e) => onCommitEdit(`new@${p}`, editName, false)}
                autoFocus
              />
            </div>
            {editing === `new@${p}` && editError && (
              <div className="tree-edit-error">{editError === 'duplicate' ? (
                <>A file or folder <strong>{editName}</strong> already exists at this location.<br />Please choose a different name.</>
              ) : (
                <>Invalid name. Slashes are not allowed.</>
              )}</div>
            )}
          </li>
        )}
        {editing.startsWith(`newdir@${p}`) && (
          <li className="tree-li">
            <div className="tree-row">
              <span className="tree-icon codicon codicon-folder" />
              <input
                ref={inputRef}
                className={`tree-edit-input ${editError ? 'error' : ''}`}
                placeholder="new folder name"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                onKeyDown={(e) => { if (e.key==='Enter') onCommitEdit(`newdir@${p}`, editName, true); if (e.key==='Escape') setEditing('') }}
                onBlur={(e) => onCommitEdit(`newdir@${p}`, editName, true)}
                autoFocus
              />
            </div>
            {editing === `newdir@${p}` && editError && (
              <div className="tree-edit-error">{editError === 'duplicate' ? (
                <>A file or folder <strong>{editName}</strong> already exists at this location.<br />Please choose a different name.</>
              ) : (
                <>Invalid name. Slashes are not allowed.</>
              )}</div>
            )}
          </li>
        )}
      </ul>
    )
  }

  return (
    <div
      className={`tree-root ${dropFolderPath ? 'drop-folder' : ''}`}
      ref={treeRootRef}
      tabIndex={0}
      onFocus={() => setIsActive(true)}
      onBlur={() => setIsActive(false)}
      onContextMenu={(e) => { if (!e.target.closest('.tree-row')) openMenuForEmpty(e, '.') }}
      onDoubleClick={(e) => { if (!e.target.closest('.tree-row')) startNewFile('.') }}
    >
      <div className="tree-toolbar">
        <div className="title">{title || 'FILES'}</div>
        <div className="actions">
          <button className="icon-btn" title="New File" onClick={() => startNewFile(baseForNew())}>
            <span className="codicon codicon-new-file" />
          </button>
          <button className="icon-btn" title="New Folder" onClick={() => startNewFolder(baseForNew())}>
            <span className="codicon codicon-new-folder" />
          </button>
          <button className="icon-btn" title="Reload" onClick={reloadAll}>
            <span className="codicon codicon-refresh" />
          </button>
          <button className="icon-btn" title="Collapse All" onClick={collapseAll}>
            <span className="codicon codicon-collapse-all" />
          </button>
        </div>
      </div>
      {render('.')}
      {menu && (
        <div className="tree-menu" style={{ left: menu.x, top: menu.y }} onClick={(e) => e.stopPropagation()}>
          {menu.type === 'file' && (
            <>
              <div className="tree-menu-item" onClick={() => { setEditing(menu.path); setMenu(null) }}>Rename</div>
              <div className="tree-menu-item" onClick={() => { onDuplicate(menu.path, menu.name); setMenu(null) }}>Duplicate</div>
              <div className="tree-menu-item" onClick={async () => { if (confirm(`Delete ${menu.path}?`)) { await handleDelete({ path: menu.path, is_dir: false }); await load(menu.base); } setMenu(null) }}>Delete</div>
            </>
          )}
          {menu.type === 'dir' && (
            <>
              <div className="tree-menu-item" onClick={() => { setEditing(menu.path); setMenu(null) }}>Rename</div>
              <div className="tree-menu-item" onClick={() => { startNewFile(menu.path) }}>New File</div>
              <div className="tree-menu-item" onClick={() => { startNewFolder(menu.path) }}>New Folder</div>
              <div className="tree-menu-item" onClick={async () => { if (confirm(`Delete folder ${menu.path}?`)) { await apiDelete(menu.path); await load(menu.base); } setMenu(null) }}>Delete</div>
            </>
          )}
          {menu.type === 'empty' && (
            <>
              <div className="tree-menu-item" onClick={() => startNewFile(menu.base)}>New File</div>
              <div className="tree-menu-item" onClick={() => startNewFolder(menu.base)}>New Folder</div>
            </>
          )}
        </div>
      )}
    </div>
  )
}

async function detectExtensionsDynamic(path) {
  const p = (path || '').toLowerCase()
  const fname = (path || '').split('/').pop()?.toLowerCase() || ''
  try {
    // YAML
    if (p.endsWith('.yml') || p.endsWith('.yaml')) return [yaml()]

    // JSON
    if (p.endsWith('.json')) {
      const mod = await import('@codemirror/lang-json')
      return [mod.json()]
    }

    // HTML
    if (p.endsWith('.html') || p.endsWith('.htm')) {
      const mod = await import('@codemirror/lang-html')
      return [mod.html()]
    }

    // CSS
    if (p.endsWith('.css')) {
      const mod = await import('@codemirror/lang-css')
      return [mod.css()]
    }

    // JavaScript / TypeScript (+ JSX/TSX)
    if (p.endsWith('.js') || p.endsWith('.jsx') || p.endsWith('.ts') || p.endsWith('.tsx')) {
      const mod = await import('@codemirror/lang-javascript')
      return [mod.javascript({ jsx: p.endsWith('.jsx') || p.endsWith('.tsx'), typescript: p.endsWith('.ts') || p.endsWith('.tsx') })]
    }

    // Python
    if (p.endsWith('.py')) {
      const mod = await import('@codemirror/lang-python')
      return [mod.python()]
    }

    // C/C++
    if (p.endsWith('.c') || p.endsWith('.h') || p.endsWith('.cpp') || p.endsWith('.hpp') || p.endsWith('.cc')) {
      const mod = await import('@codemirror/lang-cpp')
      return [mod.cpp()]
    }

    // Go
    if (p.endsWith('.go')) {
      const mod = await import('@codemirror/lang-go')
      return [mod.go()]
    }

    // .env files (dotenv)
    if (fname === '.env' || fname.startsWith('.env')) {
      try {
        const mod = await import('@codemirror/legacy-modes/mode/properties')
        if (mod?.properties) return [StreamLanguage.define(mod.properties)]
      } catch {}
    }

    // Dockerfile (use legacy mode for accurate highlighting)
    if (fname === 'dockerfile') {
      try {
        const mod = await import('@codemirror/legacy-modes/mode/dockerfile')
        if (mod?.dockerFile) return [StreamLanguage.define(mod.dockerFile)]
      } catch {}
      try {
        const legacy = await import('@codemirror/legacy-modes/mode/shell')
        return [StreamLanguage.define(legacy.shell)]
      } catch {}
      return []
    }
  } catch (e) {
    // language package might not be installed yet; fallback silently
    console.debug('Language extension not available:', e?.message)
  }
  return []
}

function isProbablyBinary(str) {
  if (!str) return false
  const sample = str.slice(0, 2000)
  if (sample.indexOf('\u0000') !== -1 || sample.indexOf('\x00') !== -1) return true
  let bad = 0
  const total = sample.length || 1
  for (let i = 0; i < sample.length; i++) {
    const c = sample.charCodeAt(i)
    if (c === 9 || c === 10 || c === 13) continue // tab/newline
    if (c >= 32 && c <= 126) continue // ascii printable
    if (c >= 160 && c <= 0xfffd) continue // common unicode printable
    bad++
  }
  return bad / total > 0.3
}

function isEnvFile(path) {
  const fname = (path || '').split('/').pop() || ''
  const lower = fname.toLowerCase()
  return lower === '.env' || lower.startsWith('.env')
}

// Resolve conflict markers in text. mode: 'both' keeps both sides without markers.
function resolveConflictMarkers(text, mode) {
  if (!text) return text
  const lines = text.split(/\n/)
  let out = []
  let state = 'normal'
  let ours = []
  let theirs = []
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    if (state === 'normal') {
      if (line.startsWith('<<<<<<<')) { state = 'ours'; ours = []; theirs = []; continue }
      out.push(line)
      continue
    }
    if (state === 'ours') {
      if (line.startsWith('=======')) { state = 'theirs'; continue }
      ours.push(line)
      continue
    }
    if (state === 'theirs') {
      if (line.startsWith('>>>>>>>')) {
        // end block
        if (mode === 'both') {
          out.push(...ours, ...theirs)
        } else if (mode === 'theirs') {
          out.push(...theirs)
        } else { // 'ours'
          out.push(...ours)
        }
        state = 'normal'
        continue
      }
      theirs.push(line)
      continue
    }
  }
  return out.join('\n')
}

// Build a CodeMirror linter that flags relative volume host paths and "ports:" usage, with quick fixes
function makeComposeLinter({ absRoot, moduleSlug }) {
  const replacePrefix = (p) => {
    if (!p) return p
    const trimmed = p.replace(/^\.\/?/, '')
    if (!absRoot) return p.replace(/^\.\//, '/abs/path/') // noop-ish fallback to avoid empty
    const sep = absRoot.endsWith('/') ? '' : '/'
    return `${absRoot}${sep}${trimmed}`
  }
  return linter(view => {
    const text = view.state.doc.toString()
    const lines = text.split(/\n/)
    let diags = []
    let offset = 0
    let inVolumes = false
    let volumesIndent = 0
    let inServices = false
    let servicesIndent = 0
    let currentService = ''
    let currentServiceIndent = 0
    // Track top-level networks block for later diagnostics
    let topNetworksIdx = -1
    let topNetworksIndent = 0
    let topNetworksNames = []
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i]
      const indent = line.match(/^\s*/)[0].length
      // Track top-level networks block and collect simple names under it
      if (/^\s*networks\s*:\s*(#.*)?$/.test(line) && indent === 0) {
        topNetworksIdx = i
        topNetworksIndent = indent
        // collect network keys directly under this block
        let k = i + 1
        topNetworksNames = []
        while (k < lines.length) {
          const l = lines[k]
          const ind = l.match(/^\s*/)[0].length
          if (l.trim() !== '' && ind <= topNetworksIndent) break
          const m = l.match(/^\s*([A-Za-z0-9_.-]+)\s*:\s*$/)
          if (m) topNetworksNames.push(m[1])
          k++
        }
      }
      // Track services and current service name
      if (/^\s*services\s*:\s*(#.*)?$/.test(line)) {
        inServices = true
        servicesIndent = indent
        currentService = ''
        currentServiceIndent = 0
        offset += line.length + 1
        continue
      }
      if (inServices && indent <= servicesIndent && line.trim() !== '') {
        inServices = false
        currentService = ''
      }
      if (inServices) {
        // Service declaration like "  backend:"
        const mSvc = line.match(/^(\s*)([A-Za-z0-9_.-]+)\s*:\s*(#.*)?$/)
        if (mSvc && indent > servicesIndent && (currentService === '' || indent <= currentServiceIndent)) {
          currentService = mSvc[2]
          currentServiceIndent = indent
        }
      }
      // Track whether we are inside a volumes: block (top-level or service-level)
      if (/^\s*volumes\s*:\s*(#.*)?$/.test(line)) {
        inVolumes = true
        volumesIndent = indent
        offset += line.length + 1
        continue
      }
      if (inVolumes && indent <= volumesIndent && line.trim() !== '') {
        // left the volumes block
        inVolumes = false
      }

      if (inVolumes) {
        // Pattern 1: list item form: - ./host:/container[:...]
        const m = line.match(/^\s*-\s*([^:#\s][^:]*?)\s*:\s*[^\s#]+/)
        if (m) {
          const host = m[1]
          if (/^(\.\.?\/)/.test(host)) {
            const startCol = line.indexOf(host)
            const from = offset + startCol
            const to = from + host.length
            const replacement = replacePrefix(host)
            diags.push({
              from,
              to,
              severity: 'warning',
              message: 'Relative path for volumes are not working correctly. Use absolute path instead',
              actions: [{
                name: 'Replace with absolute path',
                apply: (v) => v.dispatch({ changes: { from, to, insert: replacement } })
              }]
            })
          }
        }
        // Pattern 2: object form source: ./host
        const m2 = line.match(/^\s*source\s*:\s*([^\s#]+)\s*(#.*)?$/)
        if (m2) {
          const host = m2[1]
          if (/^(\.\.?\/)/.test(host)) {
            const startCol = line.indexOf(host)
            const from = offset + startCol
            const to = from + host.length
            const replacement = replacePrefix(host)
            diags.push({
              from,
              to,
              severity: 'warning',
              message: 'Relative path for volumes are not working correctly. Use absolute path instead',
              actions: [{
                name: 'Replace with absolute path',
                apply: (v) => v.dispatch({ changes: { from, to, insert: replacement } })
              }]
            })
          }
        }
      }

      // Lint for ports under a service
      if (inServices && currentService && /^\s*ports\s*:\s*(#.*)?$/.test(line) && indent > currentServiceIndent) {
        const from = offset + line.indexOf('ports')
        const to = from + 'ports'.length
        const serviceName = currentService
        const portsIndent = indent
        // Collect list item lines following
        let j = i + 1
        const portLines = [] // {i, line}
        while (j < lines.length) {
          const l = lines[j]
          const ind = l.match(/^\s*/)[0].length
          if (l.trim() === '') { portLines.push({ i: j, line: l }); j++; continue }
          if (ind <= portsIndent) break
          // list item or mapping line
          if (/^\s*-\s*.+/.test(l)) portLines.push({ i: j, line: l })
          else if (/^\s*[A-Za-z0-9_.-]+\s*:/.test(l)) break
          else portLines.push({ i: j, line: l })
          j++
        }

        diags.push({
          from,
          to,
          severity: 'warning',
          message: 'Please use Expose and add pan-bagnat-net to the module instead of openning a port',
          actions: [{
            name: 'Convert to expose + network',
            apply: (v) => {
              const doc = v.state.doc.toString()
              const all = doc.split(/\n/)
              // 1) replace ports: with expose:
              const lineText = all[i]
              const lineStart = doc.split(/\n/, i).join('\n').length + (i>0?1:0)
              const portsKeyIndex = lineText.indexOf('ports')
              const changes = []
              changes.push({ from: lineStart + portsKeyIndex, to: lineStart + portsKeyIndex + 5, insert: 'expose' })
              // 2) replace list items host:container -> container
              const reHostMap = /^(\s*-\s*)("?)(\d+)(?:[^:\n]*?):(\d+)(?:\/(tcp|udp))?("?)(.*)$/
              for (const pl of portLines) {
                const lt = all[pl.i]
                const m = lt.match(reHostMap)
                if (m) {
                  const before = m[1]
                  const quoteL = m[2]
                  const cont = m[4]
                  const proto = m[5] ? '/' + m[5] : ''
                  const quoteR = m[6]
                  const after = m[7] || ''
                  const replacement = `${before}${quoteL}${cont}${proto}${quoteR}${after}`
                  const ls = doc.split(/\n/, pl.i).join('\n').length + (pl.i>0?1:0)
                  changes.push({ from: ls, to: ls + lt.length, insert: replacement })
                }
              }
              // 3) Ensure top-level networks include pan-bagnat-net external: true
              const hasTopNetworksIdx = all.findIndex(l => /^networks\s*:\s*(#.*)?$/.test(l))
              let needAppendTop = false
              let insertPosDocEnd = doc.length
              if (hasTopNetworksIdx === -1) {
                needAppendTop = true
              } else {
                // Check if pan-bagnat-net exists under networks
                let k = hasTopNetworksIdx + 1
                let found = false
                while (k < all.length && (all[k].trim() === '' || all[k].match(/^\s+/))) {
                  if (/^\s*pan-bagnat-net\s*:\s*$/.test(all[k])) { found = true; break }
                  k++
                }
                if (!found) needAppendTop = true
              }
              if (needAppendTop) {
                const block = (all.length>0 && all[all.length-1].trim()!==''? '\n' : '') +
                  'networks:\n' +
                  '  pan-bagnat-net:\n' +
                  '    external: true\n'
                changes.push({ from: insertPosDocEnd, to: insertPosDocEnd, insert: block })
              }
              // 4) Add service networks with alias
              // Find service block start and end
              let sStart = i
              // move up to service name line
              while (sStart > 0 && !new RegExp(`^\\s*${serviceName}\\s*:`).test(all[sStart])) sStart--
              let sIndent = all[sStart].match(/^\s*/)[0].length
              // Determine if networks already present in service
              let sEnd = sStart + 1
              while (sEnd < all.length) {
                const ind = all[sEnd].match(/^\s*/)[0].length
                if (all[sEnd].trim()!=='' && ind <= sIndent) break
                sEnd++
              }
              const hasNetworks = all.slice(sStart+1, sEnd).some(l => l.match(/^\s*networks\s*:/))
              const alias = `${moduleSlug || 'module'}-${serviceName}`
              if (!hasNetworks) {
                const insertAfterLine = j-1 >= sStart ? j-1 : sStart
                const insertAfterPos = doc.split(/\n/, insertAfterLine).join('\n').length + (insertAfterLine>0?1:0) + all[insertAfterLine].length
                const indentStr = ' '.repeat(sIndent + 2)
                const netBlock = `\n${indentStr}networks:\n${indentStr}  pan-bagnat-net:\n${indentStr}    aliases:\n${indentStr}      - ${alias}`
                changes.push({ from: insertAfterPos, to: insertAfterPos, insert: netBlock })
              } else {
                // ensure network entry exists, else append minimal entry under networks
                let nIdx = -1
                for (let t = sStart+1; t < sEnd; t++) if (/^\s*networks\s*:\s*$/.test(all[t])) { nIdx = t; break }
                if (nIdx !== -1) {
                  let nIndent = all[nIdx].match(/^\s*/)[0].length
                  let t = nIdx + 1
                  let hasNet = false
                  let pbnIdx = -1
                  let hasAliases = false
                  while (t < sEnd) {
                    const ind = all[t].match(/^\s*/)[0].length
                    if (all[t].trim()!=='' && ind <= nIndent) break
                    if (/^\s*pan-bagnat-net\s*:\s*$/.test(all[t])) { hasNet = true; pbnIdx = t }
                    if (pbnIdx !== -1 && /^\s*aliases\s*:\s*$/.test(all[t])) { hasAliases = true }
                    t++
                  }
                  if (!hasNet) {
                    const after = doc.split(/\n/, nIdx+1).join('\n').length + (nIdx+1>0?1:0)
                    const indentStr = ' '.repeat(nIndent + 2)
                    const block = `${indentStr}pan-bagnat-net:\n${indentStr}  aliases:\n${indentStr}    - ${alias}\n`
                    changes.push({ from: after, to: after, insert: block })
                  } else if (!hasAliases && pbnIdx !== -1) {
                    const insertAt = doc.split(/\n/, pbnIdx+1).join('\n').length + (pbnIdx+1>0?1:0)
                    const indentStr = ' '.repeat((all[pbnIdx].match(/^\s*/)[0].length) + 2)
                    const block = `${indentStr}aliases:\n${indentStr}  - ${alias}\n`
                    changes.push({ from: insertAt, to: insertAt, insert: block })
                  }
                }
              }
              v.dispatch({ changes })
            }
          }]
        })
      }

      offset += line.length + 1
    }

    // Lint: only external network configured (pan-bagnat-net) → suggest adding module private network and attach to all services
    if (topNetworksIdx !== -1) {
      const onlyPbn = topNetworksNames.length === 1 && topNetworksNames[0] === 'pan-bagnat-net'
      if (onlyPbn) {
        const from = lines.slice(0, topNetworksIdx).join('\n').length + (topNetworksIdx>0?1:0)
        const to = from + 'networks'.length
        const moduleNet = `${moduleSlug || 'module'}-net`
        diags.push({
          from,
          to,
          severity: 'warning',
          message: `Only external network configured. Add a dedicated \"${moduleNet}\" network to isolate this module and attach all services to it.`,
          actions: [{
            name: 'Add module network and attach services',
            apply: (v) => {
              const doc = v.state.doc.toString()
              const all = doc.split(/\n/)
              const moduleNet = `${moduleSlug || 'module'}-net`
              const applyChanges = []
              // Ensure top-level networks has moduleNet definition
              let nIdx = -1
              for (let t = 0; t < all.length; t++) {
                if (/^\s*networks\s*:\s*(#.*)?$/.test(all[t]) && all[t].match(/^\s*/)[0].length === 0) { nIdx = t; break }
              }
              if (nIdx === -1) {
                const block = (all.length>0 && all[all.length-1].trim()!==''? '\n' : '') +
                  'networks:\n' +
                  `  ${moduleNet}:\n` +
                  `    name: ${moduleNet}\n` +
                  '    driver: bridge\n'
                applyChanges.push({ from: doc.length, to: doc.length, insert: block })
              } else {
                // check if moduleNet exists
                let exists = false
                let t = nIdx + 1
                while (t < all.length) {
                  const ind = all[t].match(/^\s*/)[0].length
                  if (all[t].trim() !== '' && ind === 0) break
                  if (new RegExp(`^\\s*${moduleNet}\\s*:`).test(all[t])) { exists = true; break }
                  t++
                }
                if (!exists) {
                  const insertAt = doc.split(/\n/, nIdx+1).join('\n').length + (nIdx+1>0?1:0)
                  const block = `  ${moduleNet}:\n    name: ${moduleNet}\n    driver: bridge\n`
                  applyChanges.push({ from: insertAt, to: insertAt, insert: block })
                }
              }
              // Attach moduleNet to all services
              // Find services block
              let sIdx = -1
              for (let t = 0; t < all.length; t++) { if (/^\s*services\s*:\s*$/.test(all[t])) { sIdx = t; break } }
              if (sIdx !== -1) {
                let t = sIdx + 1
                const sIndent = all[sIdx].match(/^\s*/)[0].length
                while (t < all.length) {
                  const l = all[t]
                  if (l.trim() !== '' && l.match(/^\s*/)[0].length <= sIndent) break
                  const svcMatch = l.match(/^(\s*)([A-Za-z0-9_.-]+)\s*:\s*$/)
                  if (svcMatch) {
                    const svcIndent = svcMatch[1].length
                    // find end of service block
                    let u = t + 1
                    while (u < all.length) {
                      const ind = all[u].match(/^\s*/)[0].length
                      if (all[u].trim() !== '' && ind <= svcIndent) break
                      u++
                    }
                    // search for networks block
                    let networksLine = -1
                    for (let k = t+1; k < u; k++) if (/^\s*networks\s*:\s*$/.test(all[k])) { networksLine = k; break }
                    if (networksLine === -1) {
                      // insert networks with moduleNet
                      const insertPos = doc.split(/\n/, u).join('\n').length + (u>0?1:0)
                      const indentStr = ' '.repeat(svcIndent + 2)
                      const block = `\n${indentStr}networks:\n${indentStr}  ${moduleNet}:\n`
                      applyChanges.push({ from: insertPos, to: insertPos, insert: block })
                    } else {
                      // ensure mapping entry exists
                      let hasModuleNet = false
                      let k = networksLine + 1
                      const nIndent = all[networksLine].match(/^\s*/)[0].length
                      while (k < u) {
                        const ind = all[k].match(/^\s*/)[0].length
                        if (all[k].trim() !== '' && ind <= nIndent) break
                        if (new RegExp(`^\\s*${moduleNet}\\s*:`).test(all[k])) { hasModuleNet = true; break }
                        k++
                      }
                      if (!hasModuleNet) {
                        const insertAt = doc.split(/\n/, networksLine+1).join('\n').length + (networksLine+1>0?1:0)
                        const indentStr = ' '.repeat(nIndent + 2)
                        applyChanges.push({ from: insertAt, to: insertAt, insert: `${indentStr}${moduleNet}:\n` })
                      }
                    }
                    t = u - 1
                  }
                  t++
                }
              }
              v.dispatch({ changes: applyChanges })
            }
          }]
        })
      }
    }
    return diags
  })
}

// Simple colorizer for .env: add spans for KEY and VALUE around '='
function dotenvColorizerExt() {
  const keyMark = Decoration.mark({ class: 'cm-dotenv-key' })
  const valMark = Decoration.mark({ class: 'cm-dotenv-value' })
  return ViewPlugin.fromClass(class {
    constructor(view) {
      this.decorations = this.build(view)
    }
    update(update) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = this.build(update.view)
      }
    }
    build(view) {
      const builder = []
      for (let { from, to } of view.visibleRanges) {
        let text = view.state.doc.sliceString(from, to)
        let pos = from
        const lines = text.split('\n')
        for (let i = 0; i < lines.length; i++) {
          const l = lines[i]
          if (!l || /^\s*#/.test(l)) { pos += l.length + 1; continue }
          const idx = l.indexOf('=')
          if (idx > 0) {
            builder.push(keyMark.range(pos, pos + idx))
            builder.push(valMark.range(pos + idx + 1, pos + l.length))
          }
          pos += l.length + 1
        }
      }
      return Decoration.set(builder, true)
    }
  }, { decorations: v => v.decorations })
}
