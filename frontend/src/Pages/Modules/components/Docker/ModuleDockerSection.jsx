import React, { useState } from 'react';
import Button from 'Global/Button';
import DockerComposePage from './DockerComposeSection';
import DockerContainersSection from './DockerContainersSection';
import ModulePageSettings from './ModulePageSettings';
import './ModuleDockerSection.css'

export default function ModuleDockerSection({ moduleId }) {
  const [dockerTab, setDockerTab] = useState('compose');

  return (
    <div className="module-docker-section">
      <div className="docker-tabs">
        <Button
          label="Compose"
          className={`custom-btn ${dockerTab === 'compose' ? 'blue' : 'gray'}`}
          onClick={() => setDockerTab('compose')}
        />
        <Button
          label="Containers"
          className={`custom-btn ${dockerTab === 'containers' ? 'blue' : 'gray'}`}
          onClick={() => setDockerTab('containers')}
        />
        <Button
          label="Pages"
          className={`custom-btn ${dockerTab === 'pages' ? 'blue' : 'gray'}`}
          onClick={() => setDockerTab('pages')}
        />
      </div>

      <div className="docker-tab-content">
        {dockerTab === 'compose' && <DockerComposePage moduleId={moduleId} />}
        {dockerTab === 'containers' && <DockerContainersSection moduleId={moduleId} />}
        {dockerTab === 'pages' && <ModulePageSettings moduleId={moduleId}/>}
      </div>
    </div>
  );
}