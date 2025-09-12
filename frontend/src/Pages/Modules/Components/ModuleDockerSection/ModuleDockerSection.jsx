import React, { useState } from 'react';
import Button from 'Global/Button/Button';
import DockerContainersSection from './DockerContainersSection/DockerContainersSection';
import DockerComposeSection from './DockerComposeSection/DockerComposeSection';
import ModulePageSection from './DockerPageSection/DockerPageSection';
import './ModuleDockerSection.css'

export default function ModuleDockerSection({ moduleId, dockerTab, setDockerTab, hideTabs = false }) {
  return (
    <div className="module-docker-section">
      {dockerTab !== 'ide' && !hideTabs && (
      <div className="docker-tabs">
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
      )}

      <div className="docker-tab-content">
        {dockerTab === 'ide' && <DockerComposeSection moduleId={moduleId} />}
        {dockerTab === 'containers' && <DockerContainersSection moduleId={moduleId} />}
        {dockerTab === 'pages' && <ModulePageSection moduleId={moduleId}/>}
      </div>
    </div>
  );
}
