import React, { useState } from 'react';
import Button from 'ui/atoms/Button/Button';
import DockerComposePage from './DockerComposeSection/DockerComposeSection';
import DockerContainersSection from './DockerContainersSection/DockerContainersSection';
import ModulePageSection from './DockerPageSection/DockerPageSection';
import './ModuleDockerSection.css'

export default function ModuleDockerSection({ moduleId, dockerTab, setDockerTab }) {
  return (
    <div className="module-docker-section">
      <div className="docker-tabs">
        <Button
          label="Compose"
          color={`${dockerTab === 'compose' ? 'blue' : 'gray'}`}
          onClick={() => setDockerTab('compose')}
        />
        <Button
          label="Containers"
          color={`${dockerTab === 'containers' ? 'blue' : 'gray'}`}
          onClick={() => setDockerTab('containers')}
          disabled={false}
          disabledMessage={"You must compose your module first"}
        />
        <Button
          label="Pages"
          color={`${dockerTab === 'pages' ? 'blue' : 'gray'}`}
          onClick={() => setDockerTab('pages')}
          disabled={false}
          disabledMessage={"You must compose your module first"}
        />
      </div>

      <div className="docker-tab-content">
        {dockerTab === 'compose' && <DockerComposePage moduleId={moduleId} />}
        {dockerTab === 'containers' && <DockerContainersSection moduleId={moduleId} />}
        {dockerTab === 'pages' && <ModulePageSection moduleId={moduleId}/>}
      </div>
    </div>
  );
}
