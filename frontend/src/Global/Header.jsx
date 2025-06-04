import React from 'react';
import './Header.css';

const Header = ({ title, value, onChange }) => {
  return (
    <div className="header-bar">
      <h2>{title}</h2>
      <div className="search-container">
        <img src="/icons/search.png" alt="Search" className="search-icon-inside" />
        <input
          className="search with-icon"
          type="text"
          placeholder="Search..."
          value={value}
          onChange={onChange}
        />
      </div>
    </div>
  );
};

export default Header;
