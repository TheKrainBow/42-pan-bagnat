// src/components/ModulePage.jsx
import React from 'react';
import './ModulePage.css'

export default function ModulePage({ moduleName }) {
  if (!moduleName) {
    return (
      <div className="module-page-placeholder">
        <p>Select a module from the sidebar to get started.</p>
      </div>
    );
  }

  return (
    <div className="module-page-container">
      <iframe
        src={`/module-page/${moduleName}`}
        title={moduleName}
        frameBorder="0"
        className="module-iframe"
      />
    </div>
  );
}
