// components/DockerComposeEditor.jsx
import React, { useState, useEffect, useRef } from 'react'
import CodeMirror from '@uiw/react-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import './DockerComposeEditor.css' // Keep CSS if styles are shared
import Button from 'Global/Button'

export default function DockerComposeEditor({ moduleId }) {
  const [configYaml, setConfigYaml]     = useState('')
  const [isSaving, setIsSaving]         = useState(false)
  const [isDeploying, setIsDeploying]   = useState(false)
  const fetchedRef = useRef(false)

  useEffect(() => {
    if (fetchedRef.current) return
    fetchedRef.current = true

    fetch(`http://localhost:8080/api/v1/modules/${moduleId}/config`)
      .then(r => r.json())
      .then(d => setConfigYaml(d.config))
      .catch(() => setConfigYaml('# Error loading docker-compose-panbagnat.yml'))
  }, [moduleId])

  const handleDeploy = async () => {
    setIsDeploying(true)
    try {
      await fetch(`http://localhost:8080/api/v1/modules/${moduleId}/deploy`, {
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
      <div className="editors-row">
        <div className="editor-pane">
          <div className="pane-header">
            docker-compose-panbagnat.yml {isSaving && '· saving…'}
          </div>
          <CodeMirror
            value={configYaml}
            height="800px"
            extensions={[yaml()]}
            theme="dark"
            onChange={setConfigYaml}
            basicSetup={{ lineNumbers: true }}
          />
        </div>
      </div>

      <div className="deploy-row">
        <Button
          label={isDeploying ? 'Deploying...' : 'Save & Deploy'}
          color="blue"
          onClick={handleDeploy}
          disabled={isDeploying}
        />
      </div>
    </div>
  )
}
