import React from 'react';
import './Header.css';
import Button from 'Global/Button';


const Header = ({
  title,
  value,
  onChange,
  filterButtonLabel,      // optional filter button text
  onFilterClick,          // optional filter click handler
  actionButtonLabel,      // optional action button text
  onActionButtonClick,    // optional action click handler
}) => {
  return (
    <div className="header-bar">
      {/* 1) TITLE (always on its own line) */}
      <h2 className="header-title">{title}</h2>

      {/* 2) SECOND ROW: search+filter on left, action on right */}
      <div className="header-controls">
        <div className="search-filter-wrapper">
          <div className="search-container">
            <img
              src="/icons/search.png"
              alt="Search"
              className="search-icon-inside"
            />
            <input
              className="search with-icon"
              type="text"
              placeholder="Search..."
              value={value}
              onChange={onChange}
            />
          </div>

          {onFilterClick && (
            <Button
              label={filterButtonLabel}
              icon="/icons/filter.png"
              color="dark-gray"
              onClick={onFilterClick}
            />
          )}
        </div>

        {actionButtonLabel && onActionButtonClick && (
          <Button
            label={actionButtonLabel}
            color="blue"
            onClick={onActionButtonClick}
          />
        )}
      </div>
    </div>
  );
};

export default Header;
