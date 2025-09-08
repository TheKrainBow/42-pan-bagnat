import React from 'react';
import './ArrayHeader.css';

export default function ArrayHeader({ title, value, onChange, onFilterClick }) {
  return (
    <div className="array-header">
      <div className="array-header-title">{title}</div>
      <div className="array-header-actions">
        <input className="array-header-input" value={value} onChange={onChange} placeholder="Filter..." />
        {onFilterClick && (
          <button className="array-header-filter" onClick={onFilterClick}>Filter</button>
        )}
      </div>
    </div>
  );
}
