import React from 'react';
import './AppIcon.css';

export default function AppIcon({ app, fallback }) {
  const src = app?.icon_url || fallback;
  const alt = app?.name || 'app';
  return (
    <img src={src} alt={alt} className="app-icon" />
  );
}
