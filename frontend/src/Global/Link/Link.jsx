import { useState } from 'react';
import './Link.css';

const Link = ({ url, shorten = false }) => {
  const [copied, setCopied] = useState(false);

  if (!url) return null;

  const handleClick = () => {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(sshKey)
        .then(() => {
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        })
        .catch((err) => {
          console.error("Copy failed", err);
        });
    } else {
      console.warn("Clipboard API not supported");
    }
  };


  let visibleText = url;
  if (typeof shorten === 'number' && shorten < url.length) {
    const half = Math.floor((shorten - 3) / 2);
    visibleText = `${url.slice(0, half)}...${url.slice(-half)}`;
  }

  return (
    <span className="copyable-link-wrapper" onClick={handleClick}>
      <span className="copyable-link" title={copied ? 'Copied!' : 'Click to copy'}>
        {visibleText}
      </span>
      {copied && <span className="copied-tooltip">Copied!</span>}
    </span>
  );
};

export default Link;