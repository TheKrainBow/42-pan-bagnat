import { useState, useEffect } from 'react';
import { getReadableStyles } from '../utils/ColorUtils';
import './AppIcon.css';

export function RoleBadge({ hexColor, children }) {
  const styles = getReadableStyles(hexColor);
  return (
    <span className="role-badge" style={styles}>
      {children}
    </span>
  );
}