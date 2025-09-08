import React, { useState, useEffect, useRef } from 'react'
import CodeMirror from '@uiw/react-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import './DockerComposeSection.css' // Keep CSS if styles are shared
import Button from 'ui/atoms/Button/Button'
import { fetchWithAuth } from 'Global/utils/Auth';

export default function DockerComposeSection({ moduleId }) {
  const [configYaml, setConfigYaml]     = useState('')
  const [isDeploying, setIsDeploying]   = useState(false)
  const fetchedRef = useRef(false)

  useEffect(() => {
    if (fetchedRef.current) return
    fetchedRef.current = true

    fetchWithAuth(`/api/v1/admin/modules/${moduleId}/docker/config`)
      .then(r => r.json())
      .then(d => setConfigYaml(d.config))
      .catch(() => setConfigYaml('# Error loading docker-compose-panbagnat.yml'))
  }, [moduleId])

  const handleDeploy = async () => {
    setIsDeploying(true)
    try {
      await fetchWithAuth(`/api/v1/admin/modules/${moduleId}/docker/deploy`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ config: configYaml }),
      })
      // WebSocket will handle deploy feedback
    } catch (e) {
      console.error('Deploy request failed:', e)
      toast.error('❌ Failed to start deployment')
    } finally {
      setIsDeploying(false)
    }
  }

  return (
  <div className="config-panel">
    <div className="editor-pane">
      <div className="pane-header">
        <span>docker-compose-panbagnat.yml</span>
        <div className="header-actions">
          <Button
            label={isDeploying ? 'Deploying...' : 'Save & Deploy'}
            color="blue"
            onClick={handleDeploy}
            disabled={isDeploying}
          />
        </div>
      </div>

      <div className="codemirror-wrapper">
        <CodeMirror
          className="codemirror-wrapper"
          value={configYaml}
          extensions={[yaml()]}
          height="auto"
          theme="dark"
          onChange={setConfigYaml}
          basicSetup={{ lineNumbers: true }}
        />
      </div>
    </div>
  </div>
  )
}
