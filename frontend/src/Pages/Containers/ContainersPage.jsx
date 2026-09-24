import React from 'react';
import ContainersGraph from 'Pages/Modules/Components/ModuleDockerSection/ContainersGraph/ContainersGraph';
import './ContainersPage.css';

export default function ContainersPage() {
  return (
    <div className="containers-page">
      <h2 className="containers-page-title">Containers</h2>
      <ContainersGraph />
    </div>
  );
}
