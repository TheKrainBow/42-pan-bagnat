// components/ModuleConfigPanel.jsx
import React, { useState, useEffect, useRef } from 'react'
import CodeMirror from '@uiw/react-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import './ModuleConfigPanel.css'
import Button from 'Global/Button';

export default function ModuleConfigPanel({ moduleId }) {
  const [moduleYaml, setModuleYaml]     = useState('');
  const [composeYaml, setComposeYaml]   = useState('');
  const [isSaving, setIsSaving]         = useState(false);
  const [isGenerating, setIsGenerating] = useState(false);
  const [isDeploying, setIsDeploying]   = useState(false);
  const saveTimer = useRef();
  const fetchedRef = useRef(false);

  useEffect(() => {
    if (fetchedRef.current) return;
    fetchedRef.current = true;
    fetch(`http://localhost:8080/api/v1/modules/${moduleId}/config`)
      .then(r => r.json())
      .then(d => setModuleYaml(d.config))
      .catch(() => setModuleYaml('# Error loading module.yml'))
  }, [moduleId])

  useEffect(() => {
    clearTimeout(saveTimer.current)
    saveTimer.current = setTimeout(async () => {
      setIsGenerating(true)
      try {
        const res = await fetch(`http://localhost:8080/api/v1/modules/${moduleId}/compose`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ config: moduleYaml }),
        })
        if (!res.ok) throw new Error(res.statusText)
        const { compose } = await res.json()
        setComposeYaml(compose)
      } catch (e) {
        console.error('Failed to generate compose.yml', e)
        setComposeYaml('# Error generating docker-compose.yml')
      } finally {
        setIsGenerating(false)
      }
    }, 10)

    return () => clearTimeout(saveTimer.current)
  }, [moduleYaml, moduleId])

  const handleDeploy = async () => {
    setIsDeploying(true);
    try {
      await fetch(`http://localhost:8080/api/v1/modules/${moduleId}/deploy`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ config: moduleYaml }),
      });
      // No alerts — wait for WS notification
    } catch (e) {
      console.error('Deploy request failed to send:', e);
      // You can optionally toast here if it's a *network* error
      toast.error('❌ Failed to start deployment');
    } finally {
      setIsDeploying(false);
    }
  };

  return (
    <div className="config-panel">
      <div className="editors-row">
        <div className="editor-pane">
          <div className="pane-header">
            module.yml {isSaving && '· saving…'}
          </div>
          <CodeMirror
            value={moduleYaml}
            height="800px"
            extensions={[yaml()]}
            theme="dark"
            onChange={setModuleYaml}
            basicSetup={{ lineNumbers: true }}
          />
        </div>

        <div className="editor-pane">
          <div className="pane-header">
            docker-compose.yml {isGenerating && '· generating…'}
          </div>
          <CodeMirror
            value={composeYaml}
            height="800px"
            extensions={[yaml()]}
            theme="dark"
            editable={false}
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
