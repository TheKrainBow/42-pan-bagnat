import React from 'react';
import { useNavigate } from 'react-router-dom';
import './Legal.css';

const LegalLayout = ({ title, updatedAt, children }) => {
  const navigate = useNavigate();
  return (
    <div className="legal-page">
      <div className="legal-back-link" onClick={() => navigate(-1)}>← Back</div>
      <h1>{title}</h1>
      <p className="legal-updated">Last updated {updatedAt}</p>
      <div className="legal-draft-banner">
        This is a draft written to describe Pan Bagnat's actual behavior (including module activity tracking). It has not
        been reviewed by a lawyer — please have it validated before relying on it for compliance purposes.
      </div>
      {children}
      <div className="legal-links">
        <a onClick={() => navigate('/legal/terms')}>Terms of Use</a>
        <a onClick={() => navigate('/legal/privacy')}>Privacy Policy</a>
      </div>
    </div>
  );
};

export default LegalLayout;
